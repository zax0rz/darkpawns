package portable

// Network settings in the admin console (docs/design/admin-network-settings.md):
// the launcher owns instance.json, so the console proposes typed changes and
// the launcher validates and saves only the network fields, preserving the
// signing secret, environment entries and every other field. Saving never
// touches a live socket; a separate apply transaction restarts the world and
// restores the previous configuration if a requested listener fails to bind.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

// ErrConfigConflict is a stale-revision refusal, mirroring the game settings
// store: reload before proposing again.
var ErrConfigConflict = errors.New("configuration changed; reload before saving")

const (
	loopbackBind = "127.0.0.1"
	publicBind   = ""
)

// ListenerProposal is one listener as the console sends it: enabled, its
// port, and whether it binds every interface (public) or this computer only.
type ListenerProposal struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
	Public  bool `json:"public"`
}

// NetworkProposal carries the three editable listeners plus the revision the
// operator loaded. HTTP is not part of it: the console keeps HTTP read-only
// so the management route itself never moves under the operator.
type NetworkProposal struct {
	Revision  int64            `json:"revision"`
	Telnet    ListenerProposal `json:"telnet"`
	TelnetTLS ListenerProposal `json:"telnet_tls"`
	Whod      ListenerProposal `json:"whod"`
}

// ListenerView is one listener as the console renders it. Scope is
// "loopback", "public" or "address" (an explicit bind set outside the
// console, shown read-only rather than silently rewritten).
type ListenerView struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	Scope   string `json:"scope"`
	Address string `json:"address,omitempty"`
}

// NetworkConfig is the launcher's answer to a read: current values, the
// revision for the next proposal, and whether a TLS certificate is
// configured (TLS cannot be enabled without one).
type NetworkConfig struct {
	Revision                 int64        `json:"revision"`
	HTTPPort                 int          `json:"http_port"`
	TLSCertificateConfigured bool         `json:"tls_certificate_configured"`
	Telnet                   ListenerView `json:"telnet"`
	TelnetTLS                ListenerView `json:"telnet_tls"`
	Whod                     ListenerView `json:"whod"`
}

// NetworkDiffEntry is one old-to-new exposure change.
type NetworkDiffEntry struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// SaveNetworkResult reports an accepted proposal.
type SaveNetworkResult struct {
	Config          NetworkConfig      `json:"config"`
	Diff            []NetworkDiffEntry `json:"diff"`
	Warnings        []string           `json:"warnings"`
	RestartRequired bool               `json:"restart_required"`
}

func bindScope(bind string) (scope, address string) {
	switch bind {
	case publicBind:
		return "public", ""
	case loopbackBind:
		return "loopback", ""
	default:
		return "address", bind
	}
}

func proposalBind(public bool) string {
	if public {
		return publicBind
	}
	return loopbackBind
}

// currentNetwork maps a Config into the console view.
func currentNetwork(cfg Config) NetworkConfig {
	view := func(enabled bool, port int, bind string) ListenerView {
		scope, address := bindScope(bind)
		return ListenerView{Enabled: enabled, Port: port, Scope: scope, Address: address}
	}
	return NetworkConfig{
		Revision:                 cfg.ConfigRevision,
		HTTPPort:                 cfg.HTTPPort,
		TLSCertificateConfigured: cfg.ServerEnv["TELNET_TLS_CERT_FILE"] != "" && cfg.ServerEnv["TELNET_TLS_KEY_FILE"] != "",
		Telnet:                   view(cfg.TelnetPort > 0, cfg.TelnetPort, cfg.TelnetBind),
		TelnetTLS:                view(cfg.TelnetTLSPort > 0, cfg.TelnetTLSPort, cfg.TelnetTLSBind),
		Whod:                     view(cfg.WhodPort > 0, cfg.WhodPort, cfg.WhodBind),
	}
}

// applyNetwork validates a proposal against the loaded configuration. Hard
// failures (range, collisions, TLS without a certificate) refuse the save;
// warnings (privileged ports, public plaintext telnet) save anyway and are
// reported for the operator to read before applying.
func applyNetwork(cfg Config, proposal NetworkProposal) (next Config, diff []NetworkDiffEntry, warnings []string, err error) {
	listeners := []struct {
		name     string
		p        ListenerProposal
		needsTLS bool
	}{
		{"telnet", proposal.Telnet, false},
		{"telnet_tls", proposal.TelnetTLS, true},
		{"whod", proposal.Whod, false},
	}
	ports := map[int]string{cfg.HTTPPort: "http"}
	for _, l := range listeners {
		if !l.p.Enabled {
			continue
		}
		if l.p.Port < 1 || l.p.Port > 65535 {
			return next, nil, nil, fmt.Errorf("%s port must be between 1 and 65535, not %d", l.name, l.p.Port)
		}
		if other, taken := ports[l.p.Port]; taken {
			return next, nil, nil, fmt.Errorf("%s port %d is already used by %s; each listener needs its own port", l.name, l.p.Port, other)
		}
		ports[l.p.Port] = l.name
		if l.p.Port < 1024 {
			warnings = append(warnings, fmt.Sprintf("The %s port %d is below 1024. Those ports are reserved on most systems, and the server can only bind one if the operating system gives it permission. If the bind is refused, the previous settings come back automatically.", l.name, l.p.Port))
		}
	}
	if proposal.Telnet.Enabled && proposal.Telnet.Public {
		warnings = append(warnings, "Public telnet without TLS sends passwords across the network in plain text. Anyone between the server and a player can read them. Prefer TLS for remote players.")
	}
	if !proposal.Telnet.Enabled && !proposal.TelnetTLS.Enabled {
		warnings = append(warnings, "No telnet listener will remain. Players can only connect through the web client until a listener is turned back on.")
	}
	if proposal.TelnetTLS.Enabled && cfg.ServerEnv["TELNET_TLS_CERT_FILE"] == "" {
		return next, nil, nil, errors.New("TLS telnet needs a certificate configured on the server first. Ask the operator to set TELNET_TLS_CERT_FILE and TELNET_TLS_KEY_FILE")
	}

	entries := []struct {
		field   string
		enabled bool
		port    int
		public  bool
	}{
		{"telnet", proposal.Telnet.Enabled, proposal.Telnet.Port, proposal.Telnet.Public},
		{"telnet_tls", proposal.TelnetTLS.Enabled, proposal.TelnetTLS.Port, proposal.TelnetTLS.Public},
		{"whod", proposal.Whod.Enabled, proposal.Whod.Port, proposal.Whod.Public},
	}
	diff = []NetworkDiffEntry{}
	for _, e := range entries {
		from, to := diffValues(cfg, e.field, e)
		if from != to {
			diff = append(diff, NetworkDiffEntry{Field: e.field, From: from, To: to})
		}
	}
	next = cfg
	next.TelnetPort, next.TelnetBind = listenerConfig(proposal.Telnet)
	next.TelnetTLSPort, next.TelnetTLSBind = listenerConfig(proposal.TelnetTLS)
	next.WhodPort, next.WhodBind = listenerConfig(proposal.Whod)
	return next, diff, warnings, nil
}

func listenerConfig(p ListenerProposal) (port int, bind string) {
	if !p.Enabled {
		return 0, ""
	}
	return p.Port, proposalBind(p.Public)
}

func diffValues(cfg Config, field string, e struct {
	field   string
	enabled bool
	port    int
	public  bool
},
) (from, to string) {
	current := struct {
		enabled bool
		port    int
		bind    string
	}{}
	switch field {
	case "telnet":
		current = struct {
			enabled bool
			port    int
			bind    string
		}{cfg.TelnetPort > 0, cfg.TelnetPort, cfg.TelnetBind}
	case "telnet_tls":
		current = struct {
			enabled bool
			port    int
			bind    string
		}{cfg.TelnetTLSPort > 0, cfg.TelnetTLSPort, cfg.TelnetTLSBind}
	case "whod":
		current = struct {
			enabled bool
			port    int
			bind    string
		}{cfg.WhodPort > 0, cfg.WhodPort, cfg.WhodBind}
	}
	describe := func(enabled bool, port int, bind string) string {
		if !enabled {
			return "off"
		}
		scope, address := bindScope(bind)
		if scope == "address" {
			return fmt.Sprintf("port %d on %s", port, address)
		}
		if scope == "public" {
			return fmt.Sprintf("port %d, public", port)
		}
		return fmt.Sprintf("port %d, this computer", port)
	}
	return describe(current.enabled, current.port, current.bind), describe(e.enabled, e.port, proposalBind(e.public))
}

// ReadNetworkConfig loads the current view for the console.
func ReadNetworkConfig(instance string) (NetworkConfig, error) {
	cfg, err := Load(instance)
	if err != nil {
		return NetworkConfig{}, err
	}
	return currentNetwork(cfg), nil
}

// SaveNetworkConfig validates and atomically saves only the network fields,
// keeping a snapshot of the previous configuration for the apply
// transaction's rollback. The live world is untouched: instance.json is read
// at start, so the change becomes real on the next restart.
func SaveNetworkConfig(instance string, proposal NetworkProposal) (SaveNetworkResult, error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return SaveNetworkResult{}, err
	}
	cfg, err := Load(instance)
	if err != nil {
		return SaveNetworkResult{}, err
	}
	if proposal.Revision != cfg.ConfigRevision {
		return SaveNetworkResult{}, ErrConfigConflict
	}
	next, diff, warnings, err := applyNetwork(cfg, proposal)
	if err != nil {
		return SaveNetworkResult{}, err
	}
	next.ConfigRevision = cfg.ConfigRevision + 1
	// The rollback snapshot is the configuration the running world was
	// started with: byte-identical, written before the new one lands.
	previous, err := os.ReadFile(filepath.Join(instance, "instance.json"))
	if err != nil {
		return SaveNetworkResult{}, err
	}
	if err := writePrivate(filepath.Join(instance, configRollbackFile), previous); err != nil {
		return SaveNetworkResult{}, err
	}
	if err := writeConfigAtomic(instance, next); err != nil {
		return SaveNetworkResult{}, err
	}
	if diff == nil {
		diff = []NetworkDiffEntry{}
	}
	if warnings == nil {
		warnings = []string{}
	}
	return SaveNetworkResult{
		Config:          currentNetwork(next),
		Diff:            diff,
		Warnings:        warnings,
		RestartRequired: len(diff) > 0,
	}, nil
}

// listenerTargets lists the enabled listeners of a configuration with the
// address the apply transaction dials to prove the bind: a public bind
// answers on loopback too, so 127.0.0.1 verifies both scopes.
func listenerTargets(cfg Config) []struct {
	Name    string
	Address string
} {
	var targets []struct {
		Name    string
		Address string
	}
	add := func(name string, port int, bind string) {
		if port <= 0 {
			return
		}
		host := bind
		if bind == publicBind {
			host = loopbackBind
		}
		targets = append(targets, struct {
			Name    string
			Address string
		}{name, net.JoinHostPort(host, strconv.Itoa(port))})
	}
	add("telnet", cfg.TelnetPort, cfg.TelnetBind)
	add("telnet_tls", cfg.TelnetTLSPort, cfg.TelnetTLSBind)
	add("whod", cfg.WhodPort, cfg.WhodBind)
	return targets
}

const configRollbackFile = ".config-rollback.json"

// writeConfigAtomic saves the whole configuration atomically: a crash
// mid-write can never leave a half-saved instance.json behind.
func writeConfigAtomic(instance string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return updateBytes(filepath.Join(instance, "instance.json"), append(data, '\n'))
}

// restoreRollbackConfig puts the snapshot written at proposal time back as
// the live configuration, byte-identical to what ran before the failed
// change. A missing snapshot means nothing was ever proposed; the caller
// treats that as "nothing to restore".
func restoreRollbackConfig(instance string) (restored bool, err error) {
	path := filepath.Join(instance, configRollbackFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Prove the snapshot is a configuration before installing it.
	if _, err := Load(instance); err != nil {
		return false, err
	}
	var check Config
	if err := json.Unmarshal(data, &check); err != nil || check.Format != Format {
		return false, errors.New("rollback snapshot is not a valid configuration; it is retained for manual recovery")
	}
	if err := updateBytes(filepath.Join(instance, "instance.json"), data); err != nil {
		return false, err
	}
	return true, nil
}

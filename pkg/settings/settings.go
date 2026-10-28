// Package settings owns the game process's operator settings file
// (lib/data/settings.json). The launcher passes its path via DP_SETTINGS_PATH;
// a checkout deployment defaults beside the world data. Environment variables
// remain a compatibility override for checkout deployments and are surfaced,
// never silently ignored: a portable instance scrubs them, so the file is the
// single authority there.
//
// The file is revisioned like data/notifications.json: saves compare the
// caller's revision and conflict instead of last-writer-wins. Secrets are
// sealed with the same AES-256-GCM scheme (a mode-0600 key beside the file)
// and never appear in any View, export, or audit detail.
package settings

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Format is the settings file schema version. Readers reject other formats;
// game and file ship in the same release, so strictness cannot strand an
// operator the way a launcher-side field could.
const Format = 1

// Group names address one card each in the console and one PUT target each.
const (
	GroupSessions  = "sessions"
	GroupAccess    = "access"
	GroupGrapevine = "grapevine"
	GroupStaffPush = "staff_push"
	GroupContact   = "contact"
)

// Secret field names, stable across the file, API and audit trail.
const (
	SecretGrapevineClient  = "grapevine_client_secret"
	SecretStaffPushToken   = "staff_push_token"
	SecretContactSMTP      = "contact_smtp_password" // #nosec G101 -- field identifier, not a credential value
	SecretContactTurnstile = "contact_turnstile_secret"
)

// Sessions holds connection and login limits. Every field applies live: the
// packages that enforce them read atomics that Save swaps.
type Sessions struct {
	LoginIdleTimeoutSeconds     int `json:"login_idle_timeout_seconds" doc:"Pre-auth idle drop for telnet and WebSocket (DP-912)"`
	TelnetMaxConns              int `json:"telnet_max_conns" doc:"Total simultaneous telnet connections"`
	TelnetMaxConnsPerIP         int `json:"telnet_max_conns_per_ip" doc:"Simultaneous telnet connections from one address"`
	WebSocketMaxConnsPerIP      int `json:"websocket_max_conns_per_ip" doc:"Simultaneous WebSocket sessions from one address"`
	WebSocketMaxTotalConns      int `json:"websocket_max_total_conns" doc:"Process-wide WebSocket session cap"`
	AccountFailureBudget        int `json:"account_failure_budget" doc:"Wrong-password attempts per account across IPs before refusal; 0 disables"`
	AccountFailureWindowMinutes int `json:"account_failure_window_minutes" doc:"Window for the account failure budget"`
}

// Access holds proxy trust and browser origins. CORS applies live; trusted
// proxies are read once at boot, so changing them marks a restart.
type Access struct {
	TrustedProxies     []string `json:"trusted_proxies" doc:"CIDRs whose X-Forwarded-For names the client; empty means the loopback default (restart required)"`
	CORSAllowedOrigins []string `json:"cors_allowed_origins" doc:"Browser origins allowed across the game API; empty means the built-in defaults"`
	AdminCORSOrigin    string   `json:"admin_cors_origin" doc:"Origin allowed to call the admin API; empty means the built-in default"`
}

// Grapevine holds the intermud relay. The relay connects at boot, so every
// field applies on restart. The client secret lives only in the sealed
// Secrets map; every view reports presence, never value.
type Grapevine struct {
	Mode     string `json:"mode" enum:"off,presence,full" doc:"Grapevine intermud mode (restart required)"`
	URL      string `json:"url" doc:"Relay endpoint (restart required)"`
	ClientID string `json:"client_id" doc:"Relay client id (restart required)"`
}

// StaffPush holds the ntfy push for the immortal log. The pusher starts at
// boot, so every field applies on restart. The token lives only in the sealed
// Secrets map.
type StaffPush struct {
	URL           string `json:"url" doc:"ntfy topic URL; empty disables staff-log push (restart required)"`
	MinLevel      int    `json:"min_level" doc:"Minimum immortal level pushed (restart required)"`
	MinType       int    `json:"min_type" doc:"Minimum syslog type pushed (restart required)"`
	RatePerMinute int    `json:"rate_per_minute" doc:"Push rate cap (restart required)"`
}

// Contact holds the public contact form's mail path. The handler builds at
// boot; the form stays disabled until every field is present. Both secrets
// live only in the sealed Secrets map.
type Contact struct {
	To                 string   `json:"to" doc:"Recipient address (restart required)"`
	SMTPHost           string   `json:"smtp_host" doc:"SMTP host (restart required)"`
	SMTPPort           int      `json:"smtp_port" doc:"SMTP port (restart required)"`
	SMTPUser           string   `json:"smtp_user" doc:"SMTP user (restart required)"`
	TurnstileHostnames []string `json:"turnstile_hostnames" doc:"Hostnames the Turnstile widget may serve (restart required)"`
}

// Config is the whole file. Secrets is the sealed map, never serialized into
// any View.
type Config struct {
	Format    int               `json:"format"`
	Revision  int64             `json:"revision"`
	Sessions  Sessions          `json:"sessions"`
	Access    Access            `json:"access"`
	Grapevine Grapevine         `json:"grapevine"`
	StaffPush StaffPush         `json:"staff_push"`
	Contact   Contact           `json:"contact"`
	Secrets   map[string]string `json:"secrets,omitempty"`
}

// ErrConflict mirrors notify.ErrConflict's contract: reload before saving.
var ErrConflict = errors.New("settings changed; reload before saving")

// Defaults returns the compiled-in values every field starts from.
func Defaults() Config {
	return Config{
		Format: Format,
		Sessions: Sessions{
			LoginIdleTimeoutSeconds:     120,
			TelnetMaxConns:              200,
			TelnetMaxConnsPerIP:         8,
			WebSocketMaxConnsPerIP:      8,
			WebSocketMaxTotalConns:      500,
			AccountFailureBudget:        0,
			AccountFailureWindowMinutes: 60,
		},
		Access:    Access{},
		Grapevine: Grapevine{Mode: "off", URL: "wss://grapevine.haus/socket"},
		StaffPush: StaffPush{MinLevel: 31, MinType: 1, RatePerMinute: 30},
	}
}

// restartFields lists, per group, the fields whose change needs a restart.
// Anything absent applies live.
var restartFields = map[string][]string{
	GroupAccess:    {"trusted_proxies"},
	GroupGrapevine: {"mode", "url", "client_id", "client_secret"},
	GroupStaffPush: {"url", "token", "min_level", "min_type", "rate_per_minute"},
	GroupContact:   {"to", "smtp_host", "smtp_port", "smtp_user", "smtp_password", "turnstile_secret", "turnstile_hostnames"},
}

func validateSessions(s Sessions) error {
	ranges := []struct {
		name string
		v    int
		lo   int
		hi   int
	}{
		{"login_idle_timeout_seconds", s.LoginIdleTimeoutSeconds, 1, 3600},
		{"telnet_max_conns", s.TelnetMaxConns, 1, 100000},
		{"telnet_max_conns_per_ip", s.TelnetMaxConnsPerIP, 1, 10000},
		{"websocket_max_conns_per_ip", s.WebSocketMaxConnsPerIP, 1, 10000},
		{"websocket_max_total_conns", s.WebSocketMaxTotalConns, 1, 100000},
		{"account_failure_budget", s.AccountFailureBudget, 0, 10000},
		{"account_failure_window_minutes", s.AccountFailureWindowMinutes, 1, 10080},
	}
	for _, r := range ranges {
		if r.v < r.lo || r.v > r.hi {
			return fmt.Errorf("sessions.%s must be between %d and %d", r.name, r.lo, r.hi)
		}
	}
	return nil
}

func validateOrigin(origin, field string) error {
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return fmt.Errorf("%s must be an origin like https://example.com, not %q", field, origin)
	}
	return nil
}

func validateAccess(a Access) error {
	for _, cidr := range a.TrustedProxies {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("access.trusted_proxies entries must be CIDRs, not %q", cidr)
		}
	}
	for _, origin := range a.CORSAllowedOrigins {
		if err := validateOrigin(origin, "access.cors_allowed_origins entries"); err != nil {
			return err
		}
	}
	if a.AdminCORSOrigin != "" {
		if err := validateOrigin(a.AdminCORSOrigin, "access.admin_cors_origin"); err != nil {
			return err
		}
	}
	return nil
}

func validateWebsocketURL(raw, field string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "ws" && u.Scheme != "wss") || u.Host == "" || u.User != nil {
		return fmt.Errorf("%s must be a ws:// or wss:// URL, not %q", field, raw)
	}
	return nil
}

func validateHTTPURL(raw, field string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return fmt.Errorf("%s must be an http:// or https:// URL, not %q", field, raw)
	}
	return nil
}

func validateGrapevine(g Grapevine, secretSet bool) error {
	switch g.Mode {
	case "off", "presence", "full":
	default:
		return fmt.Errorf("grapevine.mode must be off, presence or full, not %q", g.Mode)
	}
	if err := validateWebsocketURL(g.URL, "grapevine.url"); err != nil {
		return err
	}
	if g.Mode != "off" && g.ClientID == "" {
		return errors.New("grapevine.client_id is required unless mode is off")
	}
	if g.Mode != "off" && !secretSet {
		return errors.New("grapevine.client_secret is required unless mode is off")
	}
	return nil
}

func validateStaffPush(p StaffPush, secretSet bool) error {
	if p.URL != "" {
		if err := validateHTTPURL(p.URL, "staff_push.url"); err != nil {
			return err
		}
		if !secretSet {
			return errors.New("staff_push.token is required when url is set")
		}
	}
	if p.MinLevel < 0 || p.MinLevel > 60 {
		return errors.New("staff_push.min_level must be between 0 and 60")
	}
	if p.MinType < 0 || p.MinType > 63 {
		return errors.New("staff_push.min_type must be between 0 and 63")
	}
	if p.RatePerMinute < 1 || p.RatePerMinute > 600 {
		return errors.New("staff_push.rate_per_minute must be between 1 and 600")
	}
	return nil
}

func validateContact(c Contact, smtpPassword, turnstileSecret bool) error {
	if c.SMTPPort < 0 || c.SMTPPort > 65535 {
		return errors.New("contact.smtp_port must be between 0 and 65535")
	}
	for _, hostname := range c.TurnstileHostnames {
		if hostname == "" || strings.Contains(hostname, "/") {
			return fmt.Errorf("contact.turnstile_hostnames entries must be hostnames, not %q", hostname)
		}
	}
	// The mail handler stays disabled until the whole set is present, exactly
	// as the seven CONTACT_* variables were all-or-nothing. Report what is
	// missing rather than guessing intent.
	var missing []string
	if c.To == "" {
		missing = append(missing, "to")
	}
	if c.SMTPHost == "" {
		missing = append(missing, "smtp_host")
	}
	if c.SMTPPort == 0 {
		missing = append(missing, "smtp_port")
	}
	if c.SMTPUser == "" {
		missing = append(missing, "smtp_user")
	}
	if !smtpPassword {
		missing = append(missing, "smtp_password")
	}
	if !turnstileSecret {
		missing = append(missing, "turnstile_secret")
	}
	if len(c.TurnstileHostnames) == 0 {
		missing = append(missing, "turnstile_hostnames")
	}
	if len(missing) > 0 {
		return &ContactIncompleteError{Missing: missing}
	}
	return nil
}

// ContactIncompleteError reports the missing fields that keep the contact
// handler disabled. It is a warning, not a rejection: saving a partial draft
// is allowed and the console shows what is missing.
type ContactIncompleteError struct{ Missing []string }

func (e *ContactIncompleteError) Error() string {
	return "contact form stays disabled until set: " + strings.Join(e.Missing, ", ")
}

// Validate checks the whole merged config. secretSet reports which sealed
// secrets exist; clear-text values are never stored outside the seal.
func Validate(c Config, secretSet map[string]bool) error {
	if err := validateSessions(c.Sessions); err != nil {
		return err
	}
	if err := validateAccess(c.Access); err != nil {
		return err
	}
	if err := validateGrapevine(c.Grapevine, secretSet[SecretGrapevineClient]); err != nil {
		return err
	}
	if err := validateStaffPush(c.StaffPush, secretSet[SecretStaffPushToken]); err != nil {
		return err
	}
	var contactErr error
	if err := validateContact(c.Contact, secretSet[SecretContactSMTP], secretSet[SecretContactTurnstile]); err != nil {
		if _, ok := err.(*ContactIncompleteError); !ok {
			return err
		}
		contactErr = err
	}
	return contactErr
}

// Live is the subset hot paths read. Save swaps it atomically; readers never
// take a lock.
type Live struct {
	LoginIdleTimeout       time.Duration
	TelnetMaxConns         int
	TelnetMaxConnsPerIP    int
	WebSocketMaxConnsPerIP int
	WebSocketMaxTotalConns int
	AccountFailureBudget   int
	AccountFailureWindow   time.Duration
	CORSAllowedOrigins     []string // nil keeps the built-in default list
	AdminCORSOrigin        string   // "" keeps the built-in default behavior
}

func liveFrom(c Config) *Live {
	return &Live{
		LoginIdleTimeout:       time.Duration(c.Sessions.LoginIdleTimeoutSeconds) * time.Second,
		TelnetMaxConns:         c.Sessions.TelnetMaxConns,
		TelnetMaxConnsPerIP:    c.Sessions.TelnetMaxConnsPerIP,
		WebSocketMaxConnsPerIP: c.Sessions.WebSocketMaxConnsPerIP,
		WebSocketMaxTotalConns: c.Sessions.WebSocketMaxTotalConns,
		AccountFailureBudget:   c.Sessions.AccountFailureBudget,
		AccountFailureWindow:   time.Duration(c.Sessions.AccountFailureWindowMinutes) * time.Minute,
		CORSAllowedOrigins:     c.Access.CORSAllowedOrigins,
		AdminCORSOrigin:        c.Access.AdminCORSOrigin,
	}
}

// parseEnvInt reads an environment integer, returning ok only for values the
// original init() blocks would have accepted (positive, in range here).
func parseEnvInt(name string, lo, hi int) (int, bool) {
	raw, set := lookupEnv(name)
	if !set || raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		return 0, false
	}
	return n, true
}

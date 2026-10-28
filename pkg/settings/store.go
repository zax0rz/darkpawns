package settings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// lookupEnv is a seam so tests can simulate a checkout deployment's exported
// variables without mutating the real environment.
var lookupEnv = os.LookupEnv

// ErrStorage mirrors notify.ErrCredentialStorage: the sealed settings exist
// but their key is missing or unreadable, and guessing would destroy data.
var ErrStorage = errors.New("settings encryption key is missing, invalid, or unreadable; restore the original private key file")

// Manager owns the settings file. All saves pass through commit(), which
// revalidates the whole merged config, keeps one previous generation, writes
// atomically, bumps the revision, and republishes the live snapshot.
type Manager struct {
	path    string
	keyPath string

	mu            sync.Mutex
	cfg           Config   // file truth; secrets sealed
	effective     Config   // file truth with environment overrides applied
	envOverridden []string // "group.field" paths an exported variable still wins
	notes         []string // migration/override notes for the boot log
	degraded      error    // set when the file on disk is unusable; saves refuse

	aead   cipher.AEAD
	onLive func(*Live)
	live   atomic.Pointer[Live]
}

// New loads the settings file, importing the current environment on first
// boot. The returned notes explain what was imported or overridden; callers
// log them once.
func New(path string) (*Manager, error) {
	keyPath := os.Getenv("DP_SETTINGS_KEY_PATH")
	if keyPath == "" {
		keyPath = path + ".key"
	}
	m := &Manager{path: filepath.Clean(path), keyPath: filepath.Clean(keyPath)}
	data, err := os.ReadFile(m.path) // #nosec G104 -- handled below
	switch {
	case err == nil:
		var stored Config
		var decodeErr error
		dec := json.NewDecoder(strings.NewReader(string(data)))
		dec.DisallowUnknownFields()
		if decodeErr = dec.Decode(&stored); decodeErr == nil {
			decodeErr = dec.Decode(&struct{}{})
			if decodeErr == io.EOF {
				decodeErr = nil
			}
		}
		if decodeErr == nil && stored.Format != Format {
			decodeErr = fmt.Errorf("format %d, this server reads %d", stored.Format, Format)
		}
		if decodeErr == nil {
			if sealErr := m.loadSealed(stored.Secrets); sealErr != nil {
				decodeErr = sealErr
			}
		}
		if decodeErr != nil {
			// A damaged settings file must not wedge the game. Run on
			// defaults plus exported variables, refuse every save, and tell
			// the operator loudly; the file on disk is never overwritten, so
			// the operator can repair or restore it.
			m.cfg = Defaults()
			m.cfg.Revision = 0
			m.applyEnvironmentOverrides()
			m.degraded = fmt.Errorf("settings file %s is unusable (%w); running on defaults until it is repaired; %s.previous retains the last good copy if it exists", m.path, decodeErr, m.path)
			m.notes = append(m.notes, m.degraded.Error())
			m.publishLive()
			return m, m.degraded
		}
		m.cfg = stored
	case errors.Is(err, os.ErrNotExist):
		migrated, notes, mErr := m.migrateFromEnvironment()
		if mErr != nil {
			// Never return a nil manager: boot continues on defaults plus
			// exported variables, saves refuse, and the operator is told.
			m.cfg = Defaults()
			m.cfg.Revision = 0
			m.applyEnvironmentOverrides()
			m.degraded = fmt.Errorf("could not create settings file %s (%w); running on defaults until the path is writable", m.path, mErr)
			m.notes = notes
			m.notes = append(m.notes, m.degraded.Error())
			m.publishLive()
			return m, m.degraded
		}
		m.cfg = migrated
		m.notes = notes
	default:
		return nil, err
	}
	m.applyEnvironmentOverrides()
	m.publishLive()
	return m, nil
}

// Degraded reports why saves are refused (corrupt or unreadable file), or
// nil when the store is healthy.
func (m *Manager) Degraded() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.degraded
}

// Path is the settings file location, for doctor-style reporting.
func (m *Manager) Path() string { return m.path }

// Notes returns boot-time import/override explanations (possibly empty).
func (m *Manager) Notes() []string { return append([]string(nil), m.notes...) }

// SetOnLive registers the boot applier. It must be set before ApplyLive so a
// save during startup cannot observe a half-wired process.
func (m *Manager) SetOnLive(apply func(*Live)) {
	m.mu.Lock()
	m.onLive = apply
	m.mu.Unlock()
}

// ApplyLive pushes the current snapshot into the applier registered by boot.
func (m *Manager) ApplyLive() {
	m.mu.Lock()
	live := liveFrom(m.effective)
	onLive := m.onLive
	m.mu.Unlock()
	m.live.Store(live)
	if onLive != nil {
		onLive(live)
	}
}

func (m *Manager) publishLive() { m.ApplyLive() }

// Live returns the current live snapshot; never nil after New.
func (m *Manager) Live() *Live {
	if l := m.live.Load(); l != nil {
		return l
	}
	return &Live{}
}

// Revision is the file-level revision used for optimistic concurrency.
func (m *Manager) Revision() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg.Revision
}

// Effective returns the merged config (file + environment overrides). Boot
// reads this to construct the restart-scoped integrations.
func (m *Manager) Effective() Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.effective
}

// SecretValue resolves a secret: an exported variable wins over the seal, and
// absence is reported as false. Used only by boot construction and tests;
// never by any view.
func (m *Manager) SecretValue(name string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.secretLocked(name)
}

func (m *Manager) secretLocked(name string) (string, bool) {
	for _, field := range envSecretNames(name) {
		if value, set := lookupEnv(field); set && value != "" {
			return value, true
		}
	}
	encoded, ok := m.cfg.Secrets[name]
	if !ok || encoded == "" {
		return "", false
	}
	plain, err := m.openSealed(name, encoded)
	if err != nil {
		return "", false
	}
	return plain, true
}

// The map values name environment variables that HOLD secrets at boot; the
// identifiers themselves are not credentials.
//
// #nosec G101 -- identifiers, not credential material
func envSecretNames(name string) []string {
	// These name environment variables that HOLD secrets at boot; the
	// strings are identifiers, not credential material.
	switch name {
	case SecretGrapevineClient:
		return []string{"GRAPEVINE_CLIENT_SECRET"} // #nosec G101 -- identifier, not a credential
	case SecretStaffPushToken:
		return []string{"DP_MUDLOG_NTFY_TOKEN"} // #nosec G101 -- identifier, not a credential
	case SecretContactSMTP:
		return []string{"CONTACT_SMTP_PASSWORD"} // #nosec G101 -- identifier, not a credential
	case SecretContactTurnstile:
		return []string{"CONTACT_TURNSTILE_SECRET"} // #nosec G101 -- identifier, not a credential
	}
	return nil
}

func (m *Manager) secretSetLocked(name string) bool {
	_, ok := m.secretLocked(name)
	return ok
}

// SettingsView is the API-facing settings document: every group, restart metadata,
// secret presence instead of secret values, and which fields an exported
// variable still overrides.
type SettingsView struct {
	Format    int                  `json:"format"`
	Revision  int64                `json:"revision"`
	Sessions  Sessions             `json:"sessions"`
	Access    Access               `json:"access"`
	Grapevine GrapevineView        `json:"grapevine"`
	StaffPush StaffPushView        `json:"staff_push"`
	Contact   ContactView          `json:"contact"`
	Meta      map[string]GroupMeta `json:"meta"`
}

// GrapevineView is the grapevine card plus secret presence.
type GrapevineView struct {
	Grapevine
	ClientSecretSet bool `json:"client_secret_set"`
}

// StaffPushView is the staff push card plus secret presence.
type StaffPushView struct {
	StaffPush
	TokenSet bool `json:"token_set"`
}

// ContactView is the contact card plus secret presence and whether the whole
// set is complete enough for the handler to run.
type ContactView struct {
	Contact
	SMTPPasswordSet    bool `json:"smtp_password_set"`
	TurnstileSecretSet bool `json:"turnstile_secret_set"`
	Effective          bool `json:"effective"`
}

// GroupMeta carries the console's field decorations.
type GroupMeta struct {
	RestartFields []string `json:"restart_fields"`
	EnvOverridden []string `json:"env_overridden,omitempty"`
}

func (m *Manager) SettingsView() SettingsView {
	m.mu.Lock()
	defer m.mu.Unlock()
	view := SettingsView{
		Format:    m.effective.Format,
		Revision:  m.cfg.Revision,
		Sessions:  m.effective.Sessions,
		Access:    m.effective.Access,
		Grapevine: GrapevineView{Grapevine: m.effective.Grapevine, ClientSecretSet: m.secretSetLocked(SecretGrapevineClient)},
		StaffPush: StaffPushView{StaffPush: m.effective.StaffPush, TokenSet: m.secretSetLocked(SecretStaffPushToken)},
		Contact: ContactView{
			Contact:            m.effective.Contact,
			SMTPPasswordSet:    m.secretSetLocked(SecretContactSMTP),
			TurnstileSecretSet: m.secretSetLocked(SecretContactTurnstile),
		},
		Meta: map[string]GroupMeta{},
	}
	view.Contact.Effective = view.Contact.SMTPPasswordSet && view.Contact.TurnstileSecretSet &&
		contactComplete(view.Contact.Contact)
	for _, group := range []string{GroupSessions, GroupAccess, GroupGrapevine, GroupStaffPush, GroupContact} {
		meta := GroupMeta{RestartFields: restartFields[group]}
		for _, path := range m.envOverridden {
			// Every group carries its overrides, live fields included, so the
			// console badges a checkout deployment's exported variables
			// wherever they still win.
			if grouped, field, _ := strings.Cut(path, "."); grouped == group && !containsField(meta.EnvOverridden, field) {
				meta.EnvOverridden = append(meta.EnvOverridden, field)
			}
		}
		view.Meta[group] = meta
	}
	return view
}

func containsField(fields []string, field string) bool {
	for _, existing := range fields {
		if existing == field {
			return true
		}
	}
	return false
}

func contactComplete(c Contact) bool {
	return c.To != "" && c.SMTPHost != "" && c.SMTPPort != 0 && c.SMTPUser != "" && len(c.TurnstileHostnames) > 0
}

func containsPath(paths []string, group, field string) bool {
	for _, p := range paths {
		if p == group+"."+field {
			return true
		}
	}
	return false
}

// SaveResult reports what a save changed.
type SaveResult struct {
	Revision        int64    `json:"revision"`
	RestartRequired []string `json:"restart_required,omitempty"`
	EnvOverridden   []string `json:"env_overridden,omitempty"`
	Warning         string   `json:"warning,omitempty"`
}

// SecretUpdate carries write-only secret updates for one save: nil keeps the
// current value, a pointer to "" clears it, anything else replaces it.
type SecretUpdate map[string]*string

var validSecrets = map[string]bool{
	SecretGrapevineClient:  true,
	SecretStaffPushToken:   true,
	SecretContactSMTP:      true,
	SecretContactTurnstile: true,
}

func (m *Manager) secretChanges(group string, changed *[]string, input SecretUpdate, out map[string]string) error {
	for name, value := range input {
		if !validSecrets[name] {
			return fmt.Errorf("unknown secret %q", name)
		}
		if value == nil {
			continue
		}
		if *value == "" {
			out[name] = ""
			*changed = append(*changed, group+"."+secretFieldName(name))
			continue
		}
		sealed, err := m.seal(name, *value)
		if err != nil {
			return err
		}
		out[name] = sealed
		*changed = append(*changed, group+"."+secretFieldName(name))
	}
	return nil
}

func secretFieldName(name string) string {
	switch name {
	case SecretGrapevineClient:
		return "client_secret"
	case SecretStaffPushToken:
		return "token"
	case SecretContactSMTP:
		return "smtp_password"
	case SecretContactTurnstile:
		return "turnstile_secret"
	}
	return name
}

func restartFor(changed []string) []string {
	var out []string
	for _, path := range changed {
		group, field, _ := strings.Cut(path, ".")
		for _, restartField := range restartFields[group] {
			if field == restartField {
				out = append(out, path)
				break
			}
		}
	}
	return out
}

// commit is the single save choke point: revision check, mutate, revalidate,
// keep the previous generation, write atomically, bump, republish.
func (m *Manager) commit(revision int64, changed []string, warnIncomplete bool, mutate func(cfg *Config) error) (SaveResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if revision != m.cfg.Revision {
		return SaveResult{}, ErrConflict
	}
	next := cloneConfig(m.cfg)
	next.Secrets = cloneSealed(m.cfg.Secrets)
	if err := mutate(&next); err != nil {
		return SaveResult{}, err
	}
	validateErr := m.validateLocked(next)
	if validateErr != nil {
		if _, incomplete := validateErr.(*ContactIncompleteError); !incomplete {
			return SaveResult{}, validateErr
		}
	}
	next.Revision = m.cfg.Revision + 1
	if err := m.persistLocked(next); err != nil {
		return SaveResult{}, err
	}
	m.cfg = next
	m.applyEnvironmentOverridesLocked()
	m.publishLocked()
	result := SaveResult{Revision: next.Revision, RestartRequired: restartFor(changed)}
	for _, path := range changed {
		group, field, _ := strings.Cut(path, ".")
		if containsPath(m.envOverridden, group, field) {
			result.EnvOverridden = append(result.EnvOverridden, path)
		}
	}
	if validateErr != nil && warnIncomplete {
		result.Warning = "contact form stays disabled until every field is set"
	}
	return result, nil
}

// validateLocked treats an incomplete contact set as a warning, not a
// rejection: drafts are legal, the handler stays off.
func (m *Manager) validateLocked(cfg Config) error {
	err := Validate(cfg, map[string]bool{
		SecretGrapevineClient:  m.secretIn(cfg, SecretGrapevineClient),
		SecretStaffPushToken:   m.secretIn(cfg, SecretStaffPushToken),
		SecretContactSMTP:      m.secretIn(cfg, SecretContactSMTP),
		SecretContactTurnstile: m.secretIn(cfg, SecretContactTurnstile),
	})
	if err == nil {
		return nil
	}
	if _, ok := err.(*ContactIncompleteError); ok {
		return err
	}
	return err
}

func (m *Manager) secretIn(cfg Config, name string) bool {
	for _, field := range envSecretNames(name) {
		if value, set := lookupEnv(field); set && value != "" {
			return true
		}
	}
	sealed, ok := cfg.Secrets[name]
	return ok && sealed != ""
}

// SaveSessions updates the live session limits.
func (m *Manager) SaveSessions(revision int64, s Sessions) (SaveResult, error) {
	changed := diffStruct(GroupSessions, m.current(GroupSessions), s)
	return m.commit(revision, changed, false, func(cfg *Config) error {
		cfg.Sessions = s
		return nil
	})
}

// SaveAccess updates proxy trust (restart) and CORS origins (live).
func (m *Manager) SaveAccess(revision int64, a Access) (SaveResult, error) {
	changed := diffStruct(GroupAccess, m.current(GroupAccess), a)
	return m.commit(revision, changed, false, func(cfg *Config) error {
		cfg.Access = a
		return nil
	})
}

// SaveGrapevine updates the relay card; secrets is write-only input.
func (m *Manager) SaveGrapevine(revision int64, g Grapevine, secrets SecretUpdate) (SaveResult, error) {
	changed := diffStruct(GroupGrapevine, m.current(GroupGrapevine), g)
	result, err := m.commit(revision, changed, false, func(cfg *Config) error {
		cfg.Grapevine = g
		return m.secretChanges(GroupGrapevine, &changed, secrets, cfg.Secrets)
	})
	if err == nil {
		// secretChanges appended to changed inside the mutation; recompute.
		result.RestartRequired = restartFor(changed)
	}
	return result, err
}

// SaveStaffPush updates the ntfy push card.
func (m *Manager) SaveStaffPush(revision int64, p StaffPush, secrets SecretUpdate) (SaveResult, error) {
	changed := diffStruct(GroupStaffPush, m.current(GroupStaffPush), p)
	result, err := m.commit(revision, changed, false, func(cfg *Config) error {
		cfg.StaffPush = p
		return m.secretChanges(GroupStaffPush, &changed, secrets, cfg.Secrets)
	})
	if err == nil {
		result.RestartRequired = restartFor(changed)
	}
	return result, err
}

// SaveContact updates the contact form card.
func (m *Manager) SaveContact(revision int64, c Contact, secrets SecretUpdate) (SaveResult, error) {
	changed := diffStruct(GroupContact, m.current(GroupContact), c)
	result, err := m.commit(revision, changed, true, func(cfg *Config) error {
		cfg.Contact = c
		return m.secretChanges(GroupContact, &changed, secrets, cfg.Secrets)
	})
	if err == nil {
		result.RestartRequired = restartFor(changed)
	}
	return result, err
}

// current returns a group's current effective value for diffing. The switch
// keeps it type-safe per group.
func (m *Manager) current(group string) any {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch group {
	case GroupSessions:
		return m.effective.Sessions
	case GroupAccess:
		return m.effective.Access
	case GroupGrapevine:
		return m.effective.Grapevine
	case GroupStaffPush:
		return m.effective.StaffPush
	case GroupContact:
		return m.effective.Contact
	}
	return nil
}

// diffStruct lists changed field paths ("group.field") by comparing JSON
// encodings field by field, so slices and scalars compare uniformly.
func diffStruct(group string, before, after any) []string {
	var changed []string
	if before == nil || after == nil {
		return changed
	}
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	var av, bv map[string]json.RawMessage
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return changed
	}
	for key, value := range bv {
		other, ok := av[key]
		if !ok || string(value) != string(other) {
			changed = append(changed, group+"."+key)
		}
	}
	return changed
}

// Bundle is the export/restore document. Secrets are never part of it.
type Bundle struct {
	Format        int       `json:"format"`
	GeneratedAt   time.Time `json:"generated_at"`
	ServerVersion string    `json:"server_version"`
	Revision      int64     `json:"revision"`
	Sessions      Sessions  `json:"sessions"`
	Access        Access    `json:"access"`
	Grapevine     Grapevine `json:"grapevine"`
	StaffPush     StaffPush `json:"staff_push"`
	Contact       Contact   `json:"contact"`
}

// Export builds the downloadable bundle from current effective values.
func (m *Manager) Export(serverVersion string, now time.Time) Bundle {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Bundle{
		Format:        Format,
		GeneratedAt:   now,
		ServerVersion: serverVersion,
		Revision:      m.cfg.Revision,
		Sessions:      m.effective.Sessions,
		Access:        m.effective.Access,
		Grapevine:     m.effective.Grapevine,
		StaffPush:     m.effective.StaffPush,
		Contact:       m.effective.Contact,
	}
}

// DiffEntry is one field-level difference for the restore preview.
type DiffEntry struct {
	Field    string          `json:"field"`
	Current  json.RawMessage `json:"current"`
	Incoming json.RawMessage `json:"incoming"`
}

// RestorePreview is the server-validated result of parsing an uploaded
// bundle. Applying it is a separate, confirmed step.
type RestorePreview struct {
	Diff             []DiffEntry `json:"diff"`
	UnknownKeys      []string    `json:"unknown_keys,omitempty"`
	SecretsUntouched []string    `json:"secrets_untouched,omitempty"`
	RestartRequired  []string    `json:"restart_required,omitempty"`
	Warning          string      `json:"warning,omitempty"`
}

// RestoreValidate parses a raw bundle document, reports unknown keys instead
// of tripping on them, and diffs it against current values. Secrets are not
// exportable, so they never appear as restorable changes.
func (m *Manager) RestoreValidate(raw []byte) (RestorePreview, error) {
	var preview RestorePreview
	var document map[string]json.RawMessage
	if err := json.Unmarshal(raw, &document); err != nil {
		return preview, fmt.Errorf("bundle is not valid JSON: %w", err)
	}
	known := map[string]bool{
		"format": true, "generated_at": true, "server_version": true, "revision": true,
		"sessions": true, "access": true, "grapevine": true, "staff_push": true, "contact": true,
	}
	for key := range document {
		if !known[key] {
			preview.UnknownKeys = append(preview.UnknownKeys, key)
		}
	}
	var bundle Bundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return preview, fmt.Errorf("bundle fields are invalid: %w", err)
	}
	if bundle.Format != Format {
		return preview, fmt.Errorf("bundle has format %d; this server reads format %d", bundle.Format, Format)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	candidate := cloneConfig(m.cfg)
	candidate.Sessions = bundle.Sessions
	candidate.Access = bundle.Access
	candidate.Grapevine = bundle.Grapevine
	candidate.StaffPush = bundle.StaffPush
	candidate.Contact = bundle.Contact
	if err := m.validateLocked(candidate); err != nil {
		if _, ok := err.(*ContactIncompleteError); !ok {
			return preview, err
		}
		preview.Warning = err.Error()
	}
	for _, pair := range []struct {
		group  string
		before any
		after  any
	}{
		{GroupSessions, m.effective.Sessions, bundle.Sessions},
		{GroupAccess, m.effective.Access, bundle.Access},
		{GroupGrapevine, m.effective.Grapevine, bundle.Grapevine},
		{GroupStaffPush, m.effective.StaffPush, bundle.StaffPush},
		{GroupContact, m.effective.Contact, bundle.Contact},
	} {
		for _, path := range diffStruct(pair.group, pair.before, pair.after) {
			field := strings.TrimPrefix(path, pair.group+".")
			current := jsonField(pair.before, field)
			incoming := jsonField(pair.after, field)
			preview.Diff = append(preview.Diff, DiffEntry{Field: path, Current: current, Incoming: incoming})
			preview.RestartRequired = append(preview.RestartRequired, restartFor([]string{path})...)
		}
	}
	for _, name := range []string{SecretGrapevineClient, SecretStaffPushToken, SecretContactSMTP, SecretContactTurnstile} {
		if m.secretIn(m.cfg, name) {
			preview.SecretsUntouched = append(preview.SecretsUntouched, name)
		}
	}
	return preview, nil
}

func jsonField(value any, field string) json.RawMessage {
	data, _ := json.Marshal(value)
	var document map[string]json.RawMessage
	if json.Unmarshal(data, &document) != nil {
		return json.RawMessage("null")
	}
	if raw, ok := document[field]; ok {
		return raw
	}
	return json.RawMessage("null")
}

// RestoreApply replaces the bundle's groups wholesale, keeping secrets and
// the revision check honest: the revision the operator previewed must still
// be current.
func (m *Manager) RestoreApply(revision int64, raw []byte) (SaveResult, error) {
	var bundle Bundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return SaveResult{}, fmt.Errorf("bundle fields are invalid: %w", err)
	}
	if bundle.Format != Format {
		return SaveResult{}, fmt.Errorf("bundle has format %d; this server reads format %d", bundle.Format, Format)
	}
	var changed []string
	result, err := m.commit(revision, changed, true, func(cfg *Config) error {
		changed = append(changed, diffStruct(GroupSessions, m.effective.Sessions, bundle.Sessions)...)
		changed = append(changed, diffStruct(GroupAccess, m.effective.Access, bundle.Access)...)
		changed = append(changed, diffStruct(GroupGrapevine, m.effective.Grapevine, bundle.Grapevine)...)
		changed = append(changed, diffStruct(GroupStaffPush, m.effective.StaffPush, bundle.StaffPush)...)
		changed = append(changed, diffStruct(GroupContact, m.effective.Contact, bundle.Contact)...)
		cfg.Sessions = bundle.Sessions
		cfg.Access = bundle.Access
		cfg.Grapevine = bundle.Grapevine
		cfg.StaffPush = bundle.StaffPush
		cfg.Contact = bundle.Contact
		return nil
	})
	if err != nil {
		return result, err
	}
	result.RestartRequired = restartFor(changed)
	return result, nil
}

func cloneConfig(c Config) Config {
	out := c
	out.Access.TrustedProxies = append([]string(nil), c.Access.TrustedProxies...)
	out.Access.CORSAllowedOrigins = append([]string(nil), c.Access.CORSAllowedOrigins...)
	out.Contact.TurnstileHostnames = append([]string(nil), c.Contact.TurnstileHostnames...)
	return out
}

func cloneSealed(sealed map[string]string) map[string]string {
	out := make(map[string]string, len(sealed))
	for key, value := range sealed {
		out[key] = value
	}
	return out
}

// persistLocked keeps one previous generation, then replaces the file
// atomically. The remove-then-rename pair mirrors the launcher's writePrivate:
// Windows cannot rename over an existing file, and a crash between the two
// leaves .previous as the recovery copy.
func (m *Manager) persistLocked(cfg Config) error {
	if m.degraded != nil {
		return m.degraded
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o750); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	if previous, err := os.ReadFile(m.path); err == nil {
		if writeErr := os.WriteFile(m.path+".previous", previous, 0o600); writeErr != nil { // #nosec G703 -- operator-configured settings path, never request-derived
			return fmt.Errorf("retain previous settings: %w", writeErr)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("retain previous settings: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(filepath.Dir(m.path), ".settings-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer func() {
		if removeErr := os.Remove(name); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	if err = temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	_, err = temp.Write(data)
	err = errors.Join(err, temp.Sync(), temp.Close())
	if err != nil {
		return err
	}
	if err = os.Remove(m.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(name, m.path)
}

func (m *Manager) publishLocked() {
	live := liveFrom(m.effective)
	onLive := m.onLive
	m.live.Store(live)
	if onLive != nil {
		onLive(live)
	}
}

// applyEnvironmentOverrides recomputes effective from file truth: an exported
// variable set in this process (a checkout deployment's flag file) wins and
// is reported, so the console can badge it instead of losing it.
func (m *Manager) applyEnvironmentOverrides() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applyEnvironmentOverridesLocked()
}

func (m *Manager) applyEnvironmentOverridesLocked() {
	effective := cloneConfig(m.cfg)
	var overridden []string
	applyInt := func(field string, envName string, lo, hi int, target *int) {
		if value, ok := parseEnvInt(envName, lo, hi); ok && value != *target {
			*target = value
			overridden = append(overridden, field)
		}
	}
	applyInt("sessions.login_idle_timeout_seconds", "LOGIN_IDLE_TIMEOUT", 1, 3600, &effective.Sessions.LoginIdleTimeoutSeconds)
	applyInt("sessions.telnet_max_conns", "TELNET_MAX_CONNS", 1, 100000, &effective.Sessions.TelnetMaxConns)
	applyInt("sessions.telnet_max_conns_per_ip", "TELNET_MAX_CONNS_PER_IP", 1, 10000, &effective.Sessions.TelnetMaxConnsPerIP)
	applyInt("sessions.websocket_max_conns_per_ip", "WEBSOCKET_MAX_CONNS_PER_IP", 1, 10000, &effective.Sessions.WebSocketMaxConnsPerIP)
	applyInt("sessions.websocket_max_total_conns", "WEBSOCKET_MAX_TOTAL_CONNS", 1, 100000, &effective.Sessions.WebSocketMaxTotalConns)
	applyInt("sessions.account_failure_budget", "DP_ACCOUNT_FAILURE_BUDGET", 0, 10000, &effective.Sessions.AccountFailureBudget)
	if raw, set := lookupEnv("DP_ACCOUNT_FAILURE_WINDOW"); set && raw != "" {
		if duration, err := time.ParseDuration(raw); err == nil && duration > 0 && duration <= 7*24*time.Hour {
			minutes := int(duration / time.Minute)
			if minutes != effective.Sessions.AccountFailureWindowMinutes {
				effective.Sessions.AccountFailureWindowMinutes = minutes
				overridden = append(overridden, "sessions.account_failure_window_minutes")
			}
		}
	}
	if raw, set := lookupEnv("TRUSTED_PROXIES"); set && raw != "" {
		proxies := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' })
		effective.Access.TrustedProxies = proxies
		overridden = append(overridden, "access.trusted_proxies")
	}
	if raw, set := lookupEnv("CORS_ALLOWED_ORIGINS"); set && raw != "" {
		effective.Access.CORSAllowedOrigins = splitList(raw)
		overridden = append(overridden, "access.cors_allowed_origins")
	}
	if raw, set := lookupEnv("ADMIN_CORS_ORIGIN"); set && raw != "" {
		effective.Access.AdminCORSOrigin = raw
		overridden = append(overridden, "access.admin_cors_origin")
	}
	applyString := func(field, envName string, target *string) {
		if raw, set := lookupEnv(envName); set && raw != "" && raw != *target {
			*target = raw
			overridden = append(overridden, field)
		}
	}
	applyString("grapevine.mode", "GRAPEVINE_MODE", &effective.Grapevine.Mode)
	applyString("grapevine.url", "GRAPEVINE_URL", &effective.Grapevine.URL)
	applyString("grapevine.client_id", "GRAPEVINE_CLIENT_ID", &effective.Grapevine.ClientID)
	applyString("staff_push.url", "DP_MUDLOG_NTFY_URL", &effective.StaffPush.URL)
	applyInt("staff_push.min_level", "DP_MUDLOG_NTFY_LEVEL", 0, 60, &effective.StaffPush.MinLevel)
	applyInt("staff_push.min_type", "DP_MUDLOG_NTFY_TYPE", 0, 63, &effective.StaffPush.MinType)
	applyInt("staff_push.rate_per_minute", "DP_MUDLOG_NTFY_RATE", 1, 600, &effective.StaffPush.RatePerMinute)
	applyString("contact.to", "CONTACT_TO", &effective.Contact.To)
	applyString("contact.smtp_host", "CONTACT_SMTP_HOST", &effective.Contact.SMTPHost)
	applyInt("contact.smtp_port", "CONTACT_SMTP_PORT", 1, 65535, &effective.Contact.SMTPPort)
	applyString("contact.smtp_user", "CONTACT_SMTP_USER", &effective.Contact.SMTPUser)
	applyList := func(field, envName string, target *[]string) {
		if raw, set := lookupEnv(envName); set && raw != "" {
			*target = splitList(raw)
			overridden = append(overridden, field)
		}
	}
	applyList("contact.turnstile_hostnames", "CONTACT_TURNSTILE_HOSTNAMES", &effective.Contact.TurnstileHostnames)
	for _, name := range []string{SecretGrapevineClient, SecretStaffPushToken, SecretContactSMTP, SecretContactTurnstile} {
		for _, field := range envSecretNames(name) {
			if value, set := lookupEnv(field); set && value != "" {
				overridden = append(overridden, secretGroup(name)+"."+secretFieldName(name))
			}
		}
	}
	m.effective = effective
	m.envOverridden = overridden
}

func secretGroup(name string) string {
	switch name {
	case SecretGrapevineClient:
		return GroupGrapevine
	case SecretStaffPushToken:
		return GroupStaffPush
	default:
		return GroupContact
	}
}

func splitList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// migrateFromEnvironment builds the first settings file from exported
// variables, sealing their secret values. Invalid values keep defaults and
// produce a note; nothing here can refuse to boot.
func (m *Manager) migrateFromEnvironment() (Config, []string, error) {
	cfg := Defaults()
	var notes []string
	setInt := func(envName string, lo, hi int, target *int, fallback int) {
		if raw, set := lookupEnv(envName); set && raw != "" {
			if value, err := strconv.Atoi(raw); err == nil && value >= lo && value <= hi {
				*target = value
				return
			}
			notes = append(notes, envName+" ignored (invalid or out of range); keeping default")
			*target = fallback
		}
	}
	setInt("LOGIN_IDLE_TIMEOUT", 1, 3600, &cfg.Sessions.LoginIdleTimeoutSeconds, 120)
	setInt("TELNET_MAX_CONNS", 1, 100000, &cfg.Sessions.TelnetMaxConns, 200)
	setInt("TELNET_MAX_CONNS_PER_IP", 1, 10000, &cfg.Sessions.TelnetMaxConnsPerIP, 8)
	setInt("WEBSOCKET_MAX_CONNS_PER_IP", 1, 10000, &cfg.Sessions.WebSocketMaxConnsPerIP, 8)
	setInt("WEBSOCKET_MAX_TOTAL_CONNS", 1, 100000, &cfg.Sessions.WebSocketMaxTotalConns, 500)
	setInt("DP_ACCOUNT_FAILURE_BUDGET", 0, 10000, &cfg.Sessions.AccountFailureBudget, 0)
	setInt("DP_MUDLOG_NTFY_LEVEL", 0, 60, &cfg.StaffPush.MinLevel, 31)
	setInt("DP_MUDLOG_NTFY_TYPE", 0, 63, &cfg.StaffPush.MinType, 1)
	setInt("DP_MUDLOG_NTFY_RATE", 1, 600, &cfg.StaffPush.RatePerMinute, 30)
	if raw, set := lookupEnv("DP_ACCOUNT_FAILURE_WINDOW"); set && raw != "" {
		if duration, err := time.ParseDuration(raw); err == nil && duration > 0 && duration <= 7*24*time.Hour {
			cfg.Sessions.AccountFailureWindowMinutes = int(duration / time.Minute)
		} else {
			notes = append(notes, "DP_ACCOUNT_FAILURE_WINDOW ignored (invalid); keeping default")
		}
	}
	setString := func(envName string, target *string) {
		if raw, set := lookupEnv(envName); set && raw != "" {
			*target = raw
		}
	}
	if raw, set := lookupEnv("TRUSTED_PROXIES"); set && raw != "" {
		cfg.Access.TrustedProxies = strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' })
	}
	if raw, set := lookupEnv("CORS_ALLOWED_ORIGINS"); set && raw != "" {
		cfg.Access.CORSAllowedOrigins = splitList(raw)
	}
	setString("ADMIN_CORS_ORIGIN", &cfg.Access.AdminCORSOrigin)
	setString("GRAPEVINE_MODE", &cfg.Grapevine.Mode)
	setString("GRAPEVINE_URL", &cfg.Grapevine.URL)
	setString("GRAPEVINE_CLIENT_ID", &cfg.Grapevine.ClientID)
	setString("DP_MUDLOG_NTFY_URL", &cfg.StaffPush.URL)
	setString("CONTACT_TO", &cfg.Contact.To)
	setString("CONTACT_SMTP_HOST", &cfg.Contact.SMTPHost)
	setInt("CONTACT_SMTP_PORT", 1, 65535, &cfg.Contact.SMTPPort, 0)
	setString("CONTACT_SMTP_USER", &cfg.Contact.SMTPUser)
	if raw, set := lookupEnv("CONTACT_TURNSTILE_HOSTNAMES"); set && raw != "" {
		cfg.Contact.TurnstileHostnames = splitList(raw)
	}
	cfg.Secrets = map[string]string{}
	// #nosec G101 -- the values name environment variables that hold secrets
	// at boot; the literals are identifiers, not credential material.
	//
	//nolint:gosec
	for name, envName := range map[string]string{
		SecretGrapevineClient:  "GRAPEVINE_CLIENT_SECRET",
		SecretStaffPushToken:   "DP_MUDLOG_NTFY_TOKEN",
		SecretContactSMTP:      "CONTACT_SMTP_PASSWORD",
		SecretContactTurnstile: "CONTACT_TURNSTILE_SECRET",
	} {
		if raw, set := lookupEnv(envName); set && raw != "" {
			sealed, err := m.seal(name, raw)
			if err != nil {
				return cfg, notes, err
			}
			cfg.Secrets[name] = sealed
		}
	}
	cfg.Revision = 1
	if err := Validate(cfg, sealedSet(cfg)); err != nil {
		if _, ok := err.(*ContactIncompleteError); !ok {
			// A bad exported value must not wedge the boot; drop the whole
			// group to defaults and note it. The operator still has the env.
			notes = append(notes, "imported settings invalid ("+err.Error()+"); review in the console")
			defaults := Defaults()
			cfg.Sessions, cfg.Access, cfg.Grapevine, cfg.StaffPush, cfg.Contact = defaults.Sessions, defaults.Access, defaults.Grapevine, defaults.StaffPush, defaults.Contact
		}
	}
	if err := m.persistLocked(cfg); err != nil {
		return cfg, notes, err
	}
	notes = append(notes, "settings imported from environment variables into "+m.path)
	return cfg, notes, nil
}

func sealedSet(cfg Config) map[string]bool {
	out := map[string]bool{}
	for name, sealed := range cfg.Secrets {
		out[name] = sealed != ""
	}
	return out
}

func (m *Manager) loadSealed(sealed map[string]string) error {
	if len(sealed) == 0 {
		return nil
	}
	for name := range sealed {
		switch name {
		case SecretGrapevineClient, SecretStaffPushToken, SecretContactSMTP, SecretContactTurnstile:
		default:
			return fmt.Errorf("unknown sealed secret %q", name)
		}
	}
	if _, err := m.aeadKey(false); err != nil {
		return ErrStorage
	}
	return nil
}

func (m *Manager) aeadKey(create bool) (cipher.AEAD, error) {
	if m.aead != nil {
		return m.aead, nil
	}
	key, err := os.ReadFile(m.keyPath) // #nosec G304 -- operator-configured key path
	if errors.Is(err, os.ErrNotExist) && create {
		if err := os.MkdirAll(filepath.Dir(m.keyPath), 0o750); err != nil {
			return nil, ErrStorage
		}
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return nil, ErrStorage
		}
		f, openErr := os.OpenFile(m.keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 -- operator-configured key path
		if errors.Is(openErr, os.ErrExist) {
			return m.aeadKey(false)
		}
		if openErr != nil {
			return nil, ErrStorage
		}
		_, err = f.Write(key)
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			return nil, ErrStorage
		}
	} else if err != nil {
		return nil, ErrStorage
	}
	info, err := os.Stat(m.keyPath) // #nosec G304 -- operator-configured key path
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || len(key) != 32 {
		return nil, ErrStorage
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrStorage
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrStorage
	}
	m.aead = aead
	return aead, nil
}

func settingsAAD(name string) []byte { return []byte("darkpawns-settings/v1/" + name) }

func (m *Manager) seal(name, value string) (string, error) {
	aead, err := m.aeadKey(m.aead == nil)
	if err != nil {
		return "", ErrStorage
	}
	plain, err := json.Marshal(value)
	if err != nil {
		return "", ErrStorage
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", ErrStorage
	}
	return base64.StdEncoding.EncodeToString(aead.Seal(nonce, nonce, plain, settingsAAD(name))), nil
}

func (m *Manager) openSealed(name, encoded string) (string, error) {
	aead, err := m.aeadKey(false)
	if err != nil {
		return "", ErrStorage
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) < aead.NonceSize() {
		return "", ErrStorage
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], settingsAAD(name))
	if err != nil {
		return "", ErrStorage
	}
	var value string
	if json.Unmarshal(plain, &value) != nil {
		return "", ErrStorage
	}
	return value, nil
}

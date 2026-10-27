// Package notify delivers opt-in operational notifications outside the game loop.
// Structured events carry only public character names and levels, never mudlog text.
package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/time/rate"
)

const (
	Pushover  = "pushover"
	Discord   = "discord"
	Gossip    = "gossip"
	Heartbeat = "heartbeat"
	Created   = "player.created"
	Login     = "player.login"
	Logout    = "player.logout"
	Death     = "player.death"
	Level     = "player.level"
)

var eventIDs = []string{Created, Login, Logout, Death, Level}

// Event is intentionally smaller than a log record. No host, room or free text.
type Event struct {
	ID, Player string
	Level      int
}

type Destination struct {
	Enabled bool     `json:"enabled"`
	Events  []string `json:"events"`
}
type Config struct {
	Revision     int64       `json:"revision"`
	Pushover     Destination `json:"pushover"`
	Discord      Destination `json:"discord"`
	Heartbeat    bool        `json:"heartbeat"`
	Gossip       bool        `json:"gossip"`
	GossipNotice bool        `json:"gossip_notice"`
}
type Status struct {
	Queued uint64 `json:"queued"`

	Source      string     `json:"source"`
	Ready       bool       `json:"ready"`
	LastSuccess *time.Time `json:"last_success"`
	LastFailure *time.Time `json:"last_failure"`
	LastError   string     `json:"last_error"`
	Dropped     uint64     `json:"dropped"`
}
type View struct {
	Config Config            `json:"config"`
	Status map[string]Status `json:"status"`
}

var (
	ErrConflict = errors.New("settings changed; reload before saving")
	ErrInvalid  = errors.New("unknown notification event")
	ErrNotReady = errors.New("provider credentials are missing or invalid; configure this destination")
	ErrBusy     = errors.New("notification queue or rate limit reached; try again shortly")
)

type (
	job struct {
		message  string
		revision int64
		test     bool
	}
	worker struct {
		jobs    chan job
		limiter *rate.Limiter
	}
)

// Service owns one bounded queue per destination; a slow provider cannot block
// the game or another provider. Delivery is best effort, with no replay or retry.
type Service struct {
	saveMu sync.Mutex
	vault  *credentialVault

	mu      sync.Mutex
	cfg     Config
	path    string
	status  map[string]Status
	workers map[string]*worker
	send    func(context.Context, string, string) error
	ready   func(string) bool
}

// New loads settings; malformed files fail closed instead of silently enabling
// defaults or discarding an operator's selections. Credentials are loaded from encrypted storage or legacy environment settings.
func New(path string) (*Service, error) {
	s := &Service{path: filepath.Clean(path), cfg: Config{Revision: 1, GossipNotice: true, Pushover: Destination{Events: []string{}}, Discord: Destination{Events: []string{}}}, status: map[string]Status{}, workers: map[string]*worker{}}
	stored := diskConfig{Config: Config{GossipNotice: true}}
	data, err := os.ReadFile(s.path) // #nosec G703 -- path comes only from server configuration
	if err == nil {
		if err = json.Unmarshal(data, &stored); err != nil {
			return nil, fmt.Errorf("decode notification settings: %w", err)
		}
		s.cfg = stored.Config
		if err = validate(s.cfg); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read notification settings: %w", err)
	}
	s.vault, err = newCredentialVault(s.path, stored.Credentials)
	if err != nil {
		return nil, err
	}
	s.send = credentialSender(s.vault.snapshot.Load())
	s.ready = func(id string) bool { return validCredentials(id, s.vault.snapshot.Load().values[id]) }
	for _, id := range []string{Pushover, Discord, Heartbeat, Gossip} {
		s.workers[id] = &worker{jobs: make(chan job, 16), limiter: rate.NewLimiter(rate.Every(2*time.Second), 5)}
	}
	return s, nil
}

func validate(c Config) error {
	if c.Revision < 1 {
		return ErrInvalid
	}
	for _, d := range []Destination{c.Pushover, c.Discord} {
		if len(d.Events) > len(eventIDs) {
			return ErrInvalid
		}
		seen := map[string]bool{}
		for _, id := range d.Events {
			if !slices.Contains(eventIDs, id) || seen[id] {
				return ErrInvalid
			}
			seen[id] = true
		}
	}
	return nil
}

func clone(c Config) Config {
	c.Pushover.Events = append([]string{}, c.Pushover.Events...)
	c.Discord.Events = append([]string{}, c.Discord.Events...)
	return c
}

func (s *Service) View() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := View{Config: clone(s.cfg), Status: map[string]Status{}}
	for id := range s.workers {
		st := s.status[id]
		st.Ready = s.ready(id)
		st.Source = s.vault.snapshot.Load().sources[id]
		if !st.Ready {
			st.Source = "none"
		}
		out.Status[id] = st
	}
	return out
}

func (s *Service) Save(c Config) error {
	if err := validate(c); err != nil {
		return err
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	revision := s.cfg.Revision
	s.mu.Unlock()
	if c.Revision != revision {
		return ErrConflict
	}
	for id, enabled := range map[string]bool{Pushover: c.Pushover.Enabled, Discord: c.Discord.Enabled, Heartbeat: c.Heartbeat, Gossip: c.Gossip} {
		if enabled && !s.ready(id) {
			return ErrNotReady
		}
	}
	c.Revision++
	if err := s.persist(c, s.vault.encrypted); err != nil {
		return err
	}

	s.mu.Lock()
	s.cfg = clone(c)
	s.mu.Unlock()
	return nil
}

// persist is called only under saveMu, outside the publisher mutex.
func (s *Service) persist(c Config, encrypted map[string]string) error {
	data, err := json.MarshalIndent(diskConfig{Config: c, Credentials: encrypted}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".notifications-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() {
		if err := os.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("Remove temporary notification settings", "error", err)
		}
	}()
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp, s.path); err != nil {
		return err
	}
	return nil
}

func (s *Service) destination(id string) Destination {
	if id == Pushover {
		return s.cfg.Pushover
	}
	return s.cfg.Discord
}

// Publish does no disk or network I/O, and never waits for queue space.
func (s *Service) Publish(e Event) {
	if s == nil {
		return
	}
	message := format(e)
	if message == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range []string{Pushover, Discord} {
		d := s.destination(id)
		if d.Enabled && slices.Contains(d.Events, e.ID) {
			_ = s.enqueue(id, job{message: message, revision: s.cfg.Revision})
		}
	}
}

func (s *Service) enqueue(id string, j job) error {
	w := s.workers[id]
	if !w.limiter.Allow() {
		s.drop(id)
		return ErrBusy
	}
	select {
	case w.jobs <- j:
		st := s.status[id]
		st.Queued++
		s.status[id] = st
		return nil
	default:
		s.drop(id)
		return ErrBusy
	}
}
func (s *Service) drop(id string) { st := s.status[id]; st.Dropped++; s.status[id] = st }

// Test uses the saved configuration and can test a disabled destination.
// Every channel the console can configure accepts a test, including the gossip
// mirror and the heartbeat: unverified credentials are exactly what an operator
// needs to prove before enabling a channel.
func (s *Service) Test(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id != Pushover && id != Discord && id != Gossip && id != Heartbeat {
		return ErrInvalid
	}
	if !s.ready(id) {
		return ErrNotReady
	}
	return s.enqueue(id, job{message: testMessage(id), revision: s.cfg.Revision, test: true})
}

// A gossip test is a real line in a public channel, so it names itself instead
// of claiming an event subscription. A heartbeat test carries no message: the
// check-in is the request.
func testMessage(id string) string {
	switch id {
	case Gossip:
		return "Test line from Dark Pawns. Public gossip echoes here while the mirror is on."
	case Heartbeat:
		return ""
	}
	return "Test notification. This destination is ready to receive selected game events."
}

// Run starts delivery workers and optionally an outbound Healthchecks heartbeat.
// progressed must read a timestamp from the completed game-loop callback.
func (s *Service) Run(ctx context.Context, progressed func() time.Time) {
	for id, w := range s.workers {
		if id == Gossip {
			go s.runGossipWorker(ctx, w)
			continue
		}
		go s.runWorker(ctx, id, w)
	}
	if progressed != nil {
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case now := <-ticker.C:
					s.heartbeat(now, progressed())
				}
			}
		}()
	}
}

func (s *Service) heartbeat(now, last time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg.Heartbeat && !last.IsZero() && !last.After(now) && now.Sub(last) < 30*time.Second {
		_ = s.enqueue(Heartbeat, job{revision: s.cfg.Revision})
	}
}

func (s *Service) runWorker(ctx context.Context, id string, w *worker) {
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-w.jobs:
			s.mu.Lock()
			enabled := s.cfg.Heartbeat
			if id != Heartbeat {
				enabled = s.destination(id).Enabled
			}
			allowed := j.revision == s.cfg.Revision && (enabled || j.test)
			send := s.send // bind the destination before releasing the configuration epoch
			s.mu.Unlock()
			if !allowed {
				continue
			}
			deliveryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := send(deliveryCtx, id, j.message)
			cancel()
			s.mu.Lock()
			if j.revision != s.cfg.Revision {
				s.mu.Unlock()
				continue
			}
			st := s.status[id]
			now := time.Now().UTC()
			if err != nil {
				st.LastFailure = &now
				st.LastError = err.Error()
			} else {
				st.LastSuccess = &now
				st.LastError = ""
			}
			s.status[id] = st
			s.mu.Unlock()
		}
	}
}

func format(e Event) string {
	// Names are usually alphabetic. Sanitize imported names too: no mentions,
	// markdown, links, control bytes or invisible formatting in public messages.
	name := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		}
		return -1
	}, e.Player)
	if len([]rune(name)) > 32 {
		name = string([]rune(name)[:32])
	}
	if name == "" {
		return ""
	}
	switch e.ID {
	case Created:
		return name + " created a character."
	case Login:
		return name + " entered the game."
	case Logout:
		return name + " left the game."
	case Death:
		return name + " died."
	case Level:
		if e.Level > 0 {
			return fmt.Sprintf("%s reached level %d.", name, e.Level)
		}
	}
	return ""
}

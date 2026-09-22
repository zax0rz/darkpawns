package mudletmap

import (
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Cache holds the generated map and area names, rebuilt from the live world
// at most once per TTL. Builders edit rooms through OLC while the game runs,
// so the map is generated from the world the server has now, not from the
// world files it booted with.
type Cache struct {
	world World
	ttl   time.Duration
	now   func() time.Time

	mu      sync.Mutex
	built   time.Time
	data    []byte
	version string
	areas   map[int]string
}

// NewCache returns a cache over world that rebuilds at most every ttl.
func NewCache(world World, ttl time.Duration) *Cache {
	return &Cache{world: world, ttl: ttl, now: time.Now}
}

func (c *Cache) refresh() {
	if c.data != nil && c.now().Sub(c.built) < c.ttl {
		return
	}
	data, err := Generate(c.world)
	if err != nil {
		slog.Error("Mudlet map generation failed", "error", err)
		return
	}
	c.data, c.version, c.areas, c.built = data, Version(data), AreaNames(c.world), c.now()
}

// Map returns the current map document and its version.
func (c *Cache) Map() ([]byte, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refresh()
	return c.data, c.version
}

// AreaName is the Mudlet area name for a zone: the same name the downloaded
// map gives it, so rooms mapped live land in the downloaded area.
func (c *Cache) AreaName(zone int) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refresh()
	return c.areas[zone]
}

// ServeHTTP serves the map as XML, revalidated by version so a client that
// already has the current map downloads nothing.
func (c *Cache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, version := c.Map()
	if data == nil {
		http.Error(w, "map unavailable", http.StatusServiceUnavailable)
		return
	}
	etag := `"` + version + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	if r.Method == http.MethodHead {
		return
	}
	if _, err := w.Write(data); err != nil {
		slog.Debug("Mudlet map write failed", "error", err)
	}
}

package portable

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	bundle := t.TempDir()
	files := map[string]string{}
	for name, body := range map[string]string{"lib/world/wld/index": "0.wld\n$\n", "lib/world/wld/0.wld": "world", "lib/text/motd": "welcome", "web/index.html": "web", "admin-ui/index.html": "admin"} {
		path := filepath.Join(bundle, "release-assets", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(body))
		files[name] = hex.EncodeToString(sum[:])
	}
	data, err := json.Marshal(Manifest{Format: 1, Version: "test", Commit: "test", Files: files})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "release-assets", "manifest.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestInitPreservesInstanceAndSeparatesRelease(t *testing.T) {
	bundle := fixture(t)
	instance := filepath.Join(t.TempDir(), "world with spaces")
	if err := Init(bundle, instance, 4350); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(instance)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.JWTSecret) != 64 || cfg.HTTPPort != 4350 {
		t.Fatal("missing stable instance configuration")
	}
	path := filepath.Join(instance, "lib", "world", "wld", "0.wld")
	if err := os.WriteFile(path, []byte("operator edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Init(bundle, instance, 4351); err == nil {
		t.Fatal("second init overwrote an existing instance")
	}
	after, err := Load(instance)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, cfg) {
		t.Fatal("second init changed signing secret or configuration")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "operator edit" {
		t.Fatal("operator world edit overwritten")
	}
	data, err = os.ReadFile(filepath.Join(bundle, "release-assets", "lib", "world", "wld", "0.wld"))
	if err != nil || string(data) != "world" {
		t.Fatal("release starter mutated")
	}
	if err := Init(bundle, filepath.Join(bundle, "mutable-instance"), 4350); err == nil {
		t.Fatal("instance accepted inside replaceable release")
	}
	other := filepath.Join(t.TempDir(), "second")
	if err := Init(bundle, other, 4350); err != nil {
		t.Fatal(err)
	}
	second, err := Load(other)
	if err != nil {
		t.Fatal(err)
	}
	if second.JWTSecret == cfg.JWTSecret {
		t.Fatal("instances reused a signing secret")
	}
}

func TestDamagedAndUnsafeBundleRefusedBeforeInit(t *testing.T) {
	for _, kind := range []string{"damaged", "traversal", "missing-web"} {
		t.Run(kind, func(t *testing.T) {
			bundle := fixture(t)
			m, err := ReadManifest(bundle)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "damaged":
				if err := os.WriteFile(filepath.Join(bundle, "release-assets", "web", "index.html"), []byte("changed"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "traversal":
				m.Files["../outside"] = "bad"
			case "missing-web":
				delete(m.Files, "web/index.html")
			}
			data, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bundle, "release-assets", "manifest.json"), data, 0o600); err != nil {
				t.Fatal(err)
			}
			instance := filepath.Join(t.TempDir(), "new")
			if err := Init(bundle, instance, 4350); err == nil {
				t.Fatal("unsafe/incomplete release accepted")
			}
			if _, err := os.Stat(instance); !os.IsNotExist(err) {
				t.Fatal("invalid release created an instance")
			}
		})
	}
}

func TestInstanceDoesNotInheritOtherServerState(t *testing.T) {
	cfg := Config{JWTSecret: strings.Repeat("a", 64)}
	inherited := []string{"PATH=tools", "JWT_SECRET=foreign", "DP_ALLOW_NO_DB=1", "DP_NOTIFICATION_KEY_PATH=foreign", "DP_FRESH_MUD=1", "DP_CLOCK=1", "DATABASE_URL=foreign", "ADMIN_STORE_PATH=foreign", "TLS_CERT_FILE=foreign", "CONTACT_SMTP_HOST=explicit"}
	env := instanceEnv(inherited, "release", "instance", cfg)
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "foreign") || strings.Contains(joined, "DP_CLOCK=") || strings.Contains(joined, "DP_FRESH_MUD=") {
		t.Fatal("foreign state/test controls leaked into portable instance")
	}
	if !strings.Contains(joined, "PATH=tools") || !strings.Contains(joined, "JWT_SECRET="+cfg.JWTSecret) || !strings.Contains(joined, "DP_NOTIFICATION_KEY_PATH="+filepath.Join("instance", "lib", "data", "notifications.json.key")) {
		t.Fatal("instance environment missing owned paths or signing secret")
	}
}

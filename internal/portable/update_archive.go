package portable

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/internal/releaseauth"
)

const (
	maxArchive  = 512 << 20
	maxExpanded = 2 << 30
	releaseAPI  = "https://api.github.com/repos/zax0rz/darkpawns/releases/latest"
)

// releaseEndpoint returns DP_RELEASE_ENDPOINT when it names a valid http(s)
// base, else "". It points the updater at a private mirror or a local
// rehearsal feed; the signed publisher manifest still gates every byte, so
// an overridden endpoint cannot deliver anything the pinned (or linked)
// publisher key did not sign. Empty path only — the download base below
// derives its own path segments.
func releaseEndpoint() string {
	endpoint := strings.TrimSpace(os.Getenv("DP_RELEASE_ENDPOINT"))
	if endpoint == "" {
		return ""
	}
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return ""
	}
	return strings.TrimSuffix(endpoint, "/")
}

// effectiveReleaseAPI is the release lookup endpoint in force.
func effectiveReleaseAPI() string {
	if endpoint := releaseEndpoint(); endpoint != "" {
		return endpoint + "/releases/latest"
	}
	return releaseAPI
}

// downloadBaseFor is where a release's assets are served from. A mirror
// serves them under its own origin; GitHub keeps its dedicated download host.
func downloadBaseFor(tag string) string {
	if endpoint := releaseEndpoint(); endpoint != "" {
		return endpoint + "/releases/download/" + tag + "/"
	}
	return "https://github.com/zax0rz/darkpawns/releases/download/" + tag + "/"
}

type releaseAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}
type githubRelease struct {
	Tag        string         `json:"tag_name"`
	HTMLURL    string         `json:"html_url"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	Assets     []releaseAsset `json:"assets"`
}

func updateClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || req.URL.Scheme != "https" || req.URL.User != nil {
			return errors.New("unsafe release redirect")
		}
		// L1-011: a redirect may only stay on the original host or move to
		// GitHub's asset CDN — a compromised source must not aim the updater
		// at an arbitrary HTTPS server.
		if req.URL.Host != via[0].URL.Host && !strings.HasSuffix(req.URL.Host, ".githubusercontent.com") {
			return errors.New("release redirect left the allowed hosts")
		}
		return nil
	}}
}

// fetchLatestRelease performs the read-only release lookup shared by staging
// and the console's update check. It downloads nothing.
func fetchLatestRelease(ctx context.Context, client *http.Client, endpoint string) (githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "darkpawns-updater")
	response, err := client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return githubRelease{}, errors.New("no published stable portable release is available yet; your world was not stopped")
	}
	if response.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("release lookup returned HTTP %d; your world was not stopped", response.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&release); err != nil {
		return githubRelease{}, err
	}
	return release, nil
}

func latestAsset(ctx context.Context, client *http.Client, endpoint string) (releaseAsset, error) {
	release, err := fetchLatestRelease(ctx, client, endpoint)
	if err != nil {
		return releaseAsset{}, err
	}
	return selectAsset(release)
}

func selectAsset(release githubRelease) (releaseAsset, error) {
	if release.Draft || release.Prerelease || !strings.HasPrefix(release.Tag, "v") || strings.ContainsAny(release.Tag, "/\\ \r\n") {
		return releaseAsset{}, errors.New("unsupported stable release metadata")
	}
	extension := ".tar.gz"
	if runtime.GOOS == "windows" {
		extension = ".zip"
	}
	name := "darkpawns_" + strings.TrimPrefix(release.Tag, "v") + "_" + runtime.GOOS + "_" + runtime.GOARCH + extension
	var found []releaseAsset
	for _, asset := range release.Assets {
		if asset.Name == name {
			found = append(found, asset)
		}
	}
	if len(found) != 1 {
		return releaseAsset{}, errors.New("release has no unique portable archive for this platform")
	}
	asset := found[0]
	u, err := url.Parse(asset.URL)
	prefix := downloadBaseFor(release.Tag)
	// Plain http is only ever acceptable to a loopback mirror (a local
	// rehearsal feed); anything remote must be https.
	loopbackMirror := u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1")
	if err != nil || (u.Scheme != "https" && !loopbackMirror) || u.User != nil || !strings.HasPrefix(asset.URL, prefix) || u.RawQuery != "" || u.Fragment != "" || !strings.HasSuffix(asset.URL, "/"+name) || asset.Size <= 0 || asset.Size > maxArchive {
		return asset, errors.New("invalid official release asset")
	}
	if !strings.HasPrefix(asset.Digest, "sha256:") {
		return asset, errors.New("release asset has no GitHub SHA-256 digest")
	}
	if err := validDigest(strings.TrimPrefix(asset.Digest, "sha256:")); err != nil {
		return asset, err
	}
	return asset, nil
}

func validDigest(digest string) error {
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != sha256.Size {
		return errors.New("a complete SHA-256 digest is required")
	}
	return nil
}

// Fetch into private staging before trusting any archive entries or stopping a game.
func fetchUpdate(ctx context.Context, archive, digest, destination string) (string, error) {
	if archive != "" {
		if err := validDigest(digest); err != nil {
			return "", err
		}
	}
	var source io.ReadCloser
	if archive == "" {
		client := updateClient()
		defer client.CloseIdleConnections()
		asset, err := latestAsset(ctx, client, effectiveReleaseAPI())
		if err != nil {
			return "", err
		}
		// TrustedKeyBytes() equals the pinned production key in published
		// builds; snapshot-linked builds verify against their linked key so
		// the whole pipeline stays exercisable. An override endpoint cannot
		// bypass this: whatever key is in force still has to have signed the
		// release manifest.
		if err := verifyPublishedAsset(ctx, client, asset, releaseauth.TrustedKeyBytes()); err != nil {
			return "", err
		}
		digest = strings.TrimPrefix(asset.Digest, "sha256:")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
		if err != nil {
			return "", err
		}
		response, err := client.Do(req)
		if err != nil {
			return "", err
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			return "", fmt.Errorf("archive download returned HTTP %d", response.StatusCode)
		}
		source = response.Body
	} else {
		f, err := os.Open(archive)
		if err != nil {
			return "", err
		}
		source = f
	}
	defer source.Close()
	f, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	count, copyErr := io.Copy(io.MultiWriter(f, hash), io.LimitReader(source, maxArchive+1))
	if err := errors.Join(copyErr, f.Sync(), f.Close()); err != nil {
		return "", err
	}
	if count > maxArchive {
		return "", errors.New("release archive exceeds size limit")
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(got, digest) {
		return "", errors.New("release archive SHA-256 mismatch; your world was not stopped")
	}
	return got, nil
}

// Archives have exactly one wrapper. Refuse links, special files, platform aliases,
// duplicates and expansion bombs before a payload can become active.
func extractUpdate(archive, destination string) (bundle string, result error) {
	file, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	roots := map[string]bool{}
	seen := map[string]bool{}
	var expanded int64
	var entries int
	write := func(name string, size int64, mode fs.FileMode, reader io.Reader) error {
		entries++
		if entries > 50000 || size < 0 || size > maxExpanded-expanded {
			return errors.New("release expansion limit exceeded")
		}
		expanded += size
		name = strings.TrimSuffix(name, "/")
		if !fs.ValidPath(name) || strings.ContainsAny(name, "\\:") {
			return fmt.Errorf("unsafe archive path %q", name)
		}
		parts := strings.Split(name, "/")
		for _, part := range parts {
			base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
			if strings.TrimRight(part, ". ") != part || base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9') {
				return errors.New("nonportable archive path")
			}
		}
		roots[parts[0]] = true
		if len(roots) != 1 {
			return errors.New("release archive must have one top-level directory")
		}
		key := strings.ToLower(name)
		if seen[key] {
			return errors.New("duplicate archive entry")
		}
		seen[key] = true
		target := filepath.Join(destination, filepath.FromSlash(name))
		if mode.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !mode.IsRegular() || len(parts) < 2 {
			return errors.New("release archive contains a link or special file")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		n, copyErr := io.Copy(output, io.LimitReader(reader, size+1))
		if err := errors.Join(copyErr, output.Sync(), output.Close()); err != nil {
			return err
		}
		if n != size {
			return errors.New("archive entry size mismatch")
		}
		if len(parts) == 2 && (parts[1] == "darkpawns" || parts[1] == "darkpawns-server") {
			return os.Chmod(target, 0o700)
		}
		return nil
	}
	if z, err := zip.NewReader(file, info.Size()); err == nil {
		for _, entry := range z.File {
			if entry.UncompressedSize64 > maxExpanded {
				return "", errors.New("release expansion limit exceeded")
			}
			r, err := entry.Open()
			if err != nil {
				return "", err
			}
			err = write(entry.Name, int64(entry.UncompressedSize64), entry.Mode(), r)
			closeErr := r.Close()
			if err := errors.Join(err, closeErr); err != nil {
				return "", err
			}
		}
	} else {
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		gz, err := gzip.NewReader(file)
		if err != nil {
			return "", err
		}
		defer func() { result = errors.Join(result, gz.Close()) }()
		tr := tar.NewReader(gz)
		for {
			entry, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			if entry.Typeflag != tar.TypeReg && entry.Typeflag != tar.TypeDir {
				return "", errors.New("release archive contains a link or special file")
			}
			if err := write(entry.Name, entry.Size, entry.FileInfo().Mode(), tr); err != nil {
				return "", err
			}
		}
	}
	if len(roots) != 1 {
		return "", errors.New("empty release archive")
	}
	for root := range roots {
		return filepath.Join(destination, root), nil
	}
	return "", errors.New("empty release archive")
}

func verifyUpdateBundle(bundle string) (Manifest, error) {
	manifest, err := ReadManifest(bundle)
	if err != nil {
		return manifest, err
	}
	if manifest.UpdateFormat != 1 || manifest.DataFormat != 1 {
		return manifest, errors.New("release does not declare compatible updater and data formats; migration is required")
	}
	if err := Verify(bundle, manifest); err != nil {
		return manifest, err
	}
	for _, name := range []string{"darkpawns", "darkpawns-server"} {
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		info, err := buildinfo.ReadFile(filepath.Join(bundle, name))
		if err != nil {
			return manifest, fmt.Errorf("read native %s: %w", name, err)
		}
		settings := map[string]string{}
		for _, s := range info.Settings {
			settings[s.Key] = s.Value
		}
		if settings["GOOS"] != runtime.GOOS || settings["GOARCH"] != runtime.GOARCH {
			return manifest, errors.New("release binaries are for another platform")
		}
	}
	return manifest, nil
}

func verifyPublishedAsset(ctx context.Context, client *http.Client, asset releaseAsset, key []byte) error {
	base := strings.TrimSuffix(asset.URL, asset.Name)
	fetch := func(name string, limit int64) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+name, nil)
		if err != nil {
			return nil, err
		}
		response, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("publisher metadata returned HTTP %d", response.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
		if err != nil {
			return nil, err
		}
		if int64(len(data)) > limit {
			return nil, errors.New("publisher metadata exceeds limit")
		}
		return data, nil
	}
	data, err := fetch("release-manifest.txt", 32768)
	if err != nil {
		return err
	}
	signature, err := fetch("release-manifest.sig", 1024)
	if err != nil {
		return err
	}
	manifest, err := releaseauth.Verify(data, signature, key)
	if err != nil {
		return err
	}
	expectedBase := downloadBaseFor(manifest.Tag)
	if base != expectedBase {
		return errors.New("signed release identity does not match the selected release")
	}
	signed, err := manifest.Asset(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	if signed.Name != asset.Name || signed.Size != asset.Size || "sha256:"+signed.SHA256 != asset.Digest {
		return errors.New("GitHub asset does not match the publisher's signed record")
	}
	return nil
}

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
)

const (
	maxArchive  = 512 << 20
	maxExpanded = 2 << 30
	releaseAPI  = "https://api.github.com/repos/zax0rz/darkpawns/releases/latest"
)

type releaseAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}
type githubRelease struct {
	Tag        string         `json:"tag_name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	Assets     []releaseAsset `json:"assets"`
}

func updateClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || req.URL.Scheme != "https" || req.URL.User != nil {
			return errors.New("unsafe release redirect")
		}
		return nil
	}}
}

func latestAsset(ctx context.Context, client *http.Client, endpoint string) (releaseAsset, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return releaseAsset{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "darkpawns-updater")
	response, err := client.Do(req)
	if err != nil {
		return releaseAsset{}, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return releaseAsset{}, errors.New("no published stable portable release is available yet; your world was not stopped")
	}
	if response.StatusCode != http.StatusOK {
		return releaseAsset{}, fmt.Errorf("release lookup returned HTTP %d; your world was not stopped", response.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&release); err != nil {
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
	prefix := "https://github.com/zax0rz/darkpawns/releases/download/" + release.Tag + "/"
	if err != nil || u.Scheme != "https" || u.User != nil || !strings.HasPrefix(asset.URL, prefix) || u.RawQuery != "" || u.Fragment != "" || !strings.HasSuffix(asset.URL, "/"+name) || asset.Size <= 0 || asset.Size > maxArchive {
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
		asset, err := latestAsset(ctx, client, releaseAPI)
		if err != nil {
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

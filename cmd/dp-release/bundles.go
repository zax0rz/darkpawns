package main

// Per-bundle publisher signatures (L1-006/007/021): every release archive
// carries release-assets/publisher.json — a publisher-signed hash list
// covering the executables at the bundle root and the data manifest itself.
// The executables were previously covered by no integrity check at all
// after extraction; now nothing in a bundle executes unless this signature
// verifies and every listed hash matches. manifest.json stays byte-identical
// so older launchers, which verify only the data manifest, can install and
// run this release during the update transition (the v0.1.3→v0.1.4 hop).
// The outer release manifest's sizes and digests are computed after this
// step by the caller, so they cover the repacked bytes.

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/zax0rz/darkpawns/internal/releaseauth"
)

type bundleManifest struct {
	Format       int               `json:"format"`
	UpdateFormat int               `json:"update_format,omitempty"`
	DataFormat   int               `json:"data_format,omitempty"`
	Version      string            `json:"version"`
	Commit       string            `json:"commit"`
	Files        map[string]string `json:"files"`
}

type publisherRecord struct {
	Format  int               `json:"format"`
	Version string            `json:"version"`
	Files   map[string]string `json:"files"`
}

// signBundlePublisher writes release-assets/publisher.json and its publisher
// signature: a hash list covering the bundle-root executables and the data
// manifest itself. manifest.json is left byte-identical.
func signBundlePublisher(bundle string, windows bool, private *rsa.PrivateKey, public []byte) error {
	manifestPath := filepath.Join(bundle, "release-assets", "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var m bundleManifest
	if err := json.Unmarshal(manifestData, &m); err != nil {
		return fmt.Errorf("parse %s: %w", manifestPath, err)
	}
	if m.Format == 0 || m.Version == "" || len(m.Files) == 0 {
		return errors.New("unsupported or incomplete bundle manifest: " + bundle)
	}
	suffix := ""
	if windows {
		suffix = ".exe"
	}
	record := publisherRecord{Format: 1, Version: m.Version, Files: map[string]string{
		"release-assets/manifest.json": hashBytes(manifestData),
	}}
	for _, name := range []string{"darkpawns" + suffix, "darkpawns-server" + suffix} {
		sum, err := hashFile(filepath.Join(bundle, name))
		if err != nil {
			return fmt.Errorf("bundle executable %s: %w", name, err)
		}
		record.Files[name] = sum
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(bundle, "release-assets", "publisher.json"), data, 0o644); err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, private, crypto.SHA256, digest[:])
	if err != nil {
		return err
	}
	if err := releaseauth.VerifyBytes(data, signature, public); err != nil {
		return fmt.Errorf("publisher record self-verification failed: %s: %w", filepath.Base(bundle), err)
	}
	return os.WriteFile(filepath.Join(bundle, "release-assets", "publisher.json.sig"), signature, 0o644)
}

// postProcessArchive adds the publisher signature to one goreleaser archive
// and repacks. Windows archives (.zip) carry .exe-suffixed executables.
func postProcessArchive(path string, private *rsa.PrivateKey, public []byte) error {
	windows := strings.HasSuffix(path, ".zip")
	root, err := os.MkdirTemp("", "dp-release-post-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(root) }()

	headers, order, err := extractArchive(path, filepath.Join(root, "x"))
	if err != nil {
		return err
	}
	bundles, err := filepath.Glob(filepath.Join(root, "x", "*"))
	if err != nil || len(bundles) != 1 {
		return fmt.Errorf("postProcessArchive %s: expected one wrapped bundle directory: %v", filepath.Base(path), bundles)
	}
	bundle := bundles[0]
	if err := signBundlePublisher(bundle, windows, private, public); err != nil {
		return err
	}
	for _, name := range []string{"publisher.json", "publisher.json.sig"} {
		headers[wrappedPath(root, filepath.Join(bundle, "release-assets", name))] = nil
	}
	return repackArchive(path, filepath.Join(root, "x"), headers, order)
}

// wrappedPath maps an extracted file path back to its archive entry name.
func wrappedPath(root, path string) string {
	return filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(path, filepath.Join(root, "x")), string(os.PathSeparator)))
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// extractArchive unpacks archive into destination, returning each entry's
// original header (keyed by its archive path, nil for entries the archive
// did not contain) and their original order, so repacking preserves modes
// and paths exactly.
func extractArchive(path, destination string) (map[string]any, []string, error) {
	headers := map[string]any{}
	var order []string
	if strings.HasSuffix(path, ".zip") {
		r, err := zip.OpenReader(path)
		if err != nil {
			return nil, nil, err
		}
		defer func() { _ = r.Close() }()
		for _, f := range r.File {
			headers[f.Name] = f.FileHeader
			order = append(order, f.Name)
			if err := extractZipEntry(f, destination); err != nil {
				return nil, nil, err
			}
		}
		return headers, order, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = gz.Close() }()
	r := tar.NewReader(gz)
	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		headers[h.Name] = *h
		order = append(order, h.Name)
		target := filepath.Join(destination, filepath.FromSlash(h.Name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(h.Mode)); err != nil {
				return nil, nil, err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, nil, err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(h.Mode))
			if err != nil {
				return nil, nil, err
			}
			if _, err := io.Copy(out, r); err != nil {
				_ = out.Close()
				return nil, nil, err
			}
			if err := out.Close(); err != nil {
				return nil, nil, err
			}
		default:
			return nil, nil, fmt.Errorf("unsupported archive entry %s (%q)", h.Name, string(h.Typeflag))
		}
	}
	return headers, order, nil
}

func extractZipEntry(f *zip.File, destination string) error {
	target := filepath.Join(destination, filepath.FromSlash(f.Name))
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, f.Mode())
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, f.Mode())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// repackArchive rewrites the archive from tree, preserving every original
// entry header (modes, times, order) plus any entry whose header is nil
// (newly added files). Sizes are recomputed from the bytes actually written.
func repackArchive(path, tree string, headers map[string]any, order []string) (result error) {
	temp := path + ".post"
	out, err := os.Create(temp)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			result = errors.Join(result, out.Close())
		}
	}()
	isZip := strings.HasSuffix(path, ".zip")
	var buffered *bytes.Buffer
	var writer io.Writer = out
	if isZip {
		buffered = &bytes.Buffer{}
		writer = buffered
	}
	zipWriter := zip.NewWriter(writer)
	var gzWriter *gzip.Writer
	var tarWriter *tar.Writer
	if isZip {
		zipWriter = zip.NewWriter(writer)
	} else {
		gzWriter = gzip.NewWriter(writer)
		tarWriter = tar.NewWriter(gzWriter)
	}
	wrote := map[string]bool{}
	entries := append([]string{}, order...)
	for name := range headers {
		if headers[name] == nil {
			entries = append(entries, name)
		}
	}
	for _, name := range entries {
		if wrote[name] {
			continue
		}
		wrote[name] = true
		source := filepath.Join(tree, filepath.FromSlash(name))
		info, err := os.Lstat(source)
		if err != nil {
			return err
		}
		var data []byte
		if info.Mode().IsRegular() {
			if data, err = os.ReadFile(source); err != nil {
				return err
			}
		}
		if isZip {
			var header zip.FileHeader
			if original, ok := headers[name].(zip.FileHeader); ok {
				header = original
			} else {
				header.Name = name
				header.SetMode(0o644)
			}
			// Recomputed by the writer as the bytes are written; stale
			// copied sizes would corrupt the archive.
			header.CompressedSize64, header.UncompressedSize64 = 0, 0
			header.CRC32 = 0
			entry, err := zipWriter.CreateHeader(&header)
			if err != nil {
				return err
			}
			if info.IsDir() {
				continue
			}
			if _, err := entry.Write(data); err != nil {
				return err
			}
		} else {
			var header tar.Header
			if original, ok := headers[name].(tar.Header); ok {
				header = original
			} else {
				header.Name = name
				header.Mode = 0o644
				header.Format = tar.FormatPAX
			}
			if info.IsDir() {
				header.Typeflag = tar.TypeDir
				header.Size = 0
			} else {
				header.Typeflag = tar.TypeReg
				header.Size = int64(len(data))
			}
			if err := tarWriter.WriteHeader(&header); err != nil {
				return err
			}
			if header.Typeflag == tar.TypeReg {
				if _, err := tarWriter.Write(data); err != nil {
					return err
				}
			}
		}
	}
	if isZip {
		if err := zipWriter.Close(); err != nil {
			return err
		}
		if _, err := out.Write(buffered.Bytes()); err != nil {
			return err
		}
	} else {
		if err := tarWriter.Close(); err != nil {
			return err
		}
		if err := gzWriter.Close(); err != nil {
			return err
		}
	}
	if err := out.Close(); err != nil {
		return err
	}
	closed = true
	return os.Rename(temp, path)
}

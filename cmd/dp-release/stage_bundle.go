package main

// The goreleaser build post-hook stages each platform's bundle contents —
// both executables plus the staged release-assets manifest — into
// dist/signed-bundles/<os>_<arch>/, which the archive file lists include per
// platform. When a signing key is configured the publisher record is signed
// over those contents here, so everything goreleaser computes afterwards
// (checksums, Homebrew, Scoop, winget, Docker) sees the final signed bytes;
// nothing is repacked after publication. A shell script cannot serve as the
// hook on Windows, so this lives in the Go tool.

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zax0rz/darkpawns/internal/releaseauth"
)

func stageBundleContents(goos, goarch string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	suffix := ""
	if goos == "windows" {
		suffix = ".exe"
	}
	out := filepath.Join(root, "dist", "signed-bundles", goos+"_"+goarch)
	binaries := make([]string, 0, 2)
	for _, name := range []string{"darkpawns", "darkpawns-server"} {
		matches, globErr := filepath.Glob(filepath.Join(root, "dist", "*_"+goos+"_"+goarch+"*", name+suffix))
		if globErr != nil || len(matches) == 0 {
			// Builds may complete in any order; the archive stage starts
			// only after every build finished, so the last hook call per
			// platform always finds both.
			fmt.Fprintf(os.Stderr, "stage-bundle: waiting for %s (%s/%s)\n", name, goos, goarch)
			return nil
		}
		binaries = append(binaries, matches[0])
	}
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := copyTree(filepath.Join(root, "release-assets"), filepath.Join(out, "release-assets")); err != nil {
		return err
	}
	for _, binary := range binaries {
		info, err := os.Stat(binary)
		if err != nil {
			return err
		}
		if err := copyFile(binary, filepath.Join(out, filepath.Base(binary)), info.Mode()); err != nil {
			return err
		}
	}
	// Sign the staged contents when a key is configured: the production
	// key (DP_RELEASE_KEY_PATH, verified against the pinned public key) or
	// the disposable CI pair; without either, dev snapshots stay unsigned.
	keyPath := os.Getenv("DP_RELEASE_KEY_PATH")
	publicPath := os.Getenv("SNAPSHOT_PUBLIC_PEM")
	if keyPath == "" && os.Getenv("SNAPSHOT_PRIVATE_PEM") != "" && publicPath != "" {
		keyPath = os.Getenv("SNAPSHOT_PRIVATE_PEM")
	}
	if keyPath == "" {
		fmt.Fprintf(os.Stderr, "stage-bundle: no signing key configured; %s_%s staged unsigned (dev snapshot)\n", goos, goarch)
		return nil
	}
	private, public, err := loadSigningKeyPair(keyPath, publicPath)
	if err != nil {
		return err
	}
	if err := signBundlePublisher(out, goos == "windows", private, public); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "stage-bundle: staged and signed %s_%s\n", goos, goarch)
	return nil
}

// loadSigningKeyPair loads the private signing key and its public PEM. A
// production signing (no public path) must match the publisher key pinned in
// the launcher; the snapshot pair is trusted because the build linked it.
func loadSigningKeyPair(keyPath, publicPath string) (*rsa.PrivateKey, []byte, error) {
	public := releaseauth.TrustedKey
	if publicPath != "" {
		data, err := os.ReadFile(publicPath) // #nosec G304 -- operator-configured path
		if err != nil {
			return nil, nil, err
		}
		public = data
	}
	pinned, err := releaseauth.PublicKey(public)
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(keyPath) // #nosec G304 -- operator-configured path
	if err != nil {
		return nil, nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, nil, errors.New("invalid private key")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	private, ok := parsed.(*rsa.PrivateKey)
	if !ok || private.N.Cmp(pinned.N) != 0 || private.E != pinned.E {
		return nil, nil, errors.New("signing key does not match the public key")
	}
	return private, public, nil
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return errors.New("non-regular release asset: " + path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(source, target string, mode os.FileMode) error {
	in, err := os.Open(source) // #nosec G304 -- release staging paths, never request-derived
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode) // #nosec G304 -- release staging paths
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	return errors.Join(copyErr, closeErr)
}

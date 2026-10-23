// Package releaseauth authenticates publisher metadata before portable code runs.
package releaseauth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

//go:embed trusted-key.pem
var TrustedKey []byte

// linkedKeyOverride is empty in published builds. Snapshot and test builds
// link a disposable publisher key here (via -ldflags -X, fed from a
// generated key in CI) so the strict bundle-signature gate can be exercised
// end-to-end without the private release key. It cannot be changed at
// runtime: the value is fixed when the binary is linked.
var linkedKeyOverride string

// TrustedKeyBytes is the pinned publisher key bundle signatures are verified
// against. The linked override may be the PEM text or its base64 (ldflags -X
// values are single-line).
func TrustedKeyBytes() []byte {
	if linkedKeyOverride != "" {
		if !strings.Contains(linkedKeyOverride, "BEGIN") {
			if decoded, err := base64.StdEncoding.DecodeString(linkedKeyOverride); err == nil {
				return decoded
			}
		}
		return []byte(linkedKeyOverride)
	}
	return TrustedKey
}

// VerifyBytes authenticates publisher-signed data that is not a release
// manifest — the per-file bundle manifest inside a release archive — against
// the publisher key.
func VerifyBytes(data, signature, key []byte) error {
	if len(data) > 8<<20 || len(signature) > 1024 {
		return errors.New("signed data exceeds limit")
	}
	public, err := PublicKey(key)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	if err := rsa.VerifyPKCS1v15(public, crypto.SHA256, hash[:], signature); err != nil {
		return errors.New("publisher signature verification failed")
	}
	return nil
}

var (
	StableTag     = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	commitPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)
)

type (
	Asset struct {
		OS, Arch, Name, SHA256 string
		Size                   int64
	}
	Manifest struct {
		Tag, Commit string
		Assets      []Asset
	}
)

func PublicKey(data []byte) (*rsa.PublicKey, error) {
	block, rest := pem.Decode(data)
	if block == nil || len(rest) != 0 {
		return nil, errors.New("invalid publisher public key")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok || rsaKey.N.BitLen() < 2048 {
		return nil, errors.New("unsupported publisher key")
	}
	return rsaKey, nil
}

func Verify(data, signature, key []byte) (Manifest, error) {
	var m Manifest
	if len(data) > 32768 || len(signature) > 1024 {
		return m, errors.New("release metadata exceeds limit")
	}
	public, err := PublicKey(key)
	if err != nil {
		return m, err
	}
	hash := sha256.Sum256(data)
	if err := rsa.VerifyPKCS1v15(public, crypto.SHA256, hash[:], signature); err != nil {
		return m, errors.New("publisher signature verification failed")
	}
	return Parse(data)
}

func Parse(data []byte) (Manifest, error) {
	var m Manifest
	lines := strings.Split(string(data), "\n")
	if len(lines) < 5 || lines[0] != "darkpawns-release-v1" || lines[len(lines)-1] != "" {
		return m, errors.New("invalid signed release format")
	}
	if !strings.HasPrefix(lines[1], "version ") || !strings.HasPrefix(lines[2], "commit ") {
		return m, errors.New("missing release identity")
	}
	m.Tag = strings.TrimPrefix(lines[1], "version ")
	m.Commit = strings.TrimPrefix(lines[2], "commit ")
	if !StableTag.MatchString(m.Tag) || !commitPattern.MatchString(m.Commit) {
		return m, errors.New("invalid stable release identity")
	}
	seen := map[string]bool{}
	for _, line := range lines[3 : len(lines)-1] {
		fields := strings.Split(line, " ")
		if len(fields) != 7 || fields[0] != "asset" {
			return m, errors.New("invalid signed asset record")
		}
		a := Asset{OS: fields[1], Arch: fields[2], Name: fields[3], SHA256: fields[6]}
		size, err := strconv.ParseInt(fields[4], 10, 64)
		if err != nil || size < 1 || size > 512<<20 || fields[5] != "sha256" {
			return m, errors.New("invalid signed asset size/digest")
		}
		a.Size = size
		sum, err := hex.DecodeString(a.SHA256)
		if err != nil || len(sum) != sha256.Size || strings.ToLower(a.SHA256) != a.SHA256 {
			return m, errors.New("invalid signed SHA-256")
		}
		expected, err := ArchiveName(m.Tag, a.OS, a.Arch)
		if err != nil {
			return m, err
		}
		if a.Name != expected || seen[a.Name] {
			return m, errors.New("invalid or duplicate signed asset")
		}
		seen[a.Name] = true
		m.Assets = append(m.Assets, a)
	}
	return m, nil
}

func ArchiveName(tag, platform, arch string) (string, error) {
	if !StableTag.MatchString(tag) {
		return "", errors.New("a stable vMAJOR.MINOR.PATCH tag is required")
	}
	if platform == "any" {
		if arch == "sh" {
			return "install.sh", nil
		}
		if arch == "powershell" {
			return "install.ps1", nil
		}
	}
	if (platform != "linux" && platform != "darwin" && platform != "windows") || (arch != "amd64" && arch != "arm64") {
		return "", errors.New("unsupported release platform")
	}
	ext := ".tar.gz"
	if platform == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("darkpawns_%s_%s_%s%s", strings.TrimPrefix(tag, "v"), platform, arch, ext), nil
}

func (m Manifest) Asset(platform, arch string) (Asset, error) {
	for _, a := range m.Assets {
		if a.OS == platform && a.Arch == arch {
			return a, nil
		}
	}
	return Asset{}, errors.New("signed release has no asset for this platform")
}

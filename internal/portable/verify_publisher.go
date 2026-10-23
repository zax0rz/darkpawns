package portable

// Strict publisher authentication for on-disk bundles (L1-006/007/021):
// release-assets/publisher.json is a publisher-signed hash list covering
// the executables at the bundle root and the data manifest itself, and
// nothing inside the instance tree can forge or substitute it — a writable
// active pointer, a forged recovery journal, or a self-consistent planted
// data manifest all fail here. Strict mode: a bundle without a valid
// publisher signature never executes. Older bundles (v0.1.3 and earlier)
// carry no publisher record and are refused; rolling back to them means
// installing an older launcher.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zax0rz/darkpawns/internal/releaseauth"
)

// ErrUnsignedBundle is the strict-mode refusal for bundles that carry no
// publisher record.
var ErrUnsignedBundle = errors.New("release carries no publisher signature; strict mode refuses to run an unsigned bundle")

type publisherRecord struct {
	Format  int               `json:"format"`
	Version string            `json:"version"`
	Files   map[string]string `json:"files"`
}

// VerifyBundlePublisher proves a bundle's contents are the publisher's
// before anything in it is executed: the data manifest's checks, then the
// publisher record's signature and every hash it lists.
func VerifyBundlePublisher(bundle string) error {
	return verifyBundlePublisherWithKey(bundle, releaseauth.TrustedKeyBytes())
}

// PublisherVerify is the bundle gate every exec path calls; it is a
// variable so tests can substitute a disposable publisher key (the private
// release key never exists outside the release environment).
var PublisherVerify = VerifyBundlePublisher

func verifyBundlePublisherWithKey(bundle string, key []byte) (result error) {
	m, err := ReadManifest(bundle)
	if err != nil {
		return err
	}
	if err := Verify(bundle, m); err != nil {
		return err
	}
	recordPath := filepath.Join(bundle, "release-assets", "publisher.json")
	data, err := os.ReadFile(recordPath)
	if err != nil {
		return ErrUnsignedBundle
	}
	signature, err := os.ReadFile(recordPath + ".sig")
	if err != nil {
		return ErrUnsignedBundle
	}
	if err := releaseauth.VerifyBytes(data, signature, key); err != nil {
		return fmt.Errorf("publisher signature for %s: %w", filepath.Base(bundle), err)
	}
	var record publisherRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return fmt.Errorf("parse publisher record: %w", err)
	}
	if record.Format != 1 || record.Version != m.Version || len(record.Files) == 0 {
		return errors.New("unsupported or incomplete publisher record")
	}
	for name, want := range record.Files {
		// Entries are bundle-relative: the executables at the root, the data
		// manifest under release-assets.
		f, err := os.Open(filepath.Join(bundle, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("publisher record file %s: %w", name, err)
		}
		sum := sha256.New()
		_, copyErr := io.Copy(sum, f)
		closeErr := f.Close()
		if err := errors.Join(copyErr, closeErr); err != nil {
			return err
		}
		if hex.EncodeToString(sum.Sum(nil)) != want {
			return fmt.Errorf("publisher checksum mismatch: %s", name)
		}
	}
	return nil
}

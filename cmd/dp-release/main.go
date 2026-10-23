// dp-release is maintainer tooling; no private key belongs in a portable bundle.
package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/zax0rz/darkpawns/internal/releaseauth"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "dp-release:", err)
		os.Exit(1)
	}
}

func run() (result error) {
	flags := flag.NewFlagSet("dp-release", flag.ContinueOnError)
	generate := flags.String("generate-key", "", "Create a new private RSA key; requires --public output path")
	dir := flags.String("dir", "dist", "Release output")
	tag := flags.String("tag", "", "Stable tag")
	commit := flags.String("commit", "", "Full commit")
	keyPath := flags.String("key", "", "Private PEM for signing")
	publicPath := flags.String("public", "", "Alternate public PEM for disposable tests")
	templates := flags.String("templates", "bootstrap", "Installer template directory")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if *generate != "" {
		if *publicPath == "" {
			return errors.New("--public output path is required")
		}
		private, err := rsa.GenerateKey(rand.Reader, 3072)
		if err != nil {
			return err
		}
		secret, err := x509.MarshalPKCS8PrivateKey(private)
		if err != nil {
			return err
		}
		public, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
		if err != nil {
			return err
		}
		write := func(path string, data []byte) error {
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			_, writeErr := file.Write(data)
			return errors.Join(writeErr, file.Sync(), file.Close())
		}
		if err := write(*generate, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: secret})); err != nil {
			return err
		}
		return write(*publicPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}))
	}
	public := releaseauth.TrustedKey
	if *publicPath != "" {
		data, err := os.ReadFile(*publicPath)
		if err != nil {
			return err
		}
		public = data
	}
	key, err := releaseauth.PublicKey(public)
	if err != nil {
		return err
	}
	if *keyPath == "" {
		return errors.New("--key is required; pass it from the protected release environment")
	}
	data, err := os.ReadFile(*keyPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return errors.New("invalid private key")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return err
	}
	private, ok := parsed.(*rsa.PrivateKey)
	if !ok || private.N.Cmp(key.N) != 0 || private.E != key.E {
		return errors.New("signing key does not match the pinned public key")
	}
	outputRoot, err := os.OpenRoot(*dir)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, outputRoot.Close()) }()
	modulus := base64.StdEncoding.EncodeToString(key.N.Bytes())
	exponent := base64.StdEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
	for _, name := range []string{"install.sh", "install.ps1"} {
		body, err := os.ReadFile(filepath.Join(*templates, name+".in"))
		if err != nil {
			return err
		}
		text := strings.ReplaceAll(string(body), "@@PUBLIC_KEY_PEM@@", strings.TrimSpace(string(public)))
		text = strings.ReplaceAll(text, "@@RSA_MODULUS@@", modulus)
		text = strings.ReplaceAll(text, "@@RSA_EXPONENT@@", exponent)
		if err := outputRoot.WriteFile(name, []byte(text), 0o600); err != nil {
			return err
		}
		// Sign each rendered installer (L1-003): the public entry stubs verify
		// <name>.sig — published beside the installer — before executing it.
		digest := sha256.Sum256([]byte(text))
		signature, err := rsa.SignPKCS1v15(rand.Reader, private, crypto.SHA256, digest[:])
		if err != nil {
			return err
		}
		if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
			return fmt.Errorf("installer signature self-verification failed: %s", name)
		}
		if err := outputRoot.WriteFile(name+".sig", signature, 0o600); err != nil {
			return err
		}
	}
	manifest := fmt.Sprintf("darkpawns-release-v1\nversion %s\ncommit %s\n", *tag, *commit)
	for _, platform := range []string{"linux", "darwin", "windows", "any"} {
		arches := []string{"amd64", "arm64"}
		if platform == "any" {
			arches = []string{"sh", "powershell"}
		}
		for _, arch := range arches {
			name, err := releaseauth.ArchiveName(*tag, platform, arch)
			if err != nil {
				return err
			}
			file, err := outputRoot.Open(name)
			if err != nil {
				return err
			}
			hash := sha256.New()
			size, copyErr := io.Copy(hash, file)
			if err := errors.Join(copyErr, file.Close()); err != nil {
				return err
			}
			manifest += fmt.Sprintf("asset %s %s %s %d sha256 %s\n", platform, arch, name, size, hex.EncodeToString(hash.Sum(nil)))
		}
	}
	if _, err := releaseauth.Parse([]byte(manifest)); err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(manifest))
	signature, err := rsa.SignPKCS1v15(rand.Reader, private, crypto.SHA256, hash[:])
	if err != nil {
		return err
	}
	if _, err := releaseauth.Verify([]byte(manifest), signature, public); err != nil {
		return err
	}
	if err := outputRoot.WriteFile("release-manifest.txt", []byte(manifest), 0o600); err != nil {
		return err
	}
	return outputRoot.WriteFile("release-manifest.sig", signature, 0o600)
}

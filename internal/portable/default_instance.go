package portable

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// DefaultInstance is shared by command dispatch and printed operator commands.
func DefaultInstance() (string, error) {
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", errors.New("LOCALAPPDATA is unset; choose a folder with --instance")
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "Library", "Application Support")
	default:
		base = os.Getenv("XDG_DATA_HOME")
		if base != "" && !filepath.IsAbs(base) {
			return "", errors.New("XDG_DATA_HOME must be absolute; choose a folder with --instance")
		}
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".local", "share")
		}
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("default data folder must be absolute; choose a folder with --instance")
	}
	return filepath.Join(base, "darkpawns", "world"), nil
}

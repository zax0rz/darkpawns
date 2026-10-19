package session

import "os"

// Windows has no POSIX real-uid access(R_OK). Probe read access using a handle;
// as in src/file-edit.c:19-25, a failed probe succeeds without removing anything.
// Unix fidelity continues to use access(R_OK) in the platform-specific file.
func killEditorFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Remove(path)
}

package portable

import "golang.org/x/sys/windows"

func replaceUpdateRecord(from, to string) error {
	source, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	destination, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(source, destination, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

// Windows file flushes and write-through metadata moves are used; opening a directory for fsync is not portable.
func syncUpdateDirectory(_ string) error { return nil }

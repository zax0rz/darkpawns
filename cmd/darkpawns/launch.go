package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// runLauncher owns the console lifetime as well as the exit status. In a
// terminal, the caller already owns its console; only Explorer needs a pause.
// Keep this outside execute so errors are printed before waiting, including
// errors returned while dispatching to an updated release.
func runLauncher(args []string, explorer bool, run func([]string) error, in io.Reader, out, errOut io.Writer) int {
	// Our setup and explicit acknowledgement replace Cobra's timed Windows exit.
	cobra.MousetrapHelpText = ""
	if explorer && len(args) == 0 {
		args = []string{"setup"}
	}
	var result error
	if err := run(args); err != nil {
		_, writeErr := fmt.Fprintln(errOut, "darkpawns:", err)
		result = errors.Join(err, writeErr)
	}
	if explorer {
		if err := holdExplorerConsole(in, out); err != nil {
			_, writeErr := fmt.Fprintln(errOut, "darkpawns: wait for console input:", err)
			result = errors.Join(result, err, writeErr)
		}
	}
	if result != nil {
		return 1
	}
	return 0
}

func holdExplorerConsole(in io.Reader, out io.Writer) error {
	if _, err := fmt.Fprint(out, "\nPress Enter to close this window..."); err != nil {
		return err
	}
	_, err := bufio.NewReader(in).ReadString('\n')
	// EOF means the console/input has already closed; never spin or hang trying
	// to keep an unavailable window open. Ctrl+C and closing the window remain
	// normal ways to leave it.
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

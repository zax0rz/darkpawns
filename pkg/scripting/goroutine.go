package scripting

import (
	"bytes"
	"runtime"
	"strconv"
)

// goroutineID is the running goroutine's number, from the header line of its
// stack trace ("goroutine 42 [running]:"). RunScript uses it only to tell a
// nested run (a script reaching another script through a binding, on the
// same goroutine) from a run on another goroutine, which must wait for the
// engine lock.
func goroutineID() uint64 {
	var buf [64]byte
	b := buf[:runtime.Stack(buf[:], false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	if i := bytes.IndexByte(b, ' '); i >= 0 {
		b = b[:i]
	}
	id, err := strconv.ParseUint(string(b), 10, 64)
	if err != nil {
		panic("scripting: cannot parse goroutine id: " + err.Error())
	}
	return id
}

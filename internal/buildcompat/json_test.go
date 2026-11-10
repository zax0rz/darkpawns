package buildcompat

import (
	"encoding/json"
	"testing"
)

// R1/R5h: Go 1.27's default JSON encoder changes invalid-UTF-8 bytes.
// Every test invocation must retain the v1 implementation, including direct
// go test outside Make. This intentionally fails without nojsonv2.
func TestJSONV1Bytes(t *testing.T) {
	got, err := json.Marshal(string([]byte{0xff}))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `"\ufffd"` {
		t.Fatalf("encoding/json emitted %q; builds require GOEXPERIMENT=nojsonv2 (R1)", got)
	}
}

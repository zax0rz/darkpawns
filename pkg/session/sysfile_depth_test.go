package session

import "testing"

func TestSysfileNameMirrorsCIsAbbrev(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  string
		want string
		ok   bool
	}{
		{name: "empty", arg: "", ok: false},
		{name: "bugs prefix", arg: "b", want: "bugs", ok: true},
		{name: "ideas prefix", arg: "i", want: "ideas", ok: true},
		{name: "todo prefix", arg: "t", want: "todo", ok: true},
		{name: "typos prefix", arg: "ty", want: "typos", ok: true},
		{name: "case insensitive", arg: "BUGS", want: "bugs", ok: true},
		{name: "unknown", arg: "nope", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sysfileName(tt.arg)
			if got != tt.want || ok != tt.ok {
				t.Errorf("sysfileName(%q) = (%q, %t), want (%q, %t)", tt.arg, got, ok, tt.want, tt.ok)
			}
		})
	}
}

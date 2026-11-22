package main

import (
	"slices"
	"strings"
	"testing"
)

func TestServerEnvSQLiteSynchronous(t *testing.T) {
	goEnv := goServerEnv(nil, "1", "1770838461", false, "", false)
	oracleEnv := oracleServerEnv(nil, "1", "1770838461")

	if !slices.Contains(goEnv, "DP_SQLITE_SYNCHRONOUS=NORMAL") {
		t.Fatalf("goServerEnv missing DP_SQLITE_SYNCHRONOUS=NORMAL: %v", goEnv)
	}
	for _, entry := range oracleEnv {
		if strings.HasPrefix(entry, "DP_SQLITE_SYNCHRONOUS=") {
			t.Fatalf("oracleServerEnv must not set DP_SQLITE_SYNCHRONOUS, got %q", entry)
		}
	}
}

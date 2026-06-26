package web

import (
	"os"
	"testing"
)

func TestGetAllowedOrigins_TrimsWhitespaceAndSkipsEmpty(t *testing.T) {
	orig := os.Getenv("CORS_ALLOWED_ORIGINS")
	defer os.Setenv("CORS_ALLOWED_ORIGINS", orig)

	os.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example, https://b.example,, https://c.example ")
	got := getAllowedOrigins()
	want := []string{"https://a.example", "https://b.example", "https://c.example"}
	if len(got) != len(want) {
		t.Fatalf("getAllowedOrigins() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("origin[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsOriginAllowed_WithTrimmedOrigins(t *testing.T) {
	origins := []string{"https://a.example", "https://b.example"}
	if !isOriginAllowed("https://b.example", origins) {
		t.Error("isOriginAllowed should accept origin after trimming")
	}
}

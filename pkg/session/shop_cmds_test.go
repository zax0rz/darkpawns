package session

import "testing"

func TestShopListKeywordMatchesCIsNameBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		keyword  string
		keywords string
		want     bool
	}{
		{name: "complete token", keyword: "pepper", keywords: "black pepper spice", want: true},
		{name: "middle of token is not abbreviation", keyword: "pep", keywords: "black pepper spice", want: false},
		{name: "final token remains exact", keyword: "cinnamon", keywords: "cinnamon spice", want: true},
		{name: "case insensitive", keyword: "PEPPER", keywords: "black pepper spice", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shopListKeywordMatches(tt.keyword, tt.keywords); got != tt.want {
				t.Fatalf("shopListKeywordMatches(%q, %q) = %t, want %t", tt.keyword, tt.keywords, got, tt.want)
			}
		})
	}
}

package crawler

import "testing"

func TestIsRelevantURL(t *testing.T) {
	tests := []struct {
		url        string
		isRelevant bool
	}{
		// Language-specific URLs (should be irrelevant)
		{"https://www.amazon.com/-/es/s?k=lego", false},
		{"https://www.amazon.com/-/de/b?node=165793011", false},
		{"https://www.amazon.com/-/fr/b?node=165793011", false},

		// Search URLs
		{"https://www.amazon.com/s?k=lego", true},
		{"https://www.amazon.com/s/toys", true},

		// Category URLs
		{"https://www.amazon.com/b?node=165793011", true},
		{"https://www.amazon.com/b/toys", true},

		// Amazon video should be excluded
		{"https://www.amazon.com/Amazon-Video/b?node=2858778011", false},

		// Irrelevant URLs
		{"https://www.amazon.com/gp/help/customer/display.html", false},
		{"https://www.amazon.com/gp/cart/view.html", false},
	}

	for _, test := range tests {
		result := isRelevantURL(test.url)
		if result != test.isRelevant {
			t.Errorf("isRelevantURL(%q) = %v; want %v", test.url, result, test.isRelevant)
		}
	}
}

func TestFilterQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Removes /ref=... in path and keeps allowed param",
			input:    "https://amazon.com/b/ref=SHCC/?node=23528055011",
			expected: "https://amazon.com/b?node=23528055011",
		},
		{
			name:     "Keeps allowed query param, strips ref from path",
			input:    "https://amazon.com/some/ref=abc123/path?node=123&bad=1",
			expected: "https://amazon.com/some/path?node=123",
		},
		{
			name:     "Removes /ref=... with allowed 'k' param",
			input:    "https://amazon.com/search/ref=something?k=headphones&foo=bar",
			expected: "https://amazon.com/search?k=headphones",
		},
		{
			name:     "Removes trailing /ref=... with no query params",
			input:    "https://amazon.com/b/ref=SHCC/",
			expected: "https://amazon.com/b",
		},
		{
			name:     "Keeps multiple allowed query params",
			input:    "https://amazon.com/s?node=123&k=ipad&junk=1",
			expected: "https://amazon.com/s?k=ipad&node=123",
		},
		{
			name:     "No path ref, no query params",
			input:    "https://amazon.com/dp/B08N5WRWNW",
			expected: "https://amazon.com/dp/B08N5WRWNW",
		},
		{
			name:     "Removes /ref-",
			input:    "https://amazon.com/ref-GC_AGCLP_Congrats_SUB/s/?bbn=2973109011&i=gift-cards",
			expected: "https://amazon.com/s?bbn=2973109011&i=gift-cards",
		},
		{
			name:     "Empty input string",
			input:    "",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterQueryParams(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

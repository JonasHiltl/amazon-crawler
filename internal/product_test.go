package internal

import (
	"fmt"
	"testing"
)

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"0", 0, false},
		{"1K", 1000, false},
		{"1k", 1000, false},
		{"1.5k", 1500, false},
		{"1,5k", 1500, false},
		{"1.52k", 1520, false},
		{"1,52k", 1520, false},
		{"1521", 1521, false},
		{"1M", 1000000, false},
		{"1m", 1000000, false},
		{"1.5m", 1500000, false},
		{"1,5M", 1500000, false},
		{"1.502M", 1502000, false},
		{"663,088", 663088, false},
		{"663.088", 663088, false},
		{"M123", 0, true},
		{"abc", 0, true},
		{"ab,123", 0, true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("should prase %s", test.input), func(t *testing.T) {
			got, err := parseInt(test.input)
			gotErr := err != nil
			if test.hasError != gotErr {
				t.Errorf("unexpected error for %q, got %v", test.input, err)
			}

			if got != test.expected {
				t.Errorf("for %q: expected %d, got %d", test.input, test.expected, got)
			}
		})
	}
}

func TestGetAsinFromURL(t *testing.T) {
	tests := []struct {
		url      string
		asin     string
		hasError bool
	}{
		{"/dp/B0D6PQDNQS", "B0D6PQDNQS", false},
		{"/tonies-Simba-Figurine-Disneys-Lion/dp/1250365945/ref=test", "1250365945", false},
		{"/Tonies-Wizzle-Audio-Character-Doggyland/dp/1989599834", "1989599834", false},
		{"https://amazon.com/super-nice-book/dp/B0D6PQDNQS", "B0D6PQDNQS", false},
		{"https://www.amazon.com/dp/B07984JN3L", "B07984JN3L", false},
		{"https://www.amazon.com/dp/B0DK7B7G9R", "B0DK7B7G9R", false},
		{"/sspa/click?url=%2FCoogam-Educational%2Fdp%2FB09Q82N7DN%3Fpsc%3D1", "B09Q82N7DN", false},
		{"/dp/", "", true},
		{"https://amazon.com/super-nice-book", "", true},
		{"https://amazon.com", "", true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("should get asin from %s", test.url), func(t *testing.T) {
			got, err := AsinFromURL(test.url)
			gotErr := err != nil
			if test.hasError != gotErr {
				t.Errorf("unexpected error: %v", err)
			}

			if got != test.asin {
				t.Errorf("expected %s, got %s", test.asin, got)
			}
		})
	}
}

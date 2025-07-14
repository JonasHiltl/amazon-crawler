package crawler

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

func isRelevantURL(url string) bool {
	// lanaguage specific urls like /-/es/
	re := regexp.MustCompile(`/-/[a-z]{2}/`)
	if re.MatchString(url) {
		return false
	}

	return isSearchURL(url) || isCategoryURL(url)
}

func isSearchURL(url string) bool {
	contains := []string{"/s?", "/s/"}
	for _, c := range contains {
		if strings.Contains(url, c) {
			return true
		}
	}
	return false
}

func isCategoryURL(url string) bool {
	// not interested in amazon video
	if strings.Contains(url, "/Amazon-Video/") {
		return false
	}

	contains := []string{"/b?", "/b/"}
	for _, c := range contains {
		if strings.Contains(url, c) {
			return true
		}
	}
	return false
}

const AMAZON_BASE_URL = "https://amazon.com"

func createProductURL(asin string) string {
	return fmt.Sprintf("%s/dp/%s", AMAZON_BASE_URL, asin)
}

var ALLOWED_SEARCH_PARAMS = map[string]bool{
	"rnid":     true,
	"node":     true,
	"bbn":      true,
	"keywords": true,
	"k":        true,
	"c":        true,
	"i":        true,
	"page":     true,
	// "rh":             true, // used for filtering items based on their attributes
	"sprefix":        true,
	"search-alias":   true,
	"field-author":   true,
	"field-keywords": true,
	"text":           true,
	// "language": true
}

func withBaseURL(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return filterQueryParams(href)
	}

	if !strings.HasPrefix(href, "/") {
		href = "/" + href
	}

	full := AMAZON_BASE_URL + href
	return filterQueryParams(full)
}

func filterQueryParams(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	q := parsedURL.Query()
	filtered := url.Values{}
	for key, val := range q {
		if ALLOWED_SEARCH_PARAMS[key] {
			filtered[key] = val
		}
	}
	parsedURL.RawQuery = filtered.Encode()

	// clean "ref=..." path
	segments := strings.Split(parsedURL.Path, "/")
	var cleanSegments []string
	for _, s := range segments {
		if s != "" && !strings.HasPrefix(s, "ref=") && !strings.HasPrefix(s, "ref-") {
			cleanSegments = append(cleanSegments, s)
		}
	}
	parsedURL.Path = "/" + strings.Join(cleanSegments, "/")

	return parsedURL.String()
}

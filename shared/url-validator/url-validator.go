package urlValidator

import (
	"net/url"
	"strings"
)

// IsValidURL validates wether a given URL is valid
func IsValidURL(rawURL string) bool {
	_, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}

	return true
}

// IsValidURLOfGivenDomain asserts the URL is from the domain specified
func IsValidURLOfGivenDomain(rawURL, domain string) bool {
	return strings.HasPrefix(rawURL, domain) && IsValidURL(rawURL)
}

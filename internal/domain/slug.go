package domain

import (
	"regexp"
	"strings"
)

var (
	// slugRegexp matches non-alphanumeric characters
	slugRegexp = regexp.MustCompile(`[^a-z0-9]+`)
	// multiDashRegexp matches multiple consecutive dashes
	multiDashRegexp = regexp.MustCompile(`-+`)
)

// Slugify converts a string to a URL-safe slug.
// It lowercases, replaces non-alphanumeric chars with dashes,
// and trims leading/trailing dashes.
func Slugify(s string) string {
	// Lowercase
	result := strings.ToLower(s)

	// Replace non-alphanumeric with dashes
	result = slugRegexp.ReplaceAllString(result, "-")

	// Collapse multiple dashes
	result = multiDashRegexp.ReplaceAllString(result, "-")

	// Trim leading/trailing dashes
	result = strings.Trim(result, "-")

	return result
}

// GenerateThreadSlug generates a thread folder slug.
// If contactName is provided: "<slugified-contact-name>-<code>"
// Otherwise: "thread-<code>"
func GenerateThreadSlug(contactName, code string) string {
	if contactName != "" {
		return Slugify(contactName) + "-" + code
	}
	return "thread-" + code
}

// ThreadFolderPath returns the folder path for a thread given its slug.
func ThreadFolderPath(slug string) string {
	return "data/threads/" + slug
}

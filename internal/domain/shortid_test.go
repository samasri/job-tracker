package domain

import (
	"strings"
	"testing"
)

func TestNewShortID8_Length(t *testing.T) {
	id, err := NewShortID8()
	if err != nil {
		t.Fatalf("NewShortID8() error = %v", err)
	}
	if len(id) != ShortIDLength {
		t.Errorf("NewShortID8() length = %d, want %d", len(id), ShortIDLength)
	}
}

func TestNewShortID8_Charset(t *testing.T) {
	// Generate multiple IDs and verify all characters are from Crockford Base32
	for i := 0; i < 100; i++ {
		id, err := NewShortID8()
		if err != nil {
			t.Fatalf("NewShortID8() error = %v", err)
		}
		for _, c := range id {
			if !strings.ContainsRune(crockfordBase32, c) {
				t.Errorf("NewShortID8() produced invalid character %q not in alphabet %q", c, crockfordBase32)
			}
		}
	}
}

func TestNewShortID8_Uniqueness(t *testing.T) {
	// Generate 20,000 IDs and check for duplicates
	// With 32^8 = ~1.1 trillion possibilities, duplicates should be virtually impossible
	const count = 20000
	seen := make(map[string]struct{}, count)

	for i := 0; i < count; i++ {
		id, err := NewShortID8()
		if err != nil {
			t.Fatalf("NewShortID8() error = %v at iteration %d", err, i)
		}
		if _, exists := seen[id]; exists {
			t.Errorf("NewShortID8() produced duplicate ID %q at iteration %d", id, i)
		}
		seen[id] = struct{}{}
	}
}

func TestNewShortID8_Distribution(t *testing.T) {
	// Generate IDs and check that we're using the full alphabet
	// (this is a sanity check for the randomness)
	charCount := make(map[rune]int)
	const iterations = 10000

	for i := 0; i < iterations; i++ {
		id, err := NewShortID8()
		if err != nil {
			t.Fatalf("NewShortID8() error = %v", err)
		}
		for _, c := range id {
			charCount[c]++
		}
	}

	// We should see all 32 characters from the alphabet
	totalChars := iterations * ShortIDLength
	expectedPerChar := totalChars / 32

	for _, c := range crockfordBase32 {
		count := charCount[c]
		// Allow for statistical variance - each char should appear roughly 1/32 of the time
		// We're checking that each char appears at least 50% of expected frequency
		if count < expectedPerChar/2 {
			t.Errorf("Character %q appeared %d times, expected approximately %d (at least %d)",
				c, count, expectedPerChar, expectedPerChar/2)
		}
	}
}

func TestIsShortID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "valid short ID",
			id:   "ABCD1234",
			want: true,
		},
		{
			name: "UUID is not short ID",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			want: false,
		},
		{
			name: "too short",
			id:   "ABC1234",
			want: false,
		},
		{
			name: "too long",
			id:   "ABCD12345",
			want: false,
		},
		{
			name: "empty string",
			id:   "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsShortID(tt.id); got != tt.want {
				t.Errorf("IsShortID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func BenchmarkNewShortID8(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewShortID8()
	}
}

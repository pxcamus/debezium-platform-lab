package preflight

import "testing"

// TestOlderThan pins the comparison to numeric ordering. A string comparison would report
// kind 0.10.0 as newer than the 0.20 floor, which is exactly the case the check exists to
// catch and the one a lexical compare gets wrong.
func TestOlderThan(t *testing.T) {
	cases := []struct {
		version string
		floor   string
		older   bool
	}{
		{"1.36.2", "1.30", false},
		{"1.29.0", "1.30", true},
		{"0.32.0", "0.20", false},
		{"0.10.0", "0.20", true},
		{"0.9.0", "0.20", true},
		{"4.2.3", "3.0", false},
		{"2.17.0", "3.0", true},
		{"1.0.0", "1.0", false},
		{"0.9.9", "1.0", true},
		{"1.7.0", "1.0", false},
		{"1", "1.0", false},
	}

	for _, tc := range cases {
		if got := olderThan(tc.version, tc.floor); got != tc.older {
			t.Errorf("olderThan(%q, %q) = %v, want %v", tc.version, tc.floor, got, tc.older)
		}
	}
}

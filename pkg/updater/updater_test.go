package updater

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest  string
		current string
		want    bool
	}{
		{"1.2.1", "1.2.0", true},
		{"v1.2.1", "1.2.0", true},
		{"1.3.0", "1.2.9", true},
		{"2.0.0", "1.99.99", true},
		{"1.2.0", "1.2.0", false},
		{"1.2.0", "1.2.1", false},
		{"1.2", "1.2.0", false},
		{"1.2.0.1", "1.2.0", true},
		{"1.2.1-beta", "1.2.0", true},
	}

	for _, tc := range cases {
		if got := isNewer(tc.latest, tc.current); got != tc.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"1.2.0":   "1.2.0",
		"v1.2.0":  "1.2.0",
		"V1.2.0":  "1.2.0",
		" 1.2.0 ": "1.2.0",
	}
	for in, want := range cases {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

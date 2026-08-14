package cli

import "testing"

func TestIsHelpArg(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"-h", true},
		{"--h", true},
		{"-help", true},
		{"--help", true},
		{"h", false},    // no leading dash: a literal positional value, not a flag
		{"help", false}, // same
		{"github", false},
		{"owner/name", false},
		{"", false},
		{"-v", false},
		{"--db", false},
		{"-hh", false},
	}
	for _, c := range cases {
		if got := isHelpArg(c.in); got != c.want {
			t.Errorf("isHelpArg(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

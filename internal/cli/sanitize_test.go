package cli

import "testing"

func TestSanitizeTerminal(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text unchanged",
			in:   "a normal README description with punctuation!?.",
			want: "a normal README description with punctuation!?.",
		},
		{
			name: "newlines and tabs preserved",
			in:   "line one\n\tline two",
			want: "line one\n\tline two",
		},
		{
			name: "ANSI CSI color escape stripped",
			in:   "\x1b[31mred text\x1b[0m",
			want: "[31mred text[0m",
		},
		{
			name: "OSC title-spoofing escape stripped",
			in:   "safe\x1b]0;fake title\x07looking",
			want: "safe]0;fake titlelooking",
		},
		{
			name: "carriage return stripped (overwrite trick)",
			in:   "hello\rgoodbye",
			want: "hellogoodbye",
		},
		{
			name: "DEL byte stripped",
			in:   "abc\x7fdef",
			want: "abcdef",
		},
		{
			name: "unicode content unaffected",
			in:   "日本語 emoji 🎉 café",
			want: "日本語 emoji 🎉 café",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sanitizeTerminal(c.in); got != c.want {
				t.Errorf("sanitizeTerminal(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestSanitizeTerminal_NoESCOrC0Survives(t *testing.T) {
	// Defense-in-depth check: for an input containing every C0 control
	// byte, none should survive except \n and \t.
	var in []byte
	for b := 0x00; b <= 0x1f; b++ {
		in = append(in, byte(b))
	}
	in = append(in, 0x7f)
	in = append(in, []byte("visible")...)

	got := sanitizeTerminal(string(in))
	for _, r := range got {
		if r != '\n' && r != '\t' && (r < 0x20 || r == 0x7f) {
			t.Fatalf("control byte %#x survived sanitizeTerminal, got %q", r, got)
		}
	}
	// \t is 0x09 and \n is 0x0A, so in byte order \t comes first.
	if got != "\t\nvisible" {
		t.Errorf("sanitizeTerminal(all C0 bytes) = %q, want %q", got, "\t\nvisible")
	}
}

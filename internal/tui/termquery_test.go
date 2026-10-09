package tui

import "testing"

func TestParseSchemeReply(t *testing.T) {
	cases := []struct {
		in             string
		dark, ok, done bool
	}{
		{"\x1b[?997;1n\x1b[?1;2c", true, true, true},
		{"\x1b[?997;2n\x1b[?62;4c", false, true, true},
		{"\x1b[?1;2c", false, false, true}, // no 2031 support
		{"\x1b[?997;2n", false, true, false},
		{"\x1b[?997;1n\x1b[?1;", true, true, false},
	}
	for _, c := range cases {
		dark, ok, done := parseSchemeReply([]byte(c.in))
		if dark != c.dark || ok != c.ok || done != c.done {
			t.Errorf("%q: got dark=%v ok=%v done=%v", c.in, dark, ok, done)
		}
	}
}

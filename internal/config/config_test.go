package config

import "testing"

func TestSetUITheme(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"empty", "", "[ui]\ntheme = \"Nord\"\n"},
		{"replace keeps comments", "# mine\n[ui]\ntheme = \"Dracula\" # c\ncompact_tree = true\n[keys]\ntheme = \"x\"\n",
			"# mine\n[ui]\ntheme = \"Nord\"\ncompact_tree = true\n[keys]\ntheme = \"x\"\n"},
		{"insert into ui", "[ui]\ncompact_tree = true\n", "[ui]\ntheme = \"Nord\"\ncompact_tree = true\n"},
		{"ui.theme_override is not ui", "[ui.theme_override]\nborder = \"#fff\"\n", "[ui.theme_override]\nborder = \"#fff\"\n\n[ui]\ntheme = \"Nord\"\n"},
		{"append", "[keys]\nquit = \"q\"", "[keys]\nquit = \"q\"\n\n[ui]\ntheme = \"Nord\"\n"},
	}
	for _, c := range cases {
		if got := setUITheme(c.in, "Nord"); got != c.want {
			t.Errorf("%s:\ngot  %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestIsSequencePrefix(t *testing.T) {
	cases := []struct {
		key, binding string
		want         bool
	}{
		{"g", "gg", true},
		{"g", "g", false},     // single-key binding must fire immediately
		{"f", "f1,f2", false}, // function keys are not sequences
		{"p", "pgup", false},  // named keys are not sequences
		{"z", "zz,zt", true},
	}
	for _, c := range cases {
		if got := IsSequencePrefix(c.key, c.binding); got != c.want {
			t.Errorf("IsSequencePrefix(%q, %q) = %v, want %v", c.key, c.binding, got, c.want)
		}
	}
}

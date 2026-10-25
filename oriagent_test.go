package oriagent

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sitle/ori-agent/i18n"
)

// keyDiff returns the keys of a missing from b.
func keyDiff(a, b map[string]string) []string {
	var missing []string
	for k := range a {
		if _, ok := b[k]; !ok {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	return missing
}

func TestCatalogsHaveSameKeys(t *testing.T) {
	fr, en := Catalogs["fr"], Catalogs["en"]
	if m := keyDiff(fr, en); len(m) > 0 {
		t.Errorf("missing in en: %v", m)
	}
	if m := keyDiff(en, fr); len(m) > 0 {
		t.Errorf("missing in fr: %v", m)
	}
	for l, c := range Catalogs {
		if c["lang.name"] == "" {
			t.Errorf("%s: no lang.name", l)
		}
	}
}

// Verbs must match between languages, or T would format garbage.
func TestCatalogVerbsMatch(t *testing.T) {
	count := func(s string) int { return strings.Count(s, "%") - 2*strings.Count(s, "%%") }
	for k, fr := range Catalogs["fr"] {
		if en, ok := Catalogs["en"][k]; ok && count(fr) != count(en) {
			t.Errorf("%s: %d verbs in fr, %d in en", k, count(fr), count(en))
		}
	}
}

func TestBuiltinCommandsAreTranslated(t *testing.T) {
	for _, c := range builtinCommands(Config{}) {
		for _, key := range []string{c.Desc, c.Args} {
			if strings.HasPrefix(key, "cmd.") {
				if _, ok := Catalogs["fr"][key]; !ok {
					t.Errorf("/%s: key %q not in catalogs", c.Name, key)
				}
			}
		}
	}
}

func bundle(locale string) *i18n.Bundle {
	b := i18n.New("fr")
	for l, c := range Catalogs {
		b.Add(l, c)
	}
	b.SetLocale(locale)
	return b
}

func TestHumanSize(t *testing.T) {
	cases := []struct {
		n    int64
		tr   i18n.Translator
		want string
	}{
		{512, nil, "512 B"},
		{2048, nil, "2 KB"},
		{1536 << 10, nil, "1.5 MB"},
		{1536 << 10, bundle("fr"), "1,5 Mo"},
		{1536 << 10, bundle("en"), "1.5 MB"},
		{2048, bundle("fr"), "2 Ko"},
	}
	for _, c := range cases {
		if got := HumanSize(c.n, c.tr); got != c.want {
			t.Errorf("HumanSize(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("  short   text ", 40); got != "short text" {
		t.Errorf("got %q", got)
	}
	if got := truncate("abcdefghij", 5); got != "abcd…" {
		t.Errorf("got %q", got)
	}
}

func TestWrap(t *testing.T) {
	for _, c := range [][3]int{{0, 3, 0}, {3, 3, 0}, {-1, 3, 2}, {5, 0, 0}} {
		if got := wrap(c[0], c[1]); got != c[2] {
			t.Errorf("wrap(%d, %d) = %d, want %d", c[0], c[1], got, c[2])
		}
	}
}

func TestPomodoroCycle(t *testing.T) {
	var p Pomodoro
	p.init(PomodoroConfig{Focus: 25 * time.Minute, Short: 5 * time.Minute, Long: 15 * time.Minute}, bundle("en"))
	now := time.Now()

	p.Start(now)
	if left := p.Left(now.Add(10 * time.Minute)); left != 15*time.Minute {
		t.Errorf("left after 10 min = %v", left)
	}
	p.Pause(now.Add(10 * time.Minute))
	if p.Running() || p.Left(now.Add(time.Hour)) != 15*time.Minute {
		t.Error("pause should freeze the remaining time")
	}

	// Focus → short break three times, then a long break.
	modes := []string{}
	for i := 0; i < 8; i++ {
		p.Next()
		modes = append(modes, p.Mode())
	}
	want := []string{"Short break", "Focus", "Short break", "Focus", "Short break", "Focus", "Long break", "Focus"}
	if strings.Join(modes, ",") != strings.Join(want, ",") {
		t.Errorf("modes = %v, want %v", modes, want)
	}
	if p.Cycles() != 4 {
		t.Errorf("cycles = %d", p.Cycles())
	}
}

package i18n

import "testing"

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"fr": "fr", "fr-FR": "fr", "en_US.UTF-8": "en", " EN ": "en", "de@euro": "de", "": "",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLookupAndFallback(t *testing.T) {
	b := New("fr")
	b.Add("fr", map[string]string{"hello": "Bonjour %s", "only.fr": "seulement"})
	b.Add("en", map[string]string{"hello": "Hello %s"})

	if got := b.T("hello", "Teva"); got != "Bonjour Teva" {
		t.Errorf("fr: got %q", got)
	}
	if !b.SetLocale("en-GB") {
		t.Fatal("SetLocale(en-GB) failed")
	}
	if got := b.T("hello", "Teva"); got != "Hello Teva" {
		t.Errorf("en: got %q", got)
	}
	if got := b.T("only.fr"); got != "seulement" {
		t.Errorf("fallback: got %q", got)
	}
	if got := b.T("Plain text"); got != "Plain text" {
		t.Errorf("unknown key should pass through, got %q", got)
	}
	if b.SetLocale("de") {
		t.Error("SetLocale(de) should fail without a catalog")
	}
	if b.Locale() != "en" {
		t.Errorf("locale changed on failure: %q", b.Locale())
	}
	if got := b.In("fr", "hello", "x"); got != "Bonjour x" {
		t.Errorf("In: got %q", got)
	}
}

func TestMerge(t *testing.T) {
	m := Merge(
		map[string]map[string]string{"fr": {"a": "1", "b": "2"}},
		map[string]map[string]string{"fr": {"b": "3"}, "en": {"a": "one"}},
	)
	if m["fr"]["a"] != "1" || m["fr"]["b"] != "3" || m["en"]["a"] != "one" {
		t.Errorf("unexpected merge: %v", m)
	}
}

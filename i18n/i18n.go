// Package i18n translates interface strings. A Bundle holds one catalog per
// locale (key → text) and a current locale; lookups fall back to the
// fallback locale, then to the key itself, so plain text passes through
// untouched. Texts may contain fmt verbs, filled from T's arguments.
package i18n

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

// Translator is what widgets and pages need to display text.
type Translator interface {
	T(key string, args ...any) string
}

// Bundle is safe for concurrent use.
type Bundle struct {
	mu       sync.RWMutex
	locale   string
	fallback string
	msgs     map[string]map[string]string
}

// New returns an empty bundle whose current and fallback locale is fallback.
func New(fallback string) *Bundle {
	return &Bundle{locale: fallback, fallback: fallback, msgs: map[string]map[string]string{}}
}

// Add merges msgs into the catalog of locale, overriding existing keys. A
// locale with only some keys falls back for the others.
func (b *Bundle) Add(locale string, msgs map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	locale = Normalize(locale)
	c := b.msgs[locale]
	if c == nil {
		c = map[string]string{}
		b.msgs[locale] = c
	}
	for k, v := range msgs {
		c[k] = v
	}
}

// SetLocale switches language; it accepts forms like "en-US" or "fr_FR.UTF-8"
// and reports false when no catalog matches.
func (b *Bundle) SetLocale(locale string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	l := Normalize(locale)
	if _, ok := b.msgs[l]; !ok {
		return false
	}
	b.locale = l
	return true
}

func (b *Bundle) Locale() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.locale
}

// Locales lists the locales that have a catalog, sorted.
func (b *Bundle) Locales() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	ls := make([]string, 0, len(b.msgs))
	for l := range b.msgs {
		ls = append(ls, l)
	}
	sort.Strings(ls)
	return ls
}

// Has reports whether key is translated in the current or fallback locale.
func (b *Bundle) Has(key string) bool {
	_, ok := b.lookup(b.Locale(), key)
	return ok
}

func (b *Bundle) lookup(locale, key string) (string, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if s, ok := b.msgs[locale][key]; ok {
		return s, true
	}
	s, ok := b.msgs[b.fallback][key]
	return s, ok
}

// T translates key in the current locale.
func (b *Bundle) T(key string, args ...any) string { return b.In(b.Locale(), key, args...) }

// In translates key in a given locale; unknown keys are returned as is.
func (b *Bundle) In(locale, key string, args ...any) string {
	s, ok := b.lookup(Normalize(locale), key)
	if !ok {
		s = key
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// Merge combines catalog sets (locale → key → text); later ones win.
func Merge(sets ...map[string]map[string]string) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, set := range sets {
		for l, c := range set {
			if out[l] == nil {
				out[l] = map[string]string{}
			}
			for k, v := range c {
				out[l][k] = v
			}
		}
	}
	return out
}

// Normalize reduces a locale to its lower-case language code: "fr_FR.UTF-8"
// and "fr-FR" become "fr".
func Normalize(locale string) string {
	l := strings.ToLower(strings.TrimSpace(locale))
	if i := strings.IndexAny(l, "-_.@"); i >= 0 {
		l = l[:i]
	}
	return l
}

// Detect returns the user's language from the environment (LC_ALL,
// LC_MESSAGES, LANG) or the OS settings, normalized; "" if unknown.
func Detect() string {
	for _, v := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if l := Normalize(os.Getenv(v)); l != "" && l != "c" && l != "posix" {
			return l
		}
	}
	return Normalize(osLocale())
}

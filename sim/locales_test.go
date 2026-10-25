package sim

import (
	"strings"
	"testing"
)

func TestTranslationsHaveSameKeys(t *testing.T) {
	fr, en := Translations["fr"], Translations["en"]
	for k := range fr {
		if _, ok := en[k]; !ok {
			t.Errorf("missing in en: %s", k)
		}
	}
	for k := range en {
		if _, ok := fr[k]; !ok {
			t.Errorf("missing in fr: %s", k)
		}
	}
	for k, f := range fr {
		if strings.Count(f, "%") != strings.Count(en[k], "%") {
			t.Errorf("%s: verb count differs", k)
		}
	}
}

func TestHead(t *testing.T) {
	if got := Head("does-not-exist.txt", 3); got == "" {
		t.Error("expected an error message for a missing file")
	}
}

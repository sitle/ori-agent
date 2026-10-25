package ui

import "testing"

func TestFakeLevelRange(t *testing.T) {
	for i := 0; i < 2000; i++ {
		if l := FakeLevel(float64(i) * 0.013); l < 0 || l > 1 {
			t.Fatalf("FakeLevel out of [0,1]: %v", l)
		}
	}
}

//go:build !windows

package i18n

import (
	"os/exec"
	"runtime"
	"strings"
)

// osLocale reads the macOS language setting; elsewhere the environment
// variables checked by Detect are the source of truth.
func osLocale() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

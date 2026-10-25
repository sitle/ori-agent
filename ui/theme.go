// Package ui is the widget kit of ori-agent: drawing primitives, form
// controls, icons and overlays, all drawn with a Theme. It does not depend on
// the application shell and can be used in any Gio program.
package ui

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/sitle/ori-agent/theme"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Theme is the active palette plus the text shaper. Switching between light
// and dark only swaps the embedded Palette, so widgets just read th.Bg etc.
type Theme struct {
	theme.Palette
	Material *material.Theme
	Dark     bool
	// MonoFace is the typeface used for code, paths and timers.
	MonoFace font.Typeface

	light, dark theme.Palette
}

// NewTheme returns a theme in dark mode. With no font collection, the Go
// fonts (embedded, available on every OS) are used.
func NewTheme(light, dark theme.Palette, fonts ...font.FontFace) *Theme {
	if len(fonts) == 0 {
		fonts = gofont.Collection()
	}
	m := material.NewTheme()
	m.Shaper = text.NewShaper(text.WithCollection(fonts))
	t := &Theme{Material: m, MonoFace: "Go Mono", light: light, dark: dark}
	t.SetDark(true)
	return t
}

func (t *Theme) SetDark(dark bool) {
	t.Dark = dark
	if dark {
		t.Palette = t.dark
	} else {
		t.Palette = t.light
	}
	t.Material.Palette = material.Palette{Bg: t.Bg, Fg: t.Text, ContrastBg: t.Accent, ContrastFg: t.OnAccent}
}

func (t *Theme) Label(size unit.Sp, s string, c color.NRGBA, w font.Weight) material.LabelStyle {
	l := material.Label(t.Material, size, s)
	l.Color = c
	l.Font.Weight = w
	return l
}

func (t *Theme) Mono(size unit.Sp, s string, c color.NRGBA) material.LabelStyle {
	l := t.Label(size, s, c, font.Normal)
	l.Font.Typeface = t.MonoFace
	return l
}

// Body is a paragraph of chat or document text.
func (t *Theme) Body(size unit.Sp, s string) layout.Widget {
	return func(gtx C) D {
		l := t.Label(size, s, t.Text, font.Normal)
		l.LineHeightScale = 1.45
		return l.Layout(gtx)
	}
}

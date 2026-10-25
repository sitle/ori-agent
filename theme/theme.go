// Package theme holds color palettes. The defaults follow the Ori design
// system of the Polynésie française public services (https://ori.gov.pf):
// raw tokens below, mapped onto the semantic Palette used by every widget.
// Another design system only needs to provide two Palettes.
package theme

import "image/color"

// RGB builds an opaque color from 0xRRGGBB.
func RGB(c uint32) color.NRGBA {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 0xff}
}

// Raw Ori palettes (https://ori.gov.pf/fondations/palette/).
var (
	Neutral0   = RGB(0xffffff)
	Neutral50  = RGB(0xfafafa)
	Neutral100 = RGB(0xf4f4f5)
	Neutral200 = RGB(0xe4e4e7)
	Neutral300 = RGB(0xd4d4d8)
	Neutral400 = RGB(0xa1a1aa)
	Neutral500 = RGB(0x71717a)
	Neutral600 = RGB(0x52525b)
	Neutral700 = RGB(0x3f3f46)
	Neutral800 = RGB(0x27272a)
	Neutral900 = RGB(0x18181b)
	Neutral950 = RGB(0x09090b)

	Primary50  = RGB(0xe9eeff)
	Primary200 = RGB(0xa1b4ff)
	Primary300 = RGB(0x7d96ff)
	Primary500 = RGB(0x073ca5)
	Primary600 = RGB(0x062f88)
	Primary900 = RGB(0x030d31)

	Success200 = RGB(0x7cd9a6)
	Success500 = RGB(0x0bb25a)
	Success700 = RGB(0x066d36)
	Danger300  = RGB(0xef7f8d)
	Danger500  = RGB(0xcf0921)
	Warning500 = RGB(0xf97316)
)

// Palette is the set of semantic colors the widgets draw with.
type Palette struct {
	Bg, Sidebar, Surface, Hover, Selected color.NRGBA // surfaces
	Border, BorderHi                      color.NRGBA
	Text, Muted                           color.NRGBA
	Accent, AccentHover, OnAccent         color.NRGBA // brand
	Success, DiffAdd, DiffDel             color.NRGBA // feedback, diff lines
	Danger, Warning, SuccessStrong        color.NRGBA // solid fills: badges, close button
	OnStrong                              color.NRGBA // text on solid fills
	UserBubble, Code, CodeText            color.NRGBA
}

// OriLight follows Ori's :root tokens.
var OriLight = Palette{
	Bg:            Neutral0,   // surface-base
	Sidebar:       Neutral50,  // surface-muted
	Surface:       Neutral50,  // surface-muted
	Hover:         Neutral100, // surface-subtle
	Selected:      Neutral200,
	Border:        Neutral200, // border-subtle
	BorderHi:      Neutral300, // border-default
	Text:          Neutral900, // text-primary
	Muted:         Neutral500, // text-muted
	Accent:        Primary500, // brand-primary
	AccentHover:   Primary600, // brand-primary-hover
	OnAccent:      Neutral0,   // brand-on-primary
	Success:       Success500, // feedback-success
	DiffAdd:       Success700, // feedback-success-strong
	DiffDel:       Danger500,  // feedback-danger
	Danger:        Danger500,
	Warning:       Warning500,
	SuccessStrong: Success700,
	OnStrong:      Neutral0,
	UserBubble:    Primary50,  // brand-primary-subtle
	Code:          Neutral100, // surface-subtle
	CodeText:      Neutral600, // text-secondary
}

// OriDark follows Ori's [data-theme="dark"] tokens.
var OriDark = Palette{
	Bg:            Neutral900, // surface-base
	Sidebar:       Neutral950,
	Surface:       Neutral800, // surface-muted
	Hover:         Neutral800, // surface-muted
	Selected:      Neutral700, // surface-subtle
	Border:        Neutral700, // border-subtle
	BorderHi:      Neutral600, // border-default
	Text:          Neutral50,  // text-primary
	Muted:         Neutral400, // text-muted
	Accent:        Primary300, // brand-primary
	AccentHover:   Primary200, // brand-primary-hover
	OnAccent:      Neutral900, // brand-on-primary
	Success:       Success500, // feedback-success
	DiffAdd:       Success200, // feedback-success-strong
	DiffDel:       Danger300,  // feedback-danger-strong
	Danger:        Danger500,
	Warning:       Warning500,
	SuccessStrong: Success700,
	OnStrong:      Neutral0,
	UserBubble:    Primary900, // brand-primary-subtle
	Code:          Neutral950,
	CodeText:      Neutral300, // text-secondary
}

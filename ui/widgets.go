package ui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type ButtonKind int

const (
	Primary ButtonKind = iota
	Secondary
	Danger
)

func (t *Theme) Button(gtx C, btn *widget.Clickable, txt string, kind ButtonKind) D {
	return btn.Layout(gtx, func(gtx C) D {
		in := layout.Inset{Top: 8, Bottom: 8, Left: 16, Right: 16}
		bg := t.Bg
		if btn.Hovered() {
			bg = t.Hover
		}
		switch kind {
		case Primary:
			bg := t.Accent
			if btn.Hovered() {
				bg = t.AccentHover
			}
			return Card(gtx, bg, 8, in, t.Label(14, txt, t.OnAccent, font.SemiBold).Layout)
		case Danger:
			return Bordered(gtx, bg, t.DiffDel, 8, func(gtx C) D {
				return in.Layout(gtx, t.Label(14, txt, t.DiffDel, font.SemiBold).Layout)
			})
		}
		return Bordered(gtx, bg, t.BorderHi, 8, func(gtx C) D {
			return in.Layout(gtx, t.Label(14, txt, t.Text, font.Medium).Layout)
		})
	})
}

// IconButton is a small square button around an icon drawn in the given color.
func (t *Theme) IconButton(gtx C, btn *widget.Clickable, base color.NRGBA, enabled bool, icon func(gtx C, c color.NRGBA) D) D {
	return btn.Layout(gtx, func(gtx C) D {
		c, bg := t.Muted, base
		switch {
		case !enabled:
			c.A = 90
		case btn.Hovered():
			c, bg = t.Text, t.Hover
		}
		return Card(gtx, bg, 6, layout.UniformInset(7), func(gtx C) D {
			gtx.Constraints.Min = image.Pt(gtx.Dp(18), gtx.Dp(16))
			return layout.Center.Layout(gtx, func(gtx C) D { return icon(gtx, c) })
		})
	})
}

// RoundButton is a 32dp circular button, used in the composer.
func (t *Theme) RoundButton(gtx C, btn *widget.Clickable, bg, fg color.NRGBA, icon func(gtx C, c color.NRGBA) D) D {
	return btn.Layout(gtx, func(gtx C) D {
		if btn.Hovered() && bg == t.Surface {
			bg, fg = t.Hover, t.Text
		}
		s := gtx.Dp(32)
		gtx.Constraints = layout.Exact(image.Pt(s, s))
		paint.FillShape(gtx.Ops, bg, clip.Ellipse(image.Rect(0, 0, s, s)).Op(gtx.Ops))
		return layout.Center.Layout(gtx, func(gtx C) D { return icon(gtx, fg) })
	})
}

// NavRow is a full-width selectable row, as in sidebars and menus.
func (t *Theme) NavRow(gtx C, btn *widget.Clickable, base color.NRGBA, selected bool, w layout.Widget) D {
	return btn.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		bg := base
		switch {
		case selected:
			bg = t.Selected
		case btn.Hovered():
			bg = t.Hover
		}
		return Card(gtx, bg, 8, layout.Inset{Top: 7, Bottom: 7, Left: 10, Right: 10}, w)
	})
}

func (t *Theme) Chip(gtx C, s string, c color.NRGBA) D {
	return Bordered(gtx, t.Bg, t.Border, 10, func(gtx C) D {
		return layout.Inset{Top: 3, Bottom: 3, Left: 9, Right: 9}.Layout(gtx, t.Label(12, s, c, font.Medium).Layout)
	})
}

// Field is a labelled single- or multi-line text input.
func (t *Theme) Field(gtx C, label string, ed *widget.Editor, hint string) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(t.Label(12, label, t.Muted, font.Medium).Layout),
		VSpace(6),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			border := t.BorderHi
			if gtx.Focused(ed) {
				border = t.Accent
			}
			return Bordered(gtx, t.Bg, border, 8, func(gtx C) D {
				return layout.Inset{Top: 9, Bottom: 9, Left: 12, Right: 12}.Layout(gtx, func(gtx C) D {
					e := material.Editor(t.Material, ed, hint)
					e.TextSize = 14
					e.Color, e.HintColor = t.Text, t.Muted
					return e.Layout(gtx)
				})
			})
		}),
	)
}

// ToggleRow is a title, a description and a switch; the whole row toggles b.
func (t *Theme) ToggleRow(gtx C, b *widget.Bool, title, desc string) D {
	return b.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Inset{Top: 8, Bottom: 8}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(t.Label(14, title, t.Text, font.Medium).Layout),
						VSpace(2),
						layout.Rigid(t.Label(12, desc, t.Muted, font.Normal).Layout),
					)
				}),
				HSpace(16),
				layout.Rigid(func(gtx C) D { return t.Switch(gtx, b.Value) }),
			)
		})
	})
}

func (t *Theme) Switch(gtx C, on bool) D {
	w, h, pad := gtx.Dp(38), gtx.Dp(22), gtx.Dp(3)
	track := t.BorderHi
	if on {
		track = t.Accent
	}
	paint.FillShape(gtx.Ops, track, clip.UniformRRect(image.Rect(0, 0, w, h), h/2).Op(gtx.Ops))
	k := h - 2*pad
	x := pad
	if on {
		x = w - k - pad
	}
	paint.FillShape(gtx.Ops, t.OnStrong, clip.Ellipse(image.Rect(x, pad, x+k, pad+k)).Op(gtx.Ops))
	return D{Size: image.Pt(w, h)}
}

func (t *Theme) Radio(gtx C, on bool) D {
	s := gtx.Dp(18)
	if !on {
		return Ring(gtx, t.BorderHi, 18)
	}
	Ring(gtx, t.Accent, 18)
	in := gtx.Dp(5)
	paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse(image.Rect(in, in, s-in, s-in)).Op(gtx.Ops))
	return D{Size: image.Pt(s, s)}
}

// ChoiceCard is a radio option drawn as a card.
func (t *Theme) ChoiceCard(gtx C, btn *widget.Clickable, selected bool, title, desc string) D {
	return btn.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		border, bg := t.Border, t.Bg
		switch {
		case selected:
			border, bg = t.Accent, t.UserBubble
		case btn.Hovered():
			bg = t.Hover
		}
		return Bordered(gtx, bg, border, 10, func(gtx C) D {
			return layout.UniformInset(14).Layout(gtx, func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D { return t.Radio(gtx, selected) }),
					HSpace(12),
					layout.Flexed(1, func(gtx C) D {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(t.Label(14, title, t.Text, font.SemiBold).Layout),
							layout.Rigid(func(gtx C) D {
								if desc == "" {
									return D{}
								}
								return layout.Inset{Top: 2}.Layout(gtx, t.Label(12, desc, t.Muted, font.Normal).Layout)
							}),
						)
					}),
				)
			})
		})
	})
}

// Avatar is a colored disc with initials.
func (t *Theme) Avatar(gtx C, initials string, size unit.Dp) D {
	s := gtx.Dp(size)
	gtx.Constraints = layout.Exact(image.Pt(s, s))
	paint.FillShape(gtx.Ops, t.Accent, clip.Ellipse(image.Rect(0, 0, s, s)).Op(gtx.Ops))
	return layout.Center.Layout(gtx, t.Label(unit.Sp(float32(size)*0.38), initials, t.OnAccent, font.SemiBold).Layout)
}

// Panel is a bordered surface card with standard padding.
func (t *Theme) Panel(gtx C, w layout.Widget) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return Bordered(gtx, t.Surface, t.Border, 12, func(gtx C) D {
		return layout.Inset{Top: 18, Bottom: 18, Left: 20, Right: 20}.Layout(gtx, w)
	})
}

// Section is a titled Panel, the building block of settings pages.
func (t *Theme) Section(gtx C, title, desc string, w layout.Widget) D {
	return layout.Inset{Bottom: 28}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(t.Label(16, title, t.Text, font.SemiBold).Layout),
			layout.Rigid(func(gtx C) D {
				if desc == "" {
					return D{}
				}
				return layout.Inset{Top: 3}.Layout(gtx, t.Label(13, desc, t.Muted, font.Normal).Layout)
			}),
			VSpace(12),
			layout.Rigid(func(gtx C) D { return t.Panel(gtx, w) }),
		)
	})
}

// Page lays out a scrollable settings-style page: a title and a centered column.
func (t *Theme) Page(gtx C, list *widget.List, title string, content layout.Widget) D {
	list.Axis = layout.Vertical
	return material.List(t.Material, list).Layout(gtx, 1, func(gtx C, _ int) D {
		return Centered(gtx, 720, func(gtx C) D {
			return layout.Inset{Top: 32, Bottom: 40, Left: 32, Right: 32}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(t.Label(26, title, t.Text, font.SemiBold).Layout),
					VSpace(24),
					layout.Rigid(content),
				)
			})
		})
	})
}

// CodeLines renders text in the mono face; diff lines starting with + or -
// are colored. At most 40 lines are shown.
func (t *Theme) CodeLines(out string) layout.Widget {
	lines := strings.Split(strings.ReplaceAll(out, "\t", "    "), "\n")
	const maxLines = 40
	if len(lines) > maxLines {
		lines = append(lines[:maxLines], "… output truncated")
	}
	return func(gtx C) D {
		children := make([]layout.FlexChild, 0, len(lines))
		for _, ln := range lines {
			c := t.CodeText
			switch {
			case strings.HasPrefix(ln, "+"):
				c = t.DiffAdd
			case strings.HasPrefix(ln, "-"):
				c = t.DiffDel
			}
			children = append(children, layout.Rigid(t.Mono(13, ln, c).Layout))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// ProgressRing draws a track circle and an arc for frac, clockwise from 12h.
func ProgressRing(gtx C, size, width unit.Dp, frac float32, track, fg color.NRGBA) D {
	s := gtx.Dp(size)
	w := float32(gtx.Dp(width))
	half := int(w/2) + 1
	paint.FillShape(gtx.Ops, track, clip.Stroke{Path: clip.Ellipse(image.Rect(half, half, s-half, s-half)).Path(gtx.Ops), Width: w}.Op())
	if frac > 0 {
		c := f32.Pt(float32(s)/2, float32(s)/2)
		r := float32(s)/2 - float32(half)
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(f32.Pt(c.X, c.Y-r))
		p.ArcTo(c, c, min(frac, 0.9999)*2*math.Pi)
		paint.FillShape(gtx.Ops, fg, clip.Stroke{Path: p.End(), Width: w}.Op())
	}
	return D{Size: image.Pt(s, s)}
}

// FakeLevel is a plausible microphone level in [0,1] at time t (seconds):
// syllable-like bursts from a carrier, an envelope and a pseudo-random gain.
func FakeLevel(t float64) float32 {
	h := math.Sin(math.Floor(t*8)*12.9898) * 43758.5453
	gain := 0.35 + 0.65*(h-math.Floor(h))
	v := math.Abs(math.Sin(t*17)*math.Sin(t*3.1+1)) * gain
	return float32(0.08 + 0.92*v)
}

// Waveform draws scrolling level bars over the available width; level gives
// the level at a time in seconds (FakeLevel if nil).
func Waveform(gtx C, c color.NRGBA, height unit.Dp, level func(t float64) float32) D {
	if level == nil {
		level = FakeLevel
	}
	w, h := gtx.Constraints.Max.X, gtx.Dp(height)
	bw, gap := gtx.Dp(3), gtx.Dp(2)
	n := w / (bw + gap)
	now := float64(gtx.Now.UnixNano()) / 1e9
	for i := 0; i < n; i++ {
		l := level(now - float64(n-1-i)*0.045) // newest on the right
		bh := max(gtx.Dp(2), int(float32(h)*l))
		x := i * (bw + gap)
		y := (h - bh) / 2
		col := c
		col.A = uint8(110 + 145*float32(i)/float32(max(n, 1)))
		paint.FillShape(gtx.Ops, col, clip.UniformRRect(image.Rect(x, y, x+bw, y+bh), bw/2).Op(gtx.Ops))
	}
	return D{Size: image.Pt(w, h)}
}

package oriagent

import (
	"image/color"
	"net/url"
	"os/exec"
	"runtime"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/ui"
)

// Links in the transcript open in the system's browser (or with
// Config.OnLink), once the user confirms: a dialog shows the whole address,
// since the text of a link may say something else than where it goes.
// Only web and mail addresses: a link written by the model must not start
// a program or open a local file.

type linkState struct {
	url, host   string // asked to open; empty: no dialog
	mail        bool   // a mailto address: host is the recipient
	ok, cancel  widget.Clickable
	scrim, card widget.Clickable
}

// openLink asks to confirm the opening of address, if it is an http, https
// or mailto one.
func (a *App) openLink(address string) {
	u, err := url.Parse(address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "mailto") || (u.Scheme != "mailto" && u.Host == "") {
		a.notify(a.t("link.refused", address))
		return
	}
	host := u.Hostname()
	if u.Scheme == "mailto" {
		host = u.Opaque
	}
	a.link = linkState{url: u.String(), host: host, mail: u.Scheme == "mailto"}
}

func (a *App) handleLink(gtx C) {
	l := &a.link
	if l.url == "" {
		return
	}
	if l.cancel.Clicked(gtx) || l.scrim.Clicked(gtx) {
		a.link = linkState{}
		return
	}
	if !l.ok.Clicked(gtx) {
		return
	}
	address := l.url
	a.link = linkState{}
	open := a.cfg.OnLink
	if open == nil {
		open = openBrowser
	}
	go func() {
		if err := open(address); err != nil {
			a.update(func() { a.notify(a.t("link.failed", err.Error())) })
		}
	}()
}

// layoutLink draws the confirmation dialog over the window.
func (a *App) layoutLink(gtx C) D {
	l, th := &a.link, a.th
	if l.url == "" {
		return D{}
	}
	size := gtx.Constraints.Max
	l.scrim.Layout(gtx, func(gtx C) D {
		paint.FillShape(gtx.Ops, color.NRGBA{A: 0x80}, clip.Rect{Max: size}.Op())
		return D{Size: size}
	})
	gtx.Constraints = layout.Exact(size)
	return layout.Center.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X-gtx.Dp(40), gtx.Dp(520))
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		// The card takes the clicks, so that they do not close the dialog.
		return l.card.Layout(gtx, func(gtx C) D {
			return ui.Bordered(gtx, th.Bg, th.BorderHi, 12, func(gtx C) D {
				return layout.UniformInset(20).Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(th.Label(16, a.t("link.title"), th.Text, font.Medium).Layout),
						ui.VSpace(10),
						layout.Rigid(func(gtx C) D {
							key := "link.host"
							if l.mail {
								key = "link.mail"
							}
							return th.Label(14, a.t(key, l.host), th.Text, font.SemiBold).Layout(gtx)
						}),
						ui.VSpace(6),
						layout.Rigid(func(gtx C) D {
							return ui.Bordered(gtx, th.Code, th.Border, 6, func(gtx C) D {
								return layout.UniformInset(8).Layout(gtx, th.Mono(13, l.url, th.Text).Layout)
							})
						}),
						ui.VSpace(10),
						layout.Rigid(th.Label(13, a.t("link.warning"), th.Muted, font.Normal).Layout),
						ui.VSpace(18),
						layout.Rigid(func(gtx C) D {
							return layout.E.Layout(gtx, func(gtx C) D {
								return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx C) D { return th.Button(gtx, &l.cancel, a.t("link.cancel"), ui.Secondary) }),
									ui.HSpace(8),
									layout.Rigid(func(gtx C) D { return th.Button(gtx, &l.ok, a.t("link.ok"), ui.Primary) }),
								)
							})
						}),
					)
				})
			})
		})
	})
}

// openBrowser hands address to the system, which opens it in the default
// browser or mail program.
func openBrowser(address string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// No shell: the address is one argument, whatever it contains.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", address)
	case "darwin":
		cmd = exec.Command("open", address)
	default:
		cmd = exec.Command("xdg-open", address)
	}
	return cmd.Start()
}

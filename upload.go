package oriagent

import (
	"fmt"
	"image"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"github.com/ncruces/zenity"

	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/preview"
	"github.com/sitle/ori-agent/ui"
)

// Files are picked with the OS's native dialog through zenity, which needs no
// CGO: Win32 on Windows, osascript on macOS, zenity/kdialog on Linux. The
// upload itself is simulated: a progress bar, then the file is attached.
// Agents read attachments from their Path.

// MaxUpload is the size limit for attachments.
const MaxUpload = 20 << 20

type Attachment struct {
	Path, Name, Ext string // Ext is lower case, without the dot
	Size            int64
	ImageSize       image.Point // for images, once decoded

	progress     float32 // simulated upload, 0..1
	thumb        *paint.ImageOp
	open, remove widget.Clickable
}

// NewAttachment describes an existing file as an already uploaded attachment.
func NewAttachment(path string) (*Attachment, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	t := &Attachment{Path: path, Name: filepath.Base(path), Size: info.Size(), progress: 1,
		Ext: strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))}
	if t.IsImage() {
		if img, ok := preview.DecodeImage(path); ok {
			op := paint.NewImageOp(img)
			t.thumb, t.ImageSize = &op, img.Bounds().Size()
		}
	}
	return t, nil
}

func (t *Attachment) IsImage() bool   { return preview.IsImage(t.Ext) }
func (t *Attachment) uploading() bool { return t.progress < 1 }

// HumanSize formats a byte count in the language of tr ("1,2 Mo", "1.2 MB");
// a Turn or Ctl is a translator. With tr nil, units are in English.
func HumanSize(n int64, tr i18n.Translator) string {
	t := func(key, def string, args ...any) string {
		if tr == nil {
			return fmt.Sprintf(def, args...)
		}
		return tr.T(key, args...)
	}
	switch {
	case n >= 1<<20:
		v := strings.Replace(fmt.Sprintf("%.1f", float64(n)/(1<<20)), ".", t("size.decimal", "."), 1)
		return t("size.mb", "%s MB", v)
	case n >= 1<<10:
		return t("size.kb", "%d KB", (n+512)>>10)
	}
	return t("size.b", "%d B", n)
}

// pickFiles opens the native dialog without blocking the UI.
func (a *App) pickFiles() {
	if a.picking || a.cfg.Disable.Attachments {
		return
	}
	a.picking = true
	dir := a.workDir
	go func() {
		paths, err := zenity.SelectFileMultiple(
			zenity.Title(a.t("upload.dialog")),
			zenity.Filename(dir+string(os.PathSeparator)),
			zenity.FileFilters{
				{Name: a.t("upload.filter.docs"), Patterns: []string{"*.pdf", "*.md", "*.txt", "*.docx", "*.odt", "*.csv", "*.json"}, CaseFold: true},
				{Name: a.t("upload.filter.images"), Patterns: []string{"*.png", "*.jpg", "*.jpeg", "*.gif"}, CaseFold: true},
				{Name: a.t("upload.filter.code"), Patterns: []string{"*.go", "*.py", "*.js", "*.ts", "*.rs"}, CaseFold: true},
				{Name: a.t("upload.filter.all"), Patterns: []string{"*"}},
			},
		)
		a.update(func() {
			a.picking = false
			if err == nil {
				a.addFiles(paths)
			}
		})
	}()
}

// pickFolder fills the setup's working directory field from a native dialog.
func (a *App) pickFolder() {
	dir, title := a.setup.dir.Text(), a.t("upload.folderDialog")
	go func() {
		path, err := zenity.SelectFile(zenity.Title(title), zenity.Directory(), zenity.Filename(dir))
		if err == nil {
			a.update(func() { a.setup.dir.SetText(path) })
		}
	}()
}

func (a *App) addFiles(paths []string) {
	for _, p := range paths {
		p = strings.TrimSpace(p)
		info, err := os.Stat(p)
		switch {
		case err != nil || info.IsDir():
			continue
		case info.Size() > MaxUpload:
			a.notify(a.t("upload.tooBig", filepath.Base(p)))
			continue
		}
		dup := false
		for _, t := range a.attachments {
			dup = dup || t.Path == p
		}
		if dup {
			continue
		}
		t := &Attachment{Path: p, Name: filepath.Base(p), Size: info.Size(),
			Ext: strings.ToLower(strings.TrimPrefix(filepath.Ext(p), "."))}
		a.attachments = append(a.attachments, t)
		go a.upload(t)
	}
	a.view, a.focus = viewChat, true
}

// upload fakes a transfer lasting ~1 to 3 s, and decodes image thumbnails.
func (a *App) upload(t *Attachment) {
	if t.IsImage() {
		if img, ok := preview.DecodeImage(t.Path); ok {
			op := paint.NewImageOp(img)
			a.update(func() { t.thumb, t.ImageSize = &op, img.Bounds().Size() })
		}
	}
	steps := 20 + int(t.Size>>16)%20
	for i := 1; i <= steps; i++ {
		time.Sleep(time.Duration(40+rand.IntN(50)) * time.Millisecond)
		a.update(func() { t.progress = float32(i) / float32(steps) })
	}
}

func (a *App) handleAttachments(gtx C) {
	if a.attachBtn.Clicked(gtx) {
		a.pickFiles()
	}
	for i := 0; i < len(a.attachments); i++ {
		t := a.attachments[i]
		if t.remove.Clicked(gtx) {
			a.attachments = append(a.attachments[:i:i], a.attachments[i+1:]...)
			i--
			continue
		}
		if t.open.Clicked(gtx) {
			a.showPreview(preview.Open(t.Path, ""))
		}
	}
	for _, m := range a.cur.Messages {
		for _, t := range m.Attachments {
			if t.open.Clicked(gtx) {
				a.showPreview(preview.Open(t.Path, ""))
			}
		}
	}
}

func (a *App) attachButton(gtx C) D {
	th := a.th
	bg, fg := th.Surface, th.Muted
	if a.picking {
		bg, fg = th.Hover, th.Text
	}
	return th.RoundButton(gtx, &a.attachBtn, bg, fg, ui.Paperclip)
}

// fileBadge is the image thumbnail, or the extension on a color hinting at
// the file type.
func (a *App) fileBadge(gtx C, t *Attachment) D {
	th := a.th
	s := gtx.Dp(30)
	gtx.Constraints = layout.Exact(image.Pt(s, s))
	r := clip.UniformRRect(image.Rect(0, 0, s, s), gtx.Dp(6))
	if t.thumb != nil {
		defer r.Push(gtx.Ops).Pop()
		return widget.Image{Src: *t.thumb, Fit: widget.Cover}.Layout(gtx)
	}
	bg := th.Accent
	switch t.Ext {
	case "pdf":
		bg = th.Danger
	case "png", "jpg", "jpeg", "gif":
		bg = th.Warning
	case "go", "py", "js", "ts", "rs":
		bg = th.SuccessStrong
	}
	paint.FillShape(gtx.Ops, bg, r.Op(gtx.Ops))
	ext := strings.ToUpper(t.Ext)
	if len(ext) > 4 {
		ext = ext[:4]
	}
	if ext == "" {
		ext = "?"
	}
	return layout.Center.Layout(gtx, th.Label(9, ext, th.OnStrong, font.Bold).Layout)
}

// attachmentChip shows a file; removable chips belong to the composer.
func (a *App) attachmentChip(gtx C, t *Attachment, removable bool) D {
	th := a.th
	macro := op.Record(gtx.Ops)
	d := ui.Bordered(gtx, th.Bg, th.Border, 8, func(gtx C) D {
		return layout.Inset{Top: 6, Bottom: 6, Left: 6, Right: 8}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return t.open.Layout(gtx, func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D { return a.fileBadge(gtx, t) }),
							ui.HSpace(8),
							layout.Rigid(func(gtx C) D {
								gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(170))
								meta := HumanSize(t.Size, a.tr)
								if t.uploading() {
									meta = a.t("upload.progress", int(t.progress*100))
								}
								c := th.Text
								if t.open.Hovered() {
									c = th.Accent
								}
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx C) D { return ui.OneLine(gtx, th.Label(13, t.Name, c, font.Medium)) }),
									layout.Rigid(th.Label(11, meta, th.Muted, font.Normal).Layout),
								)
							}),
						)
					})
				}),
				layout.Rigid(func(gtx C) D {
					if !removable {
						return D{}
					}
					return layout.Inset{Left: 8}.Layout(gtx, func(gtx C) D {
						return t.remove.Layout(gtx, func(gtx C) D {
							c := th.Muted
							if t.remove.Hovered() {
								c = th.Text
							}
							return layout.UniformInset(3).Layout(gtx, func(gtx C) D {
								gtx.Metric.PxPerDp *= 0.8
								return ui.Close(gtx, c)
							})
						})
					})
				}),
			)
		})
	})
	call := macro.Stop()
	call.Add(gtx.Ops)
	if t.uploading() {
		// Progress bar along the bottom edge of the chip.
		y := d.Size.Y - gtx.Dp(3)
		x0, x1 := gtx.Dp(6), d.Size.X-gtx.Dp(6)
		paint.FillShape(gtx.Ops, th.Border, clip.UniformRRect(image.Rect(x0, y, x1, y+gtx.Dp(2)), gtx.Dp(1)).Op(gtx.Ops))
		xp := x0 + int(float32(x1-x0)*t.progress)
		paint.FillShape(gtx.Ops, th.Accent, clip.UniformRRect(image.Rect(x0, y, xp, y+gtx.Dp(2)), gtx.Dp(1)).Op(gtx.Ops))
	}
	return d
}

func (a *App) attachmentFlow(gtx C, atts []*Attachment, removable bool) D {
	items := make([]layout.Widget, len(atts))
	for i, t := range atts {
		items[i] = func(gtx C) D { return a.attachmentChip(gtx, t, removable) }
	}
	return ui.Flow(gtx, 8, items)
}

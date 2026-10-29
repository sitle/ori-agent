// Package preview renders read-only documents in a pane: Markdown (the block
// subset READMEs use), source code with line numbers, PDF (rasterized by
// PDFium compiled to WebAssembly, pure Go) and images.
package preview

import (
	"fmt"
	"image"
	_ "image/gif" // decoders for image documents
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/ui"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type Kind int

const (
	Code Kind = iota
	Markdown
	PDF
	Image
)

// Doc is an opened document. Layout draws the whole pane, header included;
// check Close.Clicked to know when the user closed it.
type Doc struct {
	Name, Path string
	// Kind may change while Load converts the document (OpenWith): read it
	// on the goroutine that draws the pane.
	Kind  Kind
	Close widget.Clickable
	// T translates the pane's texts (keys "preview.*"); nil means English.
	T i18n.Translator

	lines  []string     // Code
	blocks []ui.MDBlock // Markdown

	img     paint.ImageOp // Image
	imgSize image.Point

	pdfData []byte
	conv    Converter  // converts the file in Load (OpenWith)
	text    []string   // an SVG's source, shown if conv does not draw it
	label   string     // kind shown in the header, from the converter
	mu      sync.Mutex // guards Kind once Load started, and the fields below
	loading bool
	pages   []paint.ImageOp
	sizes   []image.Point
	err     string

	list widget.List
}

// Converted is what a Converter makes of a file the pane cannot show by
// itself: the pages of a PDF, or Markdown text.
type Converted struct {
	PDF      []byte
	Markdown string
	// Label is the kind shown in the header ("Word", "Excel"); the page
	// count follows it for a PDF.
	Label string
}

// Converter renders, in the background, a document the pane does not read
// (a Word file, a spreadsheet, a presentation). It returns nil for a file
// it does not handle, which is then shown as a binary file.
type Converter func(path string) (*Converted, error)

func ext(path string) string { return strings.ToLower(strings.TrimPrefix(filepath.Ext(path), ".")) }

// IsImage reports whether an extension (without dot) is a supported image.
func IsImage(ext string) bool {
	return ext == "png" || ext == "jpg" || ext == "jpeg" || ext == "gif"
}

func DecodeImage(path string) (image.Image, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err == nil
}

// Open loads path. For text documents, fallback is shown when the file can't
// be read (nil is returned if there is no fallback either). PDF pages are
// rendered by Load.
func Open(path, fallback string) *Doc { return OpenWith(path, fallback, nil) }

// OpenWith is Open, with conv for the files that are not text, images or
// PDF, and for SVG drawings, whose source is shown if conv returns nil:
// Load converts them in the background.
func OpenWith(path, fallback string, conv Converter) *Doc {
	d := &Doc{Name: filepath.Base(path), Path: path}
	d.list.Axis = layout.Vertical
	switch e := ext(path); {
	case IsImage(e):
		img, ok := DecodeImage(path)
		if !ok {
			return nil
		}
		d.Kind, d.img, d.imgSize = Image, paint.NewImageOp(img), img.Bounds().Size()
		return d
	case e == "pdf":
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		d.Kind, d.pdfData, d.loading = PDF, data, true
		return d
	}
	src := fallback
	if data, err := os.ReadFile(path); err == nil {
		if len(data) > 256<<10 {
			data = data[:256<<10]
			// The cut may fall inside a character: drop its first bytes, so
			// that a long text is not taken for a binary file.
			for i := 0; i < utf8.UTFMax-1 && !utf8.Valid(data); i++ {
				data = data[:len(data)-1]
			}
		}
		if utf8.Valid(data) && ext(path) == "svg" && conv != nil {
			text := strings.ReplaceAll(strings.TrimRight(string(data), "\n"), "\r\n", "\n")
			d.Kind, d.conv, d.loading, d.text = PDF, conv, true, strings.Split(text, "\n")
			return d
		}
		if utf8.Valid(data) {
			src = string(data)
		} else if conv != nil {
			d.Kind, d.conv, d.loading = PDF, conv, true
			return d
		} else {
			src = "(binary file)"
		}
	} else if fallback == "" {
		return nil
	}
	src = strings.ReplaceAll(strings.ReplaceAll(src, "\r\n", "\n"), "\t", "    ")
	if e := ext(path); e == "md" || e == "markdown" {
		d.Kind, d.blocks = Markdown, ui.ParseMarkdown(src)
	} else {
		d.lines = strings.Split(strings.TrimRight(src, "\n"), "\n")
	}
	return d
}

// Load renders PDF pages (or converts the document first) in the
// background and calls invalidate when done. It does nothing for other
// kinds.
func (d *Doc) Load(invalidate func()) {
	if d.Kind != PDF {
		return
	}
	go func() {
		if d.conv != nil && !d.convert() {
			invalidate()
			return
		}
		// 144 DPI gives ~1200px wide A4 pages: sharp once scaled to the pane.
		imgs, err := RenderPDF(d.pdfData, 144)
		d.mu.Lock()
		d.loading = false
		for _, img := range imgs {
			d.pages = append(d.pages, paint.NewImageOp(img))
			d.sizes = append(d.sizes, img.Bounds().Size())
		}
		if err != nil {
			d.err = err.Error()
		}
		d.mu.Unlock()
		invalidate()
	}()
}

// convert runs the converter; it reports whether there is a PDF to render.
func (d *Doc) convert() bool {
	c, err := d.conv(d.Path)
	d.mu.Lock()
	defer d.mu.Unlock()
	switch {
	case (err != nil || c == nil) && d.text != nil:
		// An SVG the application does not draw: its source.
		d.Kind, d.loading, d.lines = Code, false, d.text
		return false
	case err != nil:
		d.loading, d.err = false, err.Error()
		return false
	case c == nil:
		d.Kind, d.loading, d.lines = Code, false, []string{"(binary file)"}
		return false
	case len(c.PDF) > 0:
		d.pdfData, d.label = c.PDF, c.Label
		return true
	default:
		src := strings.ReplaceAll(c.Markdown, "\r\n", "\n")
		d.Kind, d.loading, d.blocks, d.label = Markdown, false, ui.ParseMarkdown(src), c.Label
		return false
	}
}

var lineNumber = regexp.MustCompile(`^\s*\d+  `)

// StripLineNumbers undoes "   1  " prefixes, as in a Read tool output.
func StripLineNumbers(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = lineNumber.ReplaceAllString(l, "")
	}
	return strings.Join(lines, "\n")
}

var english = map[string]string{
	"preview.code":         "Code",
	"preview.markdown":     "Markdown",
	"preview.image":        "Image · %d×%d",
	"preview.pdf":          "PDF",
	"preview.pdfPages":     "PDF · %d pp.",
	"preview.pages":        "%d pp.",
	"preview.rendering":    "Rendering PDF…",
	"preview.error":        "Could not read the PDF: %s",
	"preview.converting":   "Converting the document…",
	"preview.convertError": "Could not convert the document: %s",
}

func (d *Doc) t(key string, args ...any) string {
	if d.T != nil {
		return d.T.T(key, args...)
	}
	return fmt.Sprintf(english[key], args...)
}

func (d *Doc) kindLabel() string {
	d.mu.Lock()
	kind, label, n := d.Kind, d.label, len(d.pages)
	d.mu.Unlock()
	if label != "" {
		if kind == PDF && n > 0 {
			return label + " · " + d.t("preview.pages", n)
		}
		return label
	}
	if kind == PDF && d.conv != nil && n == 0 {
		// Still converting, or failed: the file's own kind, not "PDF".
		return strings.ToUpper(ext(d.Path))
	}
	switch kind {
	case Markdown:
		return d.t("preview.markdown")
	case Image:
		return d.t("preview.image", d.imgSize.X, d.imgSize.Y)
	case PDF:
		if n > 0 {
			return d.t("preview.pdfPages", n)
		}
		return d.t("preview.pdf")
	}
	return d.t("preview.code")
}

// Layout draws the pane: header (name, path, kind, close) and content.
func (d *Doc) Layout(gtx C, th *ui.Theme) D {
	paint.FillShape(gtx.Ops, th.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Inset{Top: 8, Bottom: 8, Left: 16, Right: 12}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(layout.Spacer{Height: 30}.Layout),
					layout.Rigid(th.Label(14, d.Name, th.Text, font.SemiBold).Layout),
					ui.HSpace(10),
					layout.Flexed(1, func(gtx C) D { return ui.OneLine(gtx, th.Mono(11, d.Path, th.Muted)) }),
					ui.HSpace(8),
					layout.Rigid(func(gtx C) D { return th.Chip(gtx, d.kindLabel(), th.Muted) }),
					ui.HSpace(6),
					layout.Rigid(func(gtx C) D {
						return d.Close.Layout(gtx, func(gtx C) D {
							bg, c := th.Bg, th.Muted
							if d.Close.Hovered() {
								bg, c = th.Hover, th.Text
							}
							return ui.Card(gtx, bg, 6, layout.UniformInset(9), func(gtx C) D { return ui.Close(gtx, c) })
						})
					}),
				)
			})
		}),
		layout.Rigid(th.HLine),
		layout.Flexed(1, func(gtx C) D {
			d.mu.Lock()
			kind := d.Kind
			d.mu.Unlock()
			switch kind {
			case PDF:
				return d.layoutPDF(gtx, th)
			case Image:
				return d.layoutImage(gtx, th)
			case Markdown:
				return material.List(th.Material, &d.list).Layout(gtx, len(d.blocks), func(gtx C, i int) D {
					in := layout.Inset{Left: 28, Right: 28}
					if i == 0 {
						in.Top = 20
					}
					if i == len(d.blocks)-1 {
						in.Bottom = 24
					}
					return in.Layout(gtx, func(gtx C) D { return th.MarkdownBlock(gtx, 14, d.blocks[i]) })
				})
			}
			return material.List(th.Material, &d.list).Layout(gtx, len(d.lines), func(gtx C, i int) D {
				in := layout.Inset{Left: 8, Right: 16}
				if i == 0 {
					in.Top = 12
				}
				return in.Layout(gtx, func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(40), gtx.Dp(40)
							l := th.Mono(13, itoa(i+1), th.Muted)
							l.Alignment = text.End
							return l.Layout(gtx)
						}),
						ui.HSpace(14),
						layout.Flexed(1, th.Mono(13, d.lines[i], th.Text).Layout),
					)
				})
			})
		}),
	)
}

func (d *Doc) layoutImage(gtx C, th *ui.Theme) D {
	paint.FillShape(gtx.Ops, th.Code, clip.Rect{Max: gtx.Constraints.Max}.Op())
	return layout.UniformInset(20).Layout(gtx, func(gtx C) D {
		// Fit inside the pane without upscaling small images.
		sz := gtx.Constraints.Max
		scale := min(float32(sz.X)/float32(d.imgSize.X), float32(sz.Y)/float32(d.imgSize.Y), 1)
		fit := image.Pt(int(float32(d.imgSize.X)*scale), int(float32(d.imgSize.Y)*scale))
		return layout.Center.Layout(gtx, func(gtx C) D {
			gtx.Constraints = layout.Exact(fit)
			return widget.Image{Src: d.img, Fit: widget.Fill}.Layout(gtx)
		})
	})
}

func (d *Doc) layoutPDF(gtx C, th *ui.Theme) D {
	paint.FillShape(gtx.Ops, th.Code, clip.Rect{Max: gtx.Constraints.Max}.Op())
	d.mu.Lock()
	loading, pages, sizes, errMsg := d.loading, d.pages, d.sizes, d.err
	converting := d.conv != nil && d.pdfData == nil
	d.mu.Unlock()
	if loading || len(pages) == 0 {
		return layout.Center.Layout(gtx, func(gtx C) D {
			if errMsg != "" {
				key := "preview.error"
				if d.conv != nil {
					key = "preview.convertError"
				}
				return layout.UniformInset(24).Layout(gtx, th.Label(13, d.t(key, errMsg), th.DiffDel, font.Normal).Layout)
			}
			msg := d.t("preview.rendering")
			if converting {
				msg = d.t("preview.converting")
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return ui.Dot(gtx, ui.Pulse(gtx, th.Accent), 9) }),
				ui.HSpace(10),
				layout.Rigid(th.Label(13, msg, th.Muted, font.Normal).Layout),
			)
		})
	}
	return material.List(th.Material, &d.list).Layout(gtx, len(pages), func(gtx C, i int) D {
		in := layout.Inset{Left: 20, Right: 20, Bottom: 16}
		if i == 0 {
			in.Top = 20
		}
		return in.Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					w := gtx.Constraints.Max.X
					h := w * sizes[i].Y / sizes[i].X
					return ui.Bordered(gtx, th.Bg, th.Border, 1, func(gtx C) D {
						gtx.Constraints = layout.Exact(image.Pt(w-2, h-2))
						return widget.Image{Src: pages[i], Fit: widget.Fill}.Layout(gtx)
					})
				}),
				ui.VSpace(6),
				layout.Rigid(th.Label(11, itoa(i+1)+" / "+itoa(len(pages)), th.Muted, font.Normal).Layout),
			)
		})
	})
}

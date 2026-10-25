package preview

import (
	"image"
	"image/draw"
	"sync"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
)

// PDFs are rendered by PDFium compiled to WebAssembly and run by wazero:
// pure Go, so no CGO or system library is needed on any OS.
var (
	pdfOnce sync.Once
	pdfPool pdfium.Pool
	pdfErr  error
)

const maxPDFPages = 50

// RenderPDF rasterizes the pages of a PDF at the given DPI. The WebAssembly
// runtime is started on first use, which takes about a second.
func RenderPDF(data []byte, dpi int) ([]*image.RGBA, error) {
	pdfOnce.Do(func() {
		pdfPool, pdfErr = webassembly.Init(webassembly.Config{MinIdle: 1, MaxIdle: 1, MaxTotal: 1})
	})
	if pdfErr != nil {
		return nil, pdfErr
	}
	inst, err := pdfPool.GetInstance(30 * time.Second)
	if err != nil {
		return nil, err
	}
	defer inst.Close()

	doc, err := inst.OpenDocument(&requests.OpenDocument{File: &data})
	if err != nil {
		return nil, err
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	count, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return nil, err
	}
	var pages []*image.RGBA
	for i := 0; i < min(count.PageCount, maxPDFPages); i++ {
		r, err := inst.RenderPageInDPI(&requests.RenderPageInDPI{
			DPI:  dpi,
			Page: requests.Page{ByIndex: &requests.PageByIndex{Document: doc.Document, Index: i}},
		})
		if err != nil {
			return pages, err
		}
		// Copy out: the WebAssembly pixel buffer is released by Cleanup.
		src := r.Result.RenderedImage
		dst := image.NewRGBA(src.Bounds())
		draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
		r.Cleanup()
		pages = append(pages, dst)
	}
	return pages, nil
}

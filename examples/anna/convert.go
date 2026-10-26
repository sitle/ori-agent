package main

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"path/filepath"
	"strings"

	"github.com/sitle/ori-agent/preview"
)

// convert is Config.Convert: a Word document (.docx) is shown as Markdown,
// read from its XML without any dependency (headings, lists, paragraphs).
// Other formats stay binary; an application could hand them to LibreOffice
// (soffice --headless --convert-to pdf) and return the PDF.
func convert(path string) (*preview.Converted, error) {
	if !strings.EqualFold(filepath.Ext(path), ".docx") {
		return nil, nil
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	f, err := z.Open("word/document.xml")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	md, err := docxMarkdown(f)
	if err != nil {
		return nil, err
	}
	return &preview.Converted{Markdown: md, Label: "Word"}, nil
}

// docxMarkdown turns the paragraphs of a word/document.xml into Markdown.
func docxMarkdown(r io.Reader) (string, error) {
	var out, par strings.Builder
	var style string
	var list, inText bool
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return strings.TrimSpace(out.String()) + "\n", nil
		}
		if err != nil {
			return "", err
		}
		switch e := tok.(type) {
		case xml.StartElement:
			switch e.Name.Local {
			case "p":
				par.Reset()
				style, list = "", false
			case "pStyle":
				for _, a := range e.Attr {
					if a.Name.Local == "val" {
						style = strings.ToLower(a.Value)
					}
				}
			case "numPr":
				list = true
			case "t":
				inText = true
			case "tab":
				par.WriteString(" ")
			case "br":
				par.WriteString("  \n")
			}
		case xml.CharData:
			if inText {
				par.Write(e)
			}
		case xml.EndElement:
			switch e.Name.Local {
			case "t":
				inText = false
			case "p":
				text := strings.TrimSpace(par.String())
				if text == "" {
					continue
				}
				out.WriteString(prefix(style, list) + text + "\n\n")
			}
		}
	}
}

// prefix is the Markdown mark of a paragraph style: Word's style ids are
// "Heading1" or, in a French Word, "Titre1"; "Title" is the document title.
func prefix(style string, list bool) string {
	for _, name := range []string{"heading", "titre"} {
		if n, ok := strings.CutPrefix(style, name); ok && len(n) == 1 && n[0] >= '1' && n[0] <= '6' {
			return strings.Repeat("#", int(n[0]-'0')) + " "
		}
	}
	switch {
	case style == "title" || style == "titre":
		return "# "
	case list:
		return "- "
	}
	return ""
}

package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

const docxBody = `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>
<w:p><w:pPr><w:pStyle w:val="Titre1"/></w:pPr><w:r><w:t>Démarches en ligne</w:t></w:r></w:p>
<w:p><w:r><w:t xml:space="preserve">Le formulaire est </w:t></w:r><w:r><w:t>prêt.</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/></w:numPr></w:pPr><w:r><w:t>Valider les champs</w:t></w:r></w:p>
<w:p></w:p>
</w:body></w:document>`

func writeDocx(t *testing.T, path, body string) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	w, _ := z.Create("word/document.xml")
	w.Write([]byte(body))
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestConvertDocx(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.docx")
	writeDocx(t, path, docxBody)
	c, err := convert(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "# Démarches en ligne\n\nLe formulaire est prêt.\n\n- Valider les champs\n"
	if c == nil || c.Markdown != want || c.Label != "Word" {
		t.Fatalf("converted %+v, want %q", c, want)
	}
	if c, err := convert(filepath.Join(t.TempDir(), "tableau.xlsx")); c != nil || err != nil {
		t.Errorf("other formats are not handled: %+v %v", c, err)
	}
}

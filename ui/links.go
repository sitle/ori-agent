package ui

import (
	"image"
	"strings"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/op/clip"
)

// Links of a text drawn by RichText: [text](url), or an http(s) address
// written as is. With Theme.OpenLink set, a click on one opens it.

// linkTag receives the clicks on the words of a link; every word of the
// same address shares it, from one frame to the next.
type linkTag string

// BareURL is the http or https address in word, without the brackets,
// quotes and punctuation around it; empty if there is none.
func BareURL(word string) string {
	word = strings.TrimLeft(word, "(<«\"'")
	if !strings.HasPrefix(word, "https://") && !strings.HasPrefix(word, "http://") {
		return ""
	}
	url := strings.TrimRight(word, ".,;:!?»\"'>")
	if strings.HasSuffix(url, ")") && strings.Count(url, "(") < strings.Count(url, ")") {
		url = strings.TrimSuffix(url, ")")
	}
	if strings.HasSuffix(url, "://") {
		return ""
	}
	return url
}

// linkArea makes the word of size, drawn at the current offset, a link to
// url: a hand cursor over it, and OpenLink on a click.
func (t *Theme) linkArea(gtx C, size image.Point, url string) {
	tag := linkTag(url)
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: tag, Kinds: pointer.Release})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && e.Kind == pointer.Release {
			t.OpenLink(url)
		}
	}
	area := clip.Rect(image.Rectangle{Max: size}).Push(gtx.Ops)
	pointer.CursorPointer.Add(gtx.Ops)
	event.Op(gtx.Ops, tag)
	area.Pop()
}

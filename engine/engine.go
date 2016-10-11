package engine

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"strings"
)

var _ = fmt.Println

type tracker struct {
	output     string
	tagContent []string

	tokenizer *html.Tokenizer
}

func newTracker(r io.Reader) *tracker {
	return &tracker{
		tokenizer:  html.NewTokenizer(r),
		tagContent: make([]string, 0),
	}
}

func (t *tracker) feed() {
	raw := string(t.tokenizer.Raw())
	nameBytes, _ := t.tokenizer.TagName()
	token := t.tokenizer.Token()
	name := string(nameBytes)

	// TODO - Make template tag configurable
	isTempTag := (len(name) >= 2 && name[0:2] == "t:")

	depth := len(t.tagContent)

	if isTempTag {
		// If tag is a defined template tag
		if token.Type == html.StartTagToken {
			// And is a start tag increse our store size
			t.tagContent = append(t.tagContent, "")
			depth++
		} else if token.Type == html.EndTagToken {
			// If an end tag get the parsed content and add to the
			// parent store or the output if no parent.
			if depth > 1 {
				t.tagContent[depth-2] += t.tagContent[depth-1]
			} else {
				t.output += t.tagContent[0]
			}
			t.tagContent = t.tagContent[:depth-1]
			depth--
		}
	} else if depth > 0 {
		// If not a template tag and we are in a template tag
		// add the content to the latest tag store
		t.tagContent[depth-1] += raw
	}

	// If the tag is not a template and we are not in a template now
	// add the content directly to the output.
	if !isTempTag && depth == 0 {
		t.output += raw
	}
}

// func (t *tracker) updateContentMap(s string) {
// 	for i := range t.tagContent {
// 		t.tagContent[key] += s
// 	}
// }

func ParseString(input string) (string, error) {
	r := strings.NewReader(input)
	tracker := newTracker(r)

	for {
		tt := tracker.tokenizer.Next()
		if tt == html.ErrorToken {
			return tracker.output, nil
		}
		tracker.feed()
	}

	return tracker.output, nil

}

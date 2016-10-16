package engine

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"path/filepath"
)

var _ = fmt.Println

type tracker struct {
	reader            io.Reader
	writer            io.Writer
	tags              []*templateTag
	contextFile       string
	contextFolderPath string
	vars              map[string]string
	templateContent   map[string]string
	tokenizer         *html.Tokenizer

	errChan chan error
}

type templateTag struct {
	reader  io.Reader
	writer  io.Writer
	name    string
	tracker *tracker
}

func newTracker(r io.Reader, contextFile string, parent *tracker) *tracker {
	t := &tracker{
		tags:              make([]*templateTag, 0),
		contextFile:       contextFile,
		contextFolderPath: filepath.Dir(contextFile),
	}

	// Copy over any parent vars and copy the reference to out template cache
	t.vars = make(map[string]string)
	if parent != nil {
		for k, v := range parent.vars {
			t.vars[k] = v
		}
		t.templateContent = parent.templateContent
		t.errChan = parent.errChan
	} else {
		t.templateContent = make(map[string]string)
	}

	t.reader, t.writer = io.Pipe()

	// Preparse template and create HTML tokenizer
	contentReader, _ := t.preParseTemplate(r)
	t.tokenizer = html.NewTokenizer(contentReader)
	return t
}

// Adds a new template tag to the stack
func (t *tracker) addTemplateTag(tagName string) *templateTag {
	templateName := tagName[2:]
	tag := &templateTag{name: templateName, tracker: t}
	t.reader, t.writer = io.Pipe()
	t.tags = append(t.tags, tag)
	return tag
}

// Closes the last template tag off and parses the content
// into the next latest template tag or, if not mor tags exist,
// adds the tag content to the output
func (t *tracker) closeTemplateTag() error {
	var closingTag *templateTag

	if t.depth() > 1 {
		closingTag = t.tags[t.depth()-1]
	} else {
		closingTag = t.tags[0]
	}

	content, err := closingTag.parseTemplate()
	if err != nil {
		return err
	}

	if t.depth() > 1 {
		_, err = io.Copy(t.tags[t.depth()-2].writer, content)
	} else {
		_, err = io.Copy(t.writer, content)
	}
	// Drop the last tag in the tracker
	t.tags = t.tags[:t.depth()-1]
	return err
}

// Get the current nesting depth of template tags
func (t *tracker) depth() int {
	return len(t.tags)
}

// feed in another token from the tag tokenizer.
func (t *tracker) feed() (string, error) {
	raw := t.tokenizer.Raw()
	nameBytes, _ := t.tokenizer.TagName()
	token := t.tokenizer.Token()
	name := string(nameBytes)

	// TODO - Make template tag configurable
	isTempTag := (len(name) >= 2 && name[0:2] == "t:")

	if isTempTag && token.Type == html.StartTagToken {
		t.addTemplateTag(name)
	} else if isTempTag && token.Type == html.EndTagToken {
		err := t.closeTemplateTag()
		if err != nil {
			return name, err
		}
	} else if t.depth() > 0 {
		// If not a template tag and we are in a template tag
		// add the content to the latest tag store
		t.tags[t.depth()-1].writer.Write(raw)
	}

	// If the tag is not a template and we are not in a template now
	// add the content directly to the output.
	if !isTempTag && t.depth() == 0 {
		t.writer.Write(raw)
	}
	return name, nil
}

func (t *tracker) parse() io.Reader {
	go func() {
		count := 0
		for {
			tt := t.tokenizer.Next()
			if tt == html.ErrorToken {
				t.errChan <- nil
			}
			_, err := t.feed()

			if err != nil {
				t.errChan <- err
			}
			count++
		}
	}()
	return t.reader
}

func Parse(r io.Reader, fileLocation string) (io.Reader, <-chan error) {
	tracker := newTracker(r, fileLocation, nil)
	return tracker.parse(), tracker.errChan
}

func parseChild(r io.Reader, fileLocation string, tracker *tracker) (io.Reader, <-chan error) {
	newTracker := newTracker(r, fileLocation, tracker)
	return newTracker.parse(), newTracker.errChan
}

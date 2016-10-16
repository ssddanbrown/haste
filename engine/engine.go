package engine

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"path/filepath"
	"strings"
)

var _ = fmt.Println

type tracker struct {
	output            string
	tags              []*templateTag
	contextFile       string
	contextFolderPath string
	vars              map[string]string

	templateContent     map[string]string
	parsedTemplateCache map[string]string

	tokenizer *html.Tokenizer
}

type templateTag struct {
	name    string
	content string
	tracker *tracker
}

func newTracker(r io.Reader, contextFile string, parent *tracker) (*tracker, error) {
	t := &tracker{
		tags:              make([]*templateTag, 0),
		contextFile:       contextFile,
		contextFolderPath: filepath.Dir(contextFile),
	}

	t.parsedTemplateCache = make(map[string]string)
	// Copy over any parent vars
	t.vars = make(map[string]string)
	if parent != nil {
		for k, v := range parent.vars {
			t.vars[k] = v
		}
		t.templateContent = parent.templateContent
	} else {
		t.templateContent = make(map[string]string)
	}

	// Preparse template and create HTML tokenizer
	content, err := t.preParseTemplate(r)
	t.tokenizer = html.NewTokenizer(strings.NewReader(content))
	return t, err
}

// Adds a new template tag to the stack
func (t *tracker) addTemplateTag(tagName string) *templateTag {
	templateName := tagName[2:]
	tag := &templateTag{name: templateName, tracker: t}
	t.tags = append(t.tags, tag)
	return tag
}

// Closes the last template tag off and parses the content
// into the next latest template tag or, if not mor tags exist,
// adds the tag content to the output
func (t *tracker) closeTemplateTag() (err error) {
	var closingTag *templateTag

	if t.depth() > 1 {
		closingTag = t.tags[t.depth()-1]
	} else {
		closingTag = t.tags[0]
	}

	key := t.contextFile + ":" + closingTag.name + ":" + closingTag.content
	var content string
	if val, ok := t.parsedTemplateCache[key]; ok {
		content = val
	} else {
		content, err = closingTag.parseTemplate()
		if err != nil {
			return
		}
		t.parsedTemplateCache[key] = content
	}

	if t.depth() > 1 {
		t.tags[t.depth()-2].content += content
	} else {
		t.output += content
	}
	// Drop the last tag in the tracker
	t.tags = t.tags[:t.depth()-1]
	return
}

// Get the current nesting depth of template tags
func (t *tracker) depth() int {
	return len(t.tags)
}

// feed in another token from the tag tokenizer.
func (t *tracker) feed() (string, error) {
	raw := string(t.tokenizer.Raw())
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
		t.tags[t.depth()-1].content += raw
	}

	// If the tag is not a template and we are not in a template now
	// add the content directly to the output.
	if !isTempTag && t.depth() == 0 {
		t.output += raw
	}
	return name, nil
}

func (t *tracker) parse() (string, error) {
	count := 0
	for {
		tt := t.tokenizer.Next()
		if tt == html.ErrorToken {
			return t.parseVariableTags(t.output), nil
		}
		_, err := t.feed()

		if err != nil {
			return "", err
		}
		count++
	}
}

func Parse(r io.Reader, fileLocation string) (string, error) {
	tracker, err := newTracker(r, fileLocation, nil)
	if err != nil {
		return "", err
	}

	s, err := tracker.parse()
	return s, err
}

func parseChild(r io.Reader, fileLocation string, tracker *tracker) (string, error) {
	tracker, err := newTracker(r, fileLocation, tracker)
	if err != nil {
		return "", err
	}

	return tracker.parse()
}

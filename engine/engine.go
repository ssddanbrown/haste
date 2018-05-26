package engine

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
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
	name        string
	content     string
	tracker     *tracker
	contentType string
	attrs       []*html.Attribute
}

func newTracker(r io.Reader, contextFile string, contextFolder string, parent *tracker) (*tracker, error) {
	t := &tracker{
		tags:              make([]*templateTag, 0),
		contextFile:       contextFile,
		contextFolderPath: contextFolder,
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
func (t *tracker) addTemplateTag(tagName string, attrs []*html.Attribute) *templateTag {
	templateName := tagName[2:]
	tag := &templateTag{name: templateName, tracker: t, attrs: attrs}
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

	attrHash := ""
	for i := 0; i < len(closingTag.attrs); i++ {
		attrHash += fmt.Sprint("[%s=%s]", closingTag.attrs[i].Key, closingTag.attrs[i].Val)
	}
	key := t.contextFile + ":" + closingTag.name + ":" + closingTag.content + ":" + attrHash

	var content string
	if val, ok := t.parsedTemplateCache[key]; ok {
		content = val
	} else {
		content, err = closingTag.parseTemplate(closingTag.attrs)
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
	nameBytes, hasAttr := t.tokenizer.TagName()
	name := string(nameBytes)

	// TODO - Make template tag configurable
	isTempTag := (len(name) >= 2 && name[0:2] == "t:")

	if !isTempTag {
		if t.depth() > 0 {
			// If not a template tag and we are in a template tag
			// add the content to the latest tag store
			t.tags[t.depth()-1].content += raw
		} else {
			t.output += raw
		}
		return name, nil
	}

	// TODO - optimize to prevent running on every tag
	var attrs []*html.Attribute
	if hasAttr {
		for {
			key, val, hasMore := t.tokenizer.TagAttr()
			newAttribute := &html.Attribute{Key: string(key), Val: string(val)}
			attrs = append(attrs, newAttribute)
			if !hasMore {
				break
			}
		}
	}

	token := t.tokenizer.Token()

	if isTempTag && token.Type == html.StartTagToken {
		t.addTemplateTag(name, attrs)
	} else if isTempTag && token.Type == html.EndTagToken {
		err := t.closeTemplateTag()
		if err != nil {
			return name, err
		}
	} else if isTempTag && token.Type == html.SelfClosingTagToken {
		t.addTemplateTag(name, attrs)
		err := t.closeTemplateTag()
		if err != nil {
			return name, err
		}
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

// Parse the contents of the reader and feed the output as a compiled
// and complete string.
func Parse(r io.Reader, fileLocation string, rootFolder string) (string, error) {
	tracker, err := newTracker(r, fileLocation, rootFolder, nil)
	if err != nil {
		return "", err
	}

	return tracker.parse()
}

func parseChild(r io.Reader, fileLocation string, contextFolder string, tracker *tracker, attrs []*html.Attribute) (string, error) {
	t, err := newTracker(r, fileLocation, contextFolder, tracker)
	for i := 0; i < len(attrs); i++ {
		t.vars[attrs[i].Key] = attrs[i].Val
	}

	if err != nil {
		return "", err
	}

	return t.parse()
}

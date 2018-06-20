package engine

import (
	"bufio"
	"bytes"
	"io"

	"golang.org/x/net/html"
)

type Builder struct {
	Reader  io.Reader
	Manager *Manager
	Vars    map[string][]byte
	Content []byte
	FilesParsed map[string]bool

	tagStack []*templateTag
}

func NewBuilder(r io.Reader, m *Manager, parent *Builder) *Builder {
	b := &Builder{
		Manager: m,
		Reader:  r,
	}

	// Create var store and copy over parent vars
	b.Vars = make(map[string][]byte)
	if parent != nil {
		b.mergeVars(parent.Vars)
		b.FilesParsed = parent.FilesParsed
	} else {
		b.FilesParsed = make(map[string]bool)
	}

	return b
}

func (b *Builder) mergeVars(vars map[string][]byte) {
	for k, v := range vars {
		b.Vars[k] = v
	}
}

func (b *Builder) Build() io.Reader {
	r := b.parseTemplateVariables(b.Reader)
	r = b.parseTemplateTags(r)
	return r
}

func (b *Builder) parseTemplateTags(r io.Reader) io.Reader {
	returnReader, writer := io.Pipe()
	tok := html.NewTokenizer(r)
	go func() {
		defer writer.Close()
		for {
			tt := tok.Next()
			if tt == html.ErrorToken {
				return
			}

			b.parseToken(tok, writer)
		}
	}()

	return returnReader
}

func (b *Builder) parseToken(tok *html.Tokenizer, w io.Writer) error {
	var err error
	raw := tok.Raw()
	name, hasAttr := tok.TagName()

	tagPrefix := b.Manager.tagPrefix
	prefixLen := len(tagPrefix)

	isTempTag := len(name) > prefixLen && bytes.Equal(name[0:prefixLen], tagPrefix)

	depth := len(b.tagStack)

	// Write content if normal tag or add to content of last in stack
	if !isTempTag {
		if depth > 0 {
			b.tagStack[depth-1].content = append(b.tagStack[depth-1].content, raw...)
		} else {
			w.Write(raw)
		}
		return err
	}

	tagName := name[prefixLen:]

	// Parse tag attrs as vars
	tagVars := make(map[string][]byte)
	if hasAttr {
		for {
			key, val, hasMore := tok.TagAttr()
			valCopy := make([]byte, len(val))
			copy(valCopy, val)
			tagVars[string(key)] = valCopy
			if !hasMore {
				break
			}
		}
	}

	token := tok.Token()

	if token.Type == html.StartTagToken || token.Type == html.SelfClosingTagToken {
		b.addTemplateTag(tagName, tagVars)
	}

	if token.Type == html.EndTagToken || token.Type == html.SelfClosingTagToken {
		err = b.closeTemplateTag(w)
	}

	return err

}

func (b *Builder) addTemplateTag(tagName []byte, attrs map[string][]byte) *templateTag {
	tag := NewTemplateTag(tagName, attrs)
	b.tagStack = append(b.tagStack, tag)
	return tag
}

// Closes the last template tag off and parses the content
// into the next latest template tag or, if not mor tags exist,
// adds the tag content to the output
func (b *Builder) closeTemplateTag(writer io.Writer) (err error) {
	var closingTag *templateTag

	cDepth := len(b.tagStack)

	if cDepth > 1 {
		closingTag = b.tagStack[cDepth-1]
	} else {
		closingTag = b.tagStack[0]
	}

	content, err := closingTag.Parse(b)
	if err != nil {
		return err
	}

	if closingTag.path != "" {
		b.FilesParsed[closingTag.path] = true
	}

	if cDepth > 1 {
		prevTag := b.tagStack[cDepth-2]
		prevTag.content = append(prevTag.content, content...)
	} else {
		writer.Write(content)
	}

	// Drop the last tag in the tracker
	b.tagStack = b.tagStack[:cDepth-1]
	return err
}

func (b *Builder) parseTemplateVariables(r io.Reader) io.Reader {
	returnReader, w := io.Pipe()

	varChar := byte('@')
	varSep := []byte{'='}

	// Read in variables at the top of file
	scanner := bufio.NewScanner(r)
	var text []byte
	for scanner.Scan() {
		text = scanner.Bytes()

		if text[0] == varChar && len(text) > 1 {
			splitVar := bytes.SplitN(text[1:], varSep, 2)
			if len(splitVar) != 2 {
				continue
			}
			b.Vars[string(splitVar[0])] = splitVar[1]
		} else {
			break
		}

	}

	// Send the remaining content back via reader
	go func() {
		defer w.Close()

		w.Write(text)
		for scanner.Scan() {
			w.Write([]byte{'\n'})
			w.Write(scanner.Bytes())
		}
	}()

	return returnReader
}

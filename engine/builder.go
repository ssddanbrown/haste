package engine

import (
	"bufio"
	"bytes"
	"github.com/ssddanbrown/haste/options"
	"io"
	"io/ioutil"

	"errors"
	"github.com/fatih/color"
	"golang.org/x/net/html"
)

type Builder struct {
	Reader      io.Reader
	Options     *options.Options
	Vars        map[string][]byte
	Content     []byte
	FilesParsed map[string]bool

	tagStack []*templateTag
}

func NewBuilder(r io.Reader, o *options.Options, parent *Builder) *Builder {
	b := &Builder{
		Reader:  r,
		Options: o,
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
	r = parseVariableTags(r, b.Vars, b.Options)
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

			err := b.parseToken(tok, writer)
			if err != nil {
				color.Red("%s", err)
			}
		}
	}()

	return returnReader
}

func (b *Builder) parseToken(tok *html.Tokenizer, w io.Writer) error {
	var err error
	raw := tok.Raw()
	name, hasAttr := tok.TagName()
	depth := len(b.tagStack)

	isTempTag := tagNameHasPrefix(name, b.Options.TagPrefix)
	if isTempTag {
		err = b.parseTemplateTag(name, hasAttr, tok, w)
		return err
	}

	isVarTag := tagNameHasPrefix(name, b.Options.VarTagPrefix)
	if isVarTag {
		// Parse var tag
		err = b.parseVariableTag(name, tok)
		return err
	}

	// Write injectedContent if normal tag or add to injectedContent of last in stack
	if depth > 0 {
		b.tagStack[depth-1].injectedContent = append(b.tagStack[depth-1].injectedContent, raw...)
	} else {
		w.Write(raw)
	}
	return err
}

func tagNameHasPrefix(tagName []byte, prefix []byte) bool {
	prefixLen := len(prefix)
	return len(tagName) > prefixLen-1  && bytes.Equal(tagName[0:prefixLen], prefix)
}

func (b *Builder) parseVariableTag(name []byte, tok *html.Tokenizer) error {
	var err error
	tagName := name[len(b.Options.TagPrefix):]
	token := tok.Token()

	if token.Type == html.StartTagToken || token.Type == html.SelfClosingTagToken {
		b.addVariableTag(tagName)
	}

	if token.Type == html.EndTagToken || token.Type == html.SelfClosingTagToken {
		err = b.closeVariableTag()
	}
	return err
}

func (b *Builder) addVariableTag(tagName []byte) *templateTag {
	tag := NewVariableTag(tagName, b.Options)
	b.tagStack = append(b.tagStack, tag)
	return tag
}

func (b *Builder) closeVariableTag() error {
	var closingTag *templateTag

	cDepth := len(b.tagStack)
	if cDepth < 2 {
		b.tagStack = b.tagStack[:cDepth-1]
		return errors.New("Variable tags can only be used within a template tag")
	}
	parentTag := b.tagStack[cDepth-2]
	if parentTag.tagType == "variable" {
		b.tagStack = b.tagStack[:cDepth-1]
		return errors.New("You cannot directly nest variable tags")
	}

	closingTag = b.tagStack[cDepth-1]

	// Add the injectedContent as an attribute variable of the parent tag
	parentTag.attrs[string(closingTag.name)] = bytes.TrimSpace(closingTag.injectedContent)

	// Drop the last tag in the tracker
	b.tagStack = b.tagStack[:cDepth-1]
	return nil
}

func (b *Builder) parseTemplateTag(name []byte, hasAttr bool, tok *html.Tokenizer, w io.Writer) error {
	var err error
	tagName := name[len(b.Options.TagPrefix):]

	// Parse tag attrs as vars
	tagVars := make(map[string][]byte)
	if hasAttr {
		for {
			key, val, hasMore := tok.TagAttr()
			valCopy := make([]byte, len(val))
			copy(valCopy, val)
			tagValReader := parseVariableTags(bytes.NewReader(valCopy), b.Vars, b.Options)
			valCopy, err = ioutil.ReadAll(tagValReader)
			tagVars[string(key)] = valCopy
			if !hasMore {
				break
			}
		}
	}

	pathAttr, ok := tagVars[":name"]
	if len(tagName) == 0 && ok {
		tagNameReader := parseVariableTags(bytes.NewReader(pathAttr), b.Vars, b.Options)
		tagName, err = ioutil.ReadAll(tagNameReader)
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
	tag := NewTemplateTag(tagName, attrs, b.Options)
	b.tagStack = append(b.tagStack, tag)
	return tag
}

// Closes the last template tag off and parses the injectedContent
// into the next latest template tag or, if not mor tags exist,
// adds the tag injectedContent to the output
func (b *Builder) closeTemplateTag(writer io.Writer) (err error) {
	var closingTag *templateTag

	cDepth := len(b.tagStack)
	closingTag = b.tagStack[cDepth-1]

	content, err := closingTag.Parse(b)
	if err != nil {
		return err
	}

	if closingTag.path != "" {
		b.FilesParsed[closingTag.path] = true
	}

	if cDepth > 1 {
		prevTag := b.tagStack[cDepth-2]
		prevTag.injectedContent = append(prevTag.injectedContent, content...)
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

		// Read as variable if starting with variable symbol and injectedContent exists
		// Otherwise stop reading variables
		if len(text) > 0 && text[0] == varChar && len(text) > 1 {
			splitVar := bytes.SplitN(text[1:], varSep, 2)
			if len(splitVar) != 2 {
				continue
			}
			key := string(splitVar[0])
			if _, exists := b.Vars[key]; !exists {
				b.Vars[key] = splitVar[1]
			}
		} else {
			break
		}

	}

	// Send the remaining injectedContent back via reader
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

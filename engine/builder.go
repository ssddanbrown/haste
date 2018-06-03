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
}

func NewBuilder(r io.Reader, m *Manager, parent *Builder) *Builder {
	b := &Builder{
		Manager: m,
		Reader:  r,
	}

	// Create var store and copy over parent vars
	b.Vars = make(map[string][]byte)
	if parent != nil {
		for k, v := range parent.Vars {
			b.Vars[k] = v
		}
	}

	return b
}

func (b *Builder) Build() io.Reader {
	r := b.parseTemplateVariables(b.Reader)
	r = b.parseTemplateTags(r)
	return r
	// TODO - Tokenize through
	// TODO - Replace variable tags
	// TODO - Inject content(s)
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

			// if err != nil {
			// 	return returnReader, err
			// }
		}
	}()

	return returnReader
}

func (b *Builder) parseToken(tok *html.Tokenizer, w io.Writer) {
	raw := tok.Raw()
	name, hasAttr := tok.TagName()

	tagPrefix := b.Manager.TagPrefix
	prefixLen := len(tagPrefix)

	isTempTag := len(name) > prefixLen && bytes.Equal(name[0:prefixLen], tagPrefix)

	// Write content if normal tag
	if !isTempTag {
		w.Write(raw)
		return
	}

	// Parse tag attrs as vars
	tagVars := make(map[string][]byte)
	if hasAttr {
		for {
			key, val, hasMore := tok.TagAttr()
			tagVars[string(key)] = val
			if !hasMore {
				break
			}
		}
	}
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

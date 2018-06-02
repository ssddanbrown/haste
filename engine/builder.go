package engine

import (
	"bufio"
	"bytes"
	"io"
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
	return b.parseTemplateVariables(b.Reader)
	// TODO - Tokenize through
	// TODO - Replace variable tags
	// TODO - Inject content(s)
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

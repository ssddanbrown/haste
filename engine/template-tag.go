package engine

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/ssddanbrown/haste/options"
)

type templateTag struct {
	options         *options.Options
	name            []byte
	injectedContent []byte
	contentType     string
	tagType         string
	path            string
	attrs           map[string][]byte
	varContent      map[string][]byte
	topLevel        bool
}

func NewVariableTag(name []byte, opts *options.Options) *templateTag {
	tag := &templateTag{
		name:    make([]byte, len(name)),
		tagType: "variable",
		options: opts,
	}
	copy(tag.name, name)
	return tag
}

func NewTemplateTag(name []byte, attrs map[string][]byte, opts *options.Options, isTopLevel bool) *templateTag {
	tag := &templateTag{
		name:     make([]byte, len(name)),
		attrs:    attrs,
		tagType:  "template",
		options:  opts,
		topLevel: isTopLevel,
	}

	copy(tag.name, name)
	return tag
}

func (t *templateTag) nameToPath(ext string) string {
	strName := string(t.name)
	strName = strings.TrimSuffix(strName, ext)
	p := strings.Replace(strName, ".", "/", -1)
	p = strings.Replace(p, ":", "../", -1) + ext
	return filepath.FromSlash(p)
}

func (t *templateTag) getReader() (io.Reader, error) {
	strName := string(t.name)
	var likelyLocations []string

	extTypes := []string{"css", "js", "html"}
	for _, baseExt := range extTypes {
		ext := "." + baseExt
		if baseExt == "html" || strings.HasSuffix(strName, ext) {
			filePath := t.nameToPath(ext)
			likelyLocations = append(likelyLocations, filePath)
			reader, err := t.options.TemplateResolver.GetTemplateReader(filePath)
			if err == nil {
				t.contentType = baseExt
				t.path = filePath
				return reader, err
			}
		}
	}

	errMsg := fmt.Sprintf("Could not find tag with name \"%s\" at of the following locations:\n%s", t.name, strings.Join(likelyLocations, "\n"))
	return nil, errors.New(errMsg)
}

func (t *templateTag) Parse(parentBuilder *Builder) ([]byte, error) {
	tagReader, err := t.getReader()
	if err != nil {
		return nil, err
	}

	// Generate injectedContent
	tagBuilder := NewBuilder(tagReader, parentBuilder.Options, parentBuilder)

	// Clean and parse inner injectedContent before merging tags
	// Prevents attr vars leaking into scope of the injectedContent
	injectedContent := bytes.Trim(t.injectedContent, "\n\r ")
	injectedContentReader := parseVariableTags(bytes.NewReader(injectedContent), tagBuilder.Vars, parentBuilder.Options, false)
	injectedContent, err = ioutil.ReadAll(injectedContentReader)

	tagBuilder.mergeVars(t.attrs)
	tagBuilder.Vars["content"] = injectedContent

	// Finished rendered result of this tag's contents
	tagSourceContentReader := tagBuilder.Build()

	// Read injectedContent and wrap if style or script
	// TODO - Refactor to stream? If possible here
	tagSourceContent, err := ioutil.ReadAll(tagSourceContentReader)
	if t.contentType == "css" {
		tagSourceContent = append([]byte("<style>\n"), tagSourceContent...)
		tagSourceContent = append(tagSourceContent, []byte("\n</style>")...)
	} else if t.contentType == "js" {
		tagSourceContent = append([]byte("<script>\n"), tagSourceContent...)
		tagSourceContent = append(tagSourceContent, []byte("\n</script>")...)
	}

	return tagSourceContent, err
}

func parseVariableTags(r io.Reader, vars map[string][]byte, opts *options.Options, isTopLevel bool) io.Reader {

	returnReader, w := io.Pipe()

	scanner := bufio.NewScanner(r)
	go func() {

		defer w.Close()

		escChar := byte('@')
		startTag := opts.VarTagOpen
		startTagLen := len(startTag)
		endTag := opts.VarTagClose
		endTagLen := len(endTag)
		escapedTagStart := bytes.Join([][]byte{
			{escChar},
			startTag,
		}, []byte(""))
		escapedTagStartLen := len(escapedTagStart)

		var line []byte
		for scanner.Scan() {

			// Restore newlines that scanner will remove
			if line != nil {
				w.Write([]byte("\n"))
			}

			inTag := false
			tagStart := 0
			tagEnd := -1

			line = scanner.Bytes()
			contentLen := len(line)
			for i := range line {
				atTagStart := contentLen >= i+startTagLen && bytes.Equal(line[i:i+startTagLen], startTag)
				if  atTagStart && (i == 0 || line[i-1] != escChar) {
					// Start tag
					if inTag {
						// Update new contents if that tag start is reset
						w.Write(line[tagStart:i])
					}
					tagStart = i
					inTag = true
				} else if inTag && contentLen >= endTagLen && bytes.Equal(line[i:i+endTagLen], endTag) && (i == 0 || line[i-1] != escChar) {
					// End tag
					inTag = false
					tagKey := string(line[tagStart+startTagLen : i])
					w.Write(vars[tagKey])
					tagEnd = i + endTagLen - 1
				} else if inTag && i-tagStart > 100 {
					// Tag name tracking cutoff
					inTag = false
					tagEnd = -1
					w.Write(line[tagStart:i])
				} else if isTopLevel && contentLen >= i+escapedTagStartLen && bytes.Equal(line[i:i+escapedTagStartLen], escapedTagStart) {
					// Start of escaped tag
					// Simply does not write out the escape char if top level
				} else if !inTag && tagEnd < i {
					// No tag
					w.Write(line[i : i+1])
				}
			}

			// Add any remaining line if a new tag was being tracked
			if inTag {
				w.Write(line[tagStart:])
			}
		}

	}()

	return returnReader
}

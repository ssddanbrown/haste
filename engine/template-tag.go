package engine

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/ssddanbrown/haste/options"
	"io"
	"io/ioutil"
	"strings"
)

type templateTag struct {
	options     *options.Options
	name        []byte
	content     []byte
	contentType string
	tagType     string
	path        string
	attrs       map[string][]byte
	varContent  map[string][]byte
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

func NewTemplateTag(name []byte, attrs map[string][]byte, opts *options.Options) *templateTag {
	tag := &templateTag{
		name:    make([]byte, len(name)),
		attrs:   attrs,
		tagType: "template",
		options: opts,
	}
	copy(tag.name, name)
	return tag
}

func (t *templateTag) nameToPath(ext string) string {
	strName := string(t.name)
	strName = strings.TrimSuffix(strName, ext)
	p := strings.Replace(strName, ".", "/", -1)
	return strings.Replace(p, ":", "../", -1) + ext
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

func (t *templateTag) Parse(b *Builder) ([]byte, error) {
	tagReader, err := t.getReader()
	if err != nil {
		return nil, err
	}

	// Generate content
	tagBuilder := NewBuilder(tagReader, b.Options, b)
	contentReader := tagBuilder.Build()

	// Clean and parse inner content before merging tags
	// Prevents attr vars leaking into scope of the content
	innerContent := bytes.Trim(t.content, "\n\r ")
	innerContentReader := parseVariableTags(bytes.NewReader(innerContent), tagBuilder.Vars, b.Options)
	innerContent, err = ioutil.ReadAll(innerContentReader)

	tagBuilder.mergeVars(t.attrs)

	// Read content and wrap if style or script
	// TODO - Refactor to stream? If possible here
	content, err := ioutil.ReadAll(contentReader)
	if t.contentType == "css" {
		content = append([]byte("<style>\n"), content...)
		content = append(content, []byte("\n</style>")...)
	} else if t.contentType == "js" {
		content = append([]byte("<script>\n"), content...)
		content = append(content, []byte("\n</script>")...)
	}

	// Read content tags
	tagBuilder.Vars["content"] = innerContent
	contentReader = parseVariableTags(bytes.NewReader(content), tagBuilder.Vars, b.Options)
	content, err = ioutil.ReadAll(contentReader)
	return content, err
}

func parseVariableTags(r io.Reader, vars map[string][]byte, opts *options.Options) io.Reader {

	returnReader, w := io.Pipe()

	scanner := bufio.NewScanner(r)
	go func() {

		defer w.Close()

		escChar := byte('@')
		startTag := opts.VarTagOpen
		startTagLen := len(startTag)
		endTag := opts.VarTagClose
		endTagLen := len(endTag)



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
				if contentLen >= i+startTagLen && bytes.Equal(line[i:i+startTagLen], startTag) && (i == 0 || line[i-1] != escChar) {
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
				} else if !inTag && tagEnd < i {
					// No tag
					w.Write(line[i:i+1])
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

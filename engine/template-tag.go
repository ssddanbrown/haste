package engine

import (
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
	innerContent = parseVariableTags(b.Options, innerContent, tagBuilder.Vars)

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
	content = parseVariableTags(b.Options, content, tagBuilder.Vars)
	return content, err
}

func parseVariableTags(opts *options.Options, content []byte, vars map[string][]byte) []byte {
	inTag := false
	tagStart := 0
	tagEnd := -1

	var newContent []byte

	escChar := byte('@')
	startTag := opts.VarTagOpen
	startTagLen := len(startTag)
	endTag := opts.VarTagClose
	endTagLen := len(endTag)

	contentLen := len(content)

	// TODO - Refactor to be piping?

	for i := range content {
		if contentLen >= i+startTagLen && bytes.Equal(content[i:i+startTagLen], startTag) && (i == 0 || content[i-1] != escChar) {
			// Start tag
			if inTag {
				// Update new contents if that tag start is reset
				newContent = append(newContent, content[tagStart:i]...)
			}
			tagStart = i
			inTag = true
		} else if inTag && contentLen >= endTagLen && bytes.Equal(content[i:i+endTagLen], endTag) && (i == 0 || content[i-1] != escChar) {
			// End tag
			inTag = false
			tagKey := string(content[tagStart+startTagLen : i])
			newContent = append(newContent, vars[tagKey]...)
			tagEnd = i + endTagLen - 1
		} else if inTag && i-tagStart > 100 {
			// Tag name tracking cutoff
			inTag = false
			newContent = append(newContent, content[tagStart:i]...)
		} else if !inTag && tagEnd < i {
			// No tag
			newContent = append(newContent, content[i])
		}
	}

	// Add any remaining content if a new tag was being tracked
	if inTag {
		newContent = append(newContent, content[tagStart:]...)
	}

	return newContent
}

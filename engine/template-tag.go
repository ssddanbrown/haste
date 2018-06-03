package engine

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type templateTag struct {
	name []byte

	content     []byte
	contentType string

	attrs map[string][]byte
}

func NewTemplateTag(name []byte, attrs map[string][]byte) *templateTag {
	tag := &templateTag{
		name:  make([]byte, len(name)),
		attrs: attrs,
	}
	copy(tag.name, name)
	return tag
}

func (t *templateTag) nameToPath(ext, root string) string {
	strName := string(t.name)
	strName = strings.TrimSuffix(strName, ext)
	p := strings.Replace(strName, ".", "/", -1)
	p = strings.Replace(p, ":", "../", -1) + ext
	return filepath.Join(root, p)
}

func (t *templateTag) getReader(manager *Manager) (io.Reader, error) {
	tagPath, err := t.findFile(manager)
	if err != nil {
		return nil, err
	}
	return os.Open(tagPath)
}

func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func (t *templateTag) findFile(manager *Manager) (string, error) {
	strName := string(t.name)
	htmlPath := t.nameToPath(".html", manager.WorkingDir)
	likelyLocations := []string{htmlPath}
	if checkFileExists(htmlPath) {
		t.contentType = "html"
		return htmlPath, nil
	}

	altTypes := []string{"css", "js"}
	for i := range altTypes {
		ext := "." + altTypes[i]
		if strings.HasSuffix(strName, ext) {
			filePath := t.nameToPath(ext, manager.WorkingDir)
			likelyLocations = append(likelyLocations, filePath)
			if checkFileExists(filePath) {
				t.contentType = altTypes[i]
				return filePath, nil
			}
		}
	}

	errMsg := fmt.Sprintf("Could not find tag with name \"%s\" at of the following locations:\n%s", t.name, strings.Join(likelyLocations, "\n"))
	return "", errors.New(errMsg)
}

func (t *templateTag) Parse(b *Builder) ([]byte, error) {
	tagReader, err := t.getReader(b.Manager)
	if err != nil {
		return nil, err
	}

	// Generate content
	tagBuilder := NewBuilder(tagReader, b.Manager, b)
	contentReader := tagBuilder.Build()
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
	innerContent := bytes.Trim(t.content, "\n\r ")
	content = bytes.Replace(content, []byte("@content"), innerContent, -1)
	content = parseVariableTags(content, tagBuilder.Vars)
	return content, err
}

func parseVariableTags(content []byte, vars map[string][]byte) []byte {
	inTag := false
	tagStart := 0
	tagEnd := -1

	var newContent []byte

	startTag := []byte("{{{")
	escChar := byte('@')
	startTagLen := len(startTag)
	endTag := []byte("}}}")
	endTagLen := len(endTag)

	contentLen := len(content)

	// TOOD - Refactor to be piping?

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

	// Add any remaning content if a new tag was being tracked
	if inTag {
		newContent = append(newContent, content[tagStart:]...)
	}

	return newContent
}

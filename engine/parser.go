package engine

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func (t *templateTag) nameToPath() string {
	p := strings.Replace(t.name, ".", "/", -1)
	return strings.Replace(p, ":", "../", -1) + ".html"
}

func (t *templateTag) parseTemplate() (string, error) {
	tempFilePath := filepath.Join(t.tracker.contextFolderPath, t.nameToPath())
	var r io.Reader

	if val, ok := t.tracker.templateContent[tempFilePath]; ok {
		r = strings.NewReader(val)
	} else {
		// If template file does not exist, throw an error
		if _, err := os.Stat(tempFilePath); os.IsNotExist(err) {
			return "", errors.New(fmt.Sprintf("Template tag with name \"%s\" does not have a template file at \"%s\"", t.name, tempFilePath))
		}

		tempFile, err := os.Open(tempFilePath)
		defer tempFile.Close()
		if err != nil {
			return "", err
		}
		content, err := ioutil.ReadAll(tempFile)
		if err != nil {
			return "", err
		}

		contentString := string(content)
		t.tracker.templateContent[tempFilePath] = contentString
		r = strings.NewReader(contentString)
	}

	// Parse in the child content
	templateContent, err := parseChild(r, tempFilePath, t.tracker)
	if err != nil {
		return "", err
	}

	// Replace the defined @content section with parsed template tag contents
	c := strings.Replace(templateContent, "@content", t.content, -1)

	return t.tracker.parseVariableTags(c), nil
}

func (t *tracker) parseVariableTags(s string) string {
	inTag := false
	tagStart := 0
	tagEnd := -1

	newContent := make([]byte, 0)
	symbols := []byte("{}@")
	b := []byte(s)
	bLen := len(b)
	for i := range b {
		if b[i] == symbols[0] && bLen > i+2 && b[i+1] == symbols[0] && b[i+2] == symbols[0] && (i == 0 || b[i-1] != symbols[2]) {
			// Start tag
			if inTag {
				newContent = append(newContent, b[tagStart:i]...) // Update new contents if that tag start is reset
			}
			tagStart = i + 1
			inTag = true
		} else if inTag && b[i] == symbols[1] && bLen > i+2 && b[i+1] == symbols[1] && b[i+2] == symbols[1] {
			// End tag
			inTag = false
			tagKey := string(b[tagStart+2 : i])
			newContent = append(newContent, t.vars[tagKey]...)
			tagEnd = i + 2
		} else if inTag && i-tagStart > 100 {
			// Tag name tracking cutoff
			inTag = false
			newContent = append(newContent, b[tagStart:i]...)
		} else if !inTag && tagEnd < i {
			// No tag
			newContent = append(newContent, b[i])
		}
	}
	// Add any remaning content if a new tag was being tracked
	if inTag {
		newContent = append(newContent, b[tagStart:]...)
	}
	return string(newContent)
}

// Perform all pre-parse actions, These modify the HTML before it
// reaches the tozeniker
func (t *tracker) preParseTemplate(r io.Reader) (string, error) {
	text, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	fileContents := ""

	// Search, Store & remove any variables
	readingVars := true
	readingLineVarName := false
	readingLineVarValue := false
	cName := ""
	cVal := ""
	symbols := []byte("@\n=\r")

	for i := range text {
		if readingVars {
			// If start of var
			if text[i] == symbols[0] && (i == 0 || text[i-1] == symbols[1]) {
				readingLineVarName = true
				readingLineVarValue = false
				cName = ""
			} else if readingLineVarName && text[i] != symbols[1] && text[i] != symbols[2] {
				cName += string(text[i])
			} else if readingLineVarName && text[i] == symbols[2] {
				readingLineVarName = false
				readingLineVarValue = true
				cVal = ""
			} else if readingLineVarValue && text[i] != symbols[1] && text[i] != symbols[3] {
				cVal += string(text[i])
			} else if readingLineVarValue {
				t.vars[cName] = cVal
				readingLineVarValue = false
			} else if i == 0 || text[i-1] == symbols[1] {
				readingVars = false
			}
		} else {
			fileContents = string(text[i-1:])
			break
		}
	}
	return fileContents, nil
}

func parseVar(line string) (key string, value string) {
	if line[0] != "@"[0] {
		return
	}
	equalsPos := strings.Index(line, "=")
	if equalsPos == -1 {
		return
	}
	return line[1:equalsPos], line[equalsPos+1:]
}

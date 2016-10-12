package engine

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func (t *templateTag) nameToPath() string {
	return strings.Replace(t.name, ".", "/", -1) + ".html"
}

func (t *templateTag) parseTemplate() (string, error) {
	tempFilePath := filepath.Join(t.tracker.contextFolderPath, t.nameToPath())

	// If template file does not exist, throw an error
	if _, err := os.Stat(tempFilePath); os.IsNotExist(err) {
		return "", errors.New(fmt.Sprintf("Template tag with name \"%s\" does not have a template file at \"%s\"", t.name, tempFilePath))
	}

	tempFile, err := os.Open(tempFilePath)
	defer tempFile.Close()
	if err != nil {
		return "", err
	}

	// Parse in the child content
	templateContent, err := parseChild(tempFile, tempFilePath, t.tracker)
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
	tagEnd := 0
	sLen := len(s)
	newString := ""
	sTag := "{"[0]
	cTag := "}"[0]
	for i := range s {
		if sLen > i+2 && s[i] == sTag && s[i+1] == sTag && s[i+2] == sTag && (i == 0 || s[i-1] != "@"[0]) {
			// Start tag
			if inTag {
				newString += s[tagStart:i] // Update new contents if that tag start is reset
			}
			tagStart = i + 1
			inTag = true
		} else if inTag && sLen > i+2 && s[i] == cTag && s[i+1] == cTag && s[i+2] == cTag {
			// End tag
			inTag = false
			tagKey := s[tagStart+2 : i]
			newString += t.vars[tagKey]
			tagEnd = i + 2
		} else if inTag && i-tagStart > 100 {
			// Tag name tracking sutoff
			inTag = false
			newString += s[tagStart:i]
		} else if !inTag && tagEnd < i {
			// No tag
			newString += string(s[i])
		}
	}
	// Add any remaning content if a new tag was being tracked
	if inTag {
		newString += s[tagStart:]
	}
	return newString
}

// Perform all pre-parse actions, These modify the HTML before it
// reaches the tozeniker
func (t *tracker) preParseTemplate(r io.Reader) (string, error) {

	scanner := bufio.NewScanner(r)
	fileContents := ""

	// Search, Store & remove any variables
	reading := true
	readingVars := true
	line := 0
	for scanner.Scan() && reading {
		text := scanner.Text()
		if readingVars && text[0] == "@"[0] {
			if key, value := parseVar(text); key != "" {
				t.vars[key] = value
			}
		} else {
			readingVars = false
			if line != 0 {
				fileContents += "\n"
			}
			fileContents += text
			line++
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

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

	"golang.org/x/net/html"
)

func (t *templateTag) nameToPath(name, ext string) string {
	p := strings.Replace(name, ".", "/", -1)
	p = strings.Replace(p, ":", "../", -1) + ext
	return filepath.Join(t.tracker.contextFolderPath, p)
}

func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func (t *templateTag) searchForPossibleFiles() (string, error) {
	htmlPath := t.nameToPath(t.name, ".html")
	likelyLocations := []string{htmlPath}
	if checkFileExists(htmlPath) {
		t.contentType = "html"
		return htmlPath, nil
	}

	altTypes := []string{"css", "js"}
	for i := range altTypes {
		ext := "." + altTypes[i]
		if strings.HasSuffix(t.name, ext) {
			filePath := t.nameToPath(strings.TrimSuffix(t.name, ext), ext)
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

func (t *templateTag) parseTemplate(attrs []*html.Attribute) (string, error) {

	fileLocation, err := t.searchForPossibleFiles()
	if err != nil {
		return "", err
	}

	var r io.Reader

	if val, ok := t.tracker.templateContent[fileLocation]; ok {
		r = strings.NewReader(val)
	} else {

		tempFile, fileErr := os.Open(fileLocation)
		defer tempFile.Close()
		if fileErr != nil {
			return "", fileErr
		}
		content, fileErr := ioutil.ReadAll(tempFile)
		if fileErr != nil {
			return "", fileErr
		}

		contentString := string(content)
		t.tracker.templateContent[fileLocation] = contentString
		r = strings.NewReader(contentString)
	}

	// Parse in the child content
	templateContent, err := parseChild(r, fileLocation, t.tracker.contextFolderPath, t.tracker, attrs)
	if err != nil {
		return "", err
	}

	// Replace the defined @content section with parsed template tag contents
	tContent := strings.Replace(templateContent, "@content", t.content, -1)
	tContent = t.tracker.parseVariableTags(tContent)

	if t.contentType == "css" {
		tContent = "<style>\n" + tContent + "\n</style>"
	} else if t.contentType == "js" {
		tContent = "<script>\n" + tContent + "\n</script>"
	}

	return tContent, nil
}

func (t *tracker) parseVariableTags(s string) string {
	inTag := false
	tagStart := 0
	tagEnd := -1

	var newContent []byte

	startTag := []byte("{{{")
	escChar := byte('@')
	startTagLen := len(startTag)
	endTag := []byte("}}}")
	endTagLen := len(endTag)

	content := []byte(s)
	contentLen := len(content)

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
			newContent = append(newContent, t.vars[tagKey]...)
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

	varChar := byte('@')
	varSep := byte('=')
	newLine := byte('\n')
	lineReturn := byte('\r')

	for i, cChar := range text {
		if readingVars {
			// If start of var
			if cChar == varChar && (i == 0 || text[i-1] == newLine) {
				readingLineVarName = true
				readingLineVarValue = false
				cName = ""
			} else if readingLineVarName && cChar != newLine && cChar != varSep {
				cName += string(cChar)
			} else if readingLineVarName && cChar == varSep {
				readingLineVarName = false
				readingLineVarValue = true
				cVal = ""
			} else if readingLineVarValue && cChar != newLine && cChar != lineReturn {
				cVal += string(cChar)
			} else if readingLineVarValue {
				t.vars[cName] = cVal
				readingLineVarValue = false
			} else if i == 0 || text[i-1] == newLine {
				readingVars = false
			}
		} else {
			fileContents = string(text[i-1:])
			break
		}
	}
	return fileContents, nil
}

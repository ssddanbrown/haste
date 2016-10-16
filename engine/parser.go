package engine

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func (t *templateTag) nameToPath() string {
	return strings.Replace(t.name, ".", "/", -1) + ".html"
}

func (t *templateTag) parseTemplate() (io.Reader, error) {
	tempFilePath := filepath.Join(t.tracker.contextFolderPath, t.nameToPath())

	var r io.Reader

	if val, ok := t.tracker.templateContent[tempFilePath]; ok {
		r = strings.NewReader(val)
	} else {
		// If template file does not exist, throw an error
		if _, err := os.Stat(tempFilePath); os.IsNotExist(err) {
			return nil, errors.New(fmt.Sprintf("Template tag with name \"%s\" does not have a template file at \"%s\"", t.name, tempFilePath))
		}

		tempFile, err := os.Open(tempFilePath)
		defer tempFile.Close()
		if err != nil {
			return nil, err
		}
		content, err := ioutil.ReadAll(tempFile)
		if err != nil {
			return nil, err
		}

		contentString := string(content)
		t.tracker.templateContent[tempFilePath] = contentString
		r = strings.NewReader(contentString)
	}

	// Parse in the child content
	templateContent, _ := parseChild(r, tempFilePath, t.tracker)

	// Replace the defined @content section with parsed template tag contents
	bufr := bufio.NewReader(templateContent)
	newr, neww := io.Pipe()
	search := []byte("@content")
	go func() {
		for {
			line, err := bufr.ReadBytes('\n')
			if err != nil && err != io.EOF {
				t.tracker.errChan <- err
			}
			content, err := ioutil.ReadAll(t.reader)
			if err != nil {
				t.tracker.errChan <- err
			}
			line = bytes.Replace(line, search, content, -1)
			neww.Write(line)
			neww.Write([]byte{'\n'})
		}
	}()

	parseReader, _ := t.tracker.parseVariableTags(newr)
	return parseReader, nil
}

func (t *tracker) parseVariableTags(r io.Reader) (io.Reader, chan bool) {

	returnReader, writer := io.Pipe()
	buf := bufio.NewWriter(writer)
	bufr := bufio.NewReader(r)
	rChan := make(chan bool)

	go func() {
		for {
			b, err := bufr.ReadBytes('\n')
			fmt.Println(string(b))
			if err != nil && err != io.EOF {
				t.errChan <- err
				rChan <- false
				break
			} else if err == io.EOF {
				rChan <- true
				buf.Write(b)
				buf.Flush()
				break
			}

			inTag := false
			tagStart := 0
			tagEnd := -1

			bLen := len(b)
			for i := range b {
				if b[i] == '{' && bLen > i+2 && b[i+1] == '{' && b[i+2] == '{' && (i == 0 || b[i-1] != '@') {
					// Start tag
					if inTag {
						buf.Write(b[tagStart:i]) // Update new contents if that tag start is reset
					}
					tagStart = i + 1
					inTag = true
				} else if inTag && b[i] == '}' && bLen > i+2 && b[i+1] == '}' && b[i+2] == '}' {
					// End tag
					inTag = false
					tagKey := string(b[tagStart+2 : i])
					buf.Write([]byte(t.vars[tagKey]))
					tagEnd = i + 2
				} else if inTag && i-tagStart > 100 {
					// Tag name tracking cutoff
					inTag = false
					buf.Write(b[tagStart:i])
				} else if !inTag && tagEnd < i {
					// No tag
					buf.Write([]byte{b[i]})
				}
			}
			// Add any remaning content if a new tag was being tracked
			if inTag {
				buf.Write(b[tagStart:])
			}
		}
	}()

	return returnReader, rChan
}

// Perform all pre-parse actions, These modify the HTML before it
// reaches the tozeniker
func (t *tracker) preParseTemplate(r io.Reader) (io.Reader, chan bool) {

	output, writer := io.Pipe()
	buf := bufio.NewWriter(writer)
	rChan := make(chan bool)

	go func() {
		bufr := bufio.NewReader(r)

		// Search, Store & remove any variables
		readingVars := true
		readingLineVarName := false
		readingLineVarValue := false
		cName := []byte{}
		cVal := []byte{}

		for {
			text, err := bufr.ReadBytes('\n')

			if err != nil && err != io.EOF {
				t.errChan <- err
				rChan <- false
				break
			} else if err == io.EOF {
				buf.Write(text)
				buf.Flush()
				rChan <- true
				break
			}

			if readingVars {
				textLen := len(text)
				for i := range text {

					if !readingVars {
						buf.WriteByte(text[i])
						continue
					}

					if i == 0 {
						readingLineVarName = false
						readingLineVarValue = false
					}

					if text[i] == '@' && i == 0 {
						readingLineVarName = true
						readingLineVarValue = false
						cName = []byte{}
					} else if readingLineVarName && text[i] != '=' {
						cName = append(cName, text[i])
					} else if readingLineVarName && text[i] == '=' {
						readingLineVarName = false
						readingLineVarValue = true
						cVal = []byte{}
					} else if readingLineVarValue && text[i] != '\r' && i != textLen-1 {
						cVal = append(cVal, text[i])
					} else if readingLineVarValue {
						t.vars[string(cName)] = string(cVal)
						readingLineVarValue = false
					} else if i == 0 {
						readingVars = false
						buf.WriteByte(text[i])
						continue
					}
				}
			} else {
				buf.Write(append(text, '\n'))
			}
		}
	}()

	return output, rChan
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

// func stringScanSplitter(search string) func() (advance int, token []byte, err error) {
// 	search = []byte(search)
// 	width := len(search)
// 	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
// 		lastestSuceess := -1
// 		dataLen := len(data)

// 		for i := 0; i < width+1; i++ {
// 			index := bytes.Index(data, search[0:i])
// 			if index != -1 {
// 				lastestSuceess = i

// 				if i == width {
// 					return index + width, data[index : index+width], nil
// 				}
// 			} else if lastestSuceess > -1 && i == dataLen {
// 				return lastestSuceess, nil, nil
// 			} else if lastestSuceess > -1 {
// 				lastestSuceess = -1
// 			}
// 		}

// 		return dataLen, nil, nil
// 	}
// }

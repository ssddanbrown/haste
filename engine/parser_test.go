package engine

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

var _ = fmt.Println
var _ = ioutil.ReadAll

var (
	testTracker *tracker
)

func trackerSetup() {
	testTracker = newTracker(strings.NewReader(""), "", nil)
}

func TestVariableParsing(t *testing.T) {
	trackerSetup()
	input := `@test=hello
@anotherItem=this is some content
@third-item="more" content to parse
<html>html code</html>
@notParsed=this should not be a variable
	`
	expected := make(map[string]string)
	expected["test"] = "hello"
	expected["anotherItem"] = "this is some content"
	expected["third-item"] = `"more" content to parse`

	reader, rchan := testTracker.preParseTemplate(strings.NewReader(input))
	go func() {
		buf := bufio.NewReader(reader)
		for {
			_, _ = buf.ReadBytes('\n')
		}
	}()

	for {
		_ = <-rchan
		break
	}

	for k, v := range expected {
		if val, ok := testTracker.vars[k]; !ok || val != v {
			t.Error("Parsed variables did not parse as expected")
		}
	}

	if _, ok := testTracker.vars["notParsed"]; ok {
		t.Error("Variable parsed should not have parsed vars defined after content but it did.")
	}

}

func TestVariableInjection(t *testing.T) {
	trackerSetup()
	input := `@test=hello
@anotherItem=this is some content
<html><div>{{{test}}}{{{anotherItem}}}</div></html>`
	expected := `<html><div>hellothis is some content</div></html>`
	r, riChan := testTracker.preParseTemplate(strings.NewReader(input))

	go func() {
		reader, rchan = testTracker.parseVariableTags(r)
	}()
	outputString := ""

	go func() {
		buf := bufio.NewReader(reader)
		for {
			text, _ := buf.ReadString('\n')
			outputString += text
		}
	}()

	for {
		_ = <-riChan
		_ = <-rchan
		break
	}

	if outputString != expected {
		t.Error(fmt.Sprintf("Variables not injecting as expected. \nExpected: %s\nRecieved: %s", expected, outputString))
	}

}

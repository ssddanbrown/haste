package engine

import (
	"fmt"
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

	reader := testTracker.preParseTemplate(strings.NewReader(input))
	_, _ = ioutil.ReadAll(reader)

	for k, v := range expected {
		if val, ok := testTracker.vars[k]; !ok || val != v {
			t.Error("Parsed variables did not parse as expected")
		}
	}

	if _, ok := testTracker.vars["notParsed"]; ok {
		t.Error("Variable parsed should not have parsed vars defined after content but it did.")
	}

}

// func TestVariableInjection(t *testing.T) {
// 	trackerSetup()
// 	input := `@test=hello
// @anotherItem=this is some content
// <html><div>{{{test}}}{{{anotherItem}}}</div></html>`
// 	expected := `<html><div>hellothis is some content</div></html>`
// 	s := testTracker.preParseTemplate(strings.NewReader(input))

// 	output := testTracker.parseVariableTags(s)

// 	outputBytes, _ := ioutil.ReadAll(output)
// 	if string(outputBytes) != expected {
// 		t.Error(fmt.Sprintf("Variables not injecting as expected. \nExpected: %s\nRecieved: %s", expected, output))
// 	}

// }

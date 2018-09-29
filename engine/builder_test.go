package engine

import (
	"fmt"
	"github.com/ssddanbrown/haste/loading"
	"github.com/ssddanbrown/haste/options"
	"io/ioutil"
	"strings"
	"testing"
)

func simpleBuild(t *testing.T, input string, resolveMap map[string]string) string {
	opts := options.NewOptions()
	resolver := loading.NewTestResolver(resolveMap)
	opts.TemplateResolver = resolver

	builder := NewBuilder(strings.NewReader(input), opts, nil)
	resultReader := builder.Build()
	result, err := ioutil.ReadAll(resultReader)

	if  err != nil {
		t.Fatalf("Recieved error when reading build result; err: %s", err)
	}

	return string(result)
}

func buildResultErrorMessage(expected, received string) string {
	return fmt.Sprintf("Expected result: \n%s \n\nRecieved:\n%s", expected, received)
}

func TestBasicBuildFunctions(t *testing.T) {
	input := strings.TrimSpace(`
<html>
<body><h1>Hello</h1></body>
</html>
`)

	result := simpleBuild(t, input, nil)
	if result != input {
		t.Errorf(buildResultErrorMessage(input, result))
	}
}

func TestSameFileVariableParsing(t *testing.T) {
	input := strings.TrimSpace(`
@a=Some test content
@Cat=hello
@dog1=tree
<html><body>
{{a}}
{{Cat}}{{dog1}}
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
Some test content
hellotree
</body></html>
`)

	received := simpleBuild(t, input, nil)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}
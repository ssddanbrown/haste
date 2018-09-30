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

func TestBasicTemplateTagUsage(t *testing.T) {
	input := strings.TrimSpace(`
<html><body>
<t:hello/><t:hello/>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<p>Hello</p><p>Hello</p>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "<p>Hello</p>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestTemplateTagContentPassing(t *testing.T) {
	input := strings.TrimSpace(`
<html><body>
<t:hello>This is some content</t:hello>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<p>Hello! This is some content</p>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "<p>Hello! {{content}}</p>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestRootVariablesPassDownToChildTemplates(t *testing.T) {
	input := strings.TrimSpace(`
@tree=World!
<html><body>
<t:hello/>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<p>Hello! World!</p>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "<p>Hello! {{tree}}</p>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestAttributesPassDownToChildTemplates(t *testing.T) {
	input := strings.TrimSpace(`
<html><body>
<t:hello tree="World!"/>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<p>Hello! World!</p>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "<p>Hello! {{tree}}</p>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestAttributesOverrideParentGlobalVars(t *testing.T) {
	input := strings.TrimSpace(`
@tree=Cat!
<html><body>
<t:hello tree="World!"/>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<p>Hello! World!</p>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "<p>Hello! {{tree}}</p>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestAttributesOverrideChildGlobalVars(t *testing.T) {
	input := strings.TrimSpace(`
<html><body>
<t:hello tree="World!"/>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<p>Hello! World!</p>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "@tree=Cat!\n<p>Hello! {{tree}}</p>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestBasicVariableTagUsage(t *testing.T) {
	input := strings.TrimSpace(`
<html><body>
<t:hello><v:tree>World!</v:tree></t:hello>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<p>Hello! World!</p>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "<p>Hello! {{tree}}</p>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestVariableTagsDontPolluteParentTagTable(t *testing.T) {
	input := strings.TrimSpace(`
@tree=Cat
<html><body>
<t:hello><v:tree>World!</v:tree></t:hello>
{{tree}}
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<p>Hello! World!</p>
Cat
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "<p>Hello! {{tree}}</p>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestCSSTagUsage(t *testing.T) {
	input := strings.TrimSpace(`
<html><body>
<t:hello.css/>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<style>
.body{background:red;}
</style>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.css": ".body{background:red;}",
		"hello.html": "wrong file",
		"hello.js": "wrong file",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestJSTagUsage(t *testing.T) {
	input := strings.TrimSpace(`
<html><body>
<t:hello.js/>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<script>
var a = 'hello';
</script>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.css": "wrong file",
		"hello.html": "wrong file",
		"hello.js": "var a = 'hello';",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestNestedTemplateTagUsage(t *testing.T) {
	input := strings.TrimSpace(`
<html><body>
<t:hello><t:hello><t:hello>beans!</t:hello></t:hello></t:hello>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<div>Hello <div>Hello <div>Hello beans!</div></div></div>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "<div>Hello {{content}}</div>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestMultiCaseAttributesAreLowerCasedWhenPassed(t *testing.T) {
	input := strings.TrimSpace(`
<html><body>
<t:hello aCoolCat="meow"/>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<div>Hello meow</div>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "<div>Hello {{acoolcat}}</div>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}

func TestMultiCaseAttributesCanByUsedWithinTemplateContent(t *testing.T) {
	input := strings.TrimSpace(`
<html><body>
<t:hello aCoolCat="meow"/>
</body></html>
`)

	expected := strings.TrimSpace(`
<html><body>
<div aCoolCat="meow">Hello</div>
</body></html>
`)

	resolveMap := map[string]string {
		"hello.html": "<div aCoolCat=\"{{acoolcat}}\">Hello</div>",
	}

	received := simpleBuild(t, input, resolveMap)
	if received != expected {
		t.Errorf(buildResultErrorMessage(expected, received))
	}
}
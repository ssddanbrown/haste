package loading

import (
	"io"
	"strings"
)

// TestResolver searches for templates in a given map for fast testing
type TestResolver struct {
	contentMap map[string]string
}

func NewTestResolver(contentMap map[string]string) *TestResolver {
	return &TestResolver{
		contentMap: contentMap,
	}
}

func (t *TestResolver) GetTemplateReader(path string) (io.Reader, error) {
	content, ok := t.contentMap[path]
	if !ok {
		return nil, ErrTemplateNotFound
	}

	return strings.NewReader(content), nil
}


package loading

import (
	"errors"
	"io"
)

var (
	ErrTemplateNotFound = errors.New("Template not found")
)

type TemplateResolver interface {
	GetTemplateReader(path string) (io.Reader, error)
}


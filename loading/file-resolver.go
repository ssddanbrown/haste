package loading

import (
	"io"
	"os"
	"path/filepath"
)

type FileTemplateResolver struct {
	rootPath string
}

func NewFileTemplateResolver(rootPath string) *FileTemplateResolver {
	return &FileTemplateResolver{rootPath}
}

func (f *FileTemplateResolver) GetTemplateReader(path string) (io.Reader, error) {
	absPath := filepath.Join(f.rootPath, path)
	_, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return nil, ErrTemplateNotFound
	}

	return os.Open(absPath)
}

package engine

type BuildFile struct {
	Path string
}

func NewBuildFile(path string) *BuildFile {
	return &BuildFile{Path: path}
}

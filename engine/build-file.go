package engine

type BuildFile struct {
	path string
	includes []string
}

func NewBuildFile(path string) *BuildFile {
	return &BuildFile{path: path}
}

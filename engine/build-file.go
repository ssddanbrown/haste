package engine

type BuildFile struct {
	path     string
	includes map[string]bool
}

func NewBuildFile(path string) *BuildFile {
	return &BuildFile{
		path:     path,
		includes: make(map[string]bool),
	}
}

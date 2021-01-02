package chezmoi

import (
	"path"
	"strings"
)

// A SourceStatePath is a relative path to an entry in the source state.
type SourceStatePath struct {
	path  string
	isDir bool
}

// NewSourceStateDirPath returns a new SourceStatePath for a directory.
func NewSourceStateDirPath(path string) SourceStatePath {
	return SourceStatePath{
		path:  path,
		isDir: true,
	}
}

// NewSourceStatePath returns a new SourceStatePath.
func NewSourceStatePath(path string) SourceStatePath {
	return SourceStatePath{
		path: path,
	}
}

// Dir returns p's directory.
func (p SourceStatePath) Dir() SourceStatePath {
	return SourceStatePath{
		path:  path.Dir(string(p.path)),
		isDir: true,
	}
}

// RelPath returns p's relative path.
func (p SourceStatePath) RelPath() RelPath {
	sourceNames := strings.Split(p.path, "/")
	relPathNames := make([]string, 0, len(sourceNames))
	if p.isDir {
		for _, sourceName := range sourceNames {
			dirAttr := parseDirAttr(sourceName)
			relPathNames = append(relPathNames, dirAttr.Name)
		}
	} else {
		for _, sourceName := range sourceNames[:len(sourceNames)-1] {
			dirAttr := parseDirAttr(sourceName)
			relPathNames = append(relPathNames, dirAttr.Name)
		}
		fileAttr := parseFileAttr(sourceNames[len(sourceNames)-1])
		relPathNames = append(relPathNames, fileAttr.Name)
	}
	return RelPath(strings.Join(relPathNames, "/"))
}

func (p SourceStatePath) String() string {
	return p.path
}

package chezmoi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	// DefaultTemplateOptions are the default template options.
	DefaultTemplateOptions = []string{"missingkey=error"}

	// EntryStateBucket is the bucket for recording the entry states.
	EntryStateBucket = []byte("entryState")

	// ScriptStateBucket is the bucket for recording the state of run once
	// scripts.
	ScriptStateBucket = []byte("scriptState")

	// Skip indicates that entry should be skipped.
	Skip = filepath.SkipDir
)

// Suffixes and prefixes.
const (
	ignorePrefix     = "."
	dotPrefix        = "dot_"
	emptyPrefix      = "empty_"
	encryptedPrefix  = "encrypted_"
	exactPrefix      = "exact_"
	executablePrefix = "executable_"
	existsPrefix     = "exists_"
	firstPrefix      = "first_"
	lastPrefix       = "last_"
	oncePrefix       = "once_"
	privatePrefix    = "private_"
	runPrefix        = "run_"
	symlinkPrefix    = "symlink_"
	TemplateSuffix   = ".tmpl"
)

// Special file names.
const (
	Prefix = ".chezmoi"

	dataName         = Prefix + "data"
	ignoreName       = Prefix + "ignore"
	removeName       = Prefix + "remove"
	templatesDirName = Prefix + "templates"
	versionName      = Prefix + "version"
)

var knownPrefixedFiles = map[string]bool{
	Prefix + ".json" + TemplateSuffix: true,
	Prefix + ".toml" + TemplateSuffix: true,
	Prefix + ".yaml" + TemplateSuffix: true,
	dataName:                          true,
	ignoreName:                        true,
	removeName:                        true,
	versionName:                       true,
}

var modeTypeNames = map[os.FileMode]string{
	0:                 "file",
	os.ModeDir:        "dir",
	os.ModeSymlink:    "symlink",
	os.ModeNamedPipe:  "named pipe",
	os.ModeSocket:     "socket",
	os.ModeDevice:     "device",
	os.ModeCharDevice: "char device",
}

type errDuplicateTarget struct {
	targetRelPath  RelPath
	sourceRelPaths SourceRelPaths
}

func (e *errDuplicateTarget) Error() string {
	sourceRelPathStrs := make([]string, 0, len(e.sourceRelPaths))
	for _, sourceRelPath := range e.sourceRelPaths {
		sourceRelPathStrs = append(sourceRelPathStrs, sourceRelPath.String())
	}
	return fmt.Sprintf("%s: duplicate target (%s)", e.targetRelPath, strings.Join(sourceRelPathStrs, ", "))
}

type errNotInAbsDir struct {
	pathAbsPath AbsPath
	dirAbsPath  AbsPath
}

func (e *errNotInAbsDir) Error() string {
	return fmt.Sprintf("%s: not in %s", e.pathAbsPath, e.dirAbsPath)
}

type errNotInRelDir struct {
	pathRelPath RelPath
	dirRelPath  RelPath
}

func (e *errNotInRelDir) Error() string {
	return fmt.Sprintf("%s: not in %s", e.pathRelPath, e.dirRelPath)
}

type errUnsupportedFileType struct {
	absPath AbsPath
	mode    os.FileMode
}

func (e *errUnsupportedFileType) Error() string {
	return fmt.Sprintf("%s: unsupported file type %s", e.absPath, modeTypeName(e.mode))
}

// An AbsPath is an absolute path.
type AbsPath string

// Dir returns p's directory.
func (p AbsPath) Dir() AbsPath {
	return AbsPath(path.Dir(string(p)))
}

// Join appends elems to p.
func (p AbsPath) Join(elems ...RelPath) AbsPath {
	elemStrs := make([]string, 0, len(elems)+1)
	elemStrs = append(elemStrs, string(p))
	for _, elem := range elems {
		elemStrs = append(elemStrs, string(elem))
	}
	return AbsPath(path.Join(elemStrs...))
}

// MustTrimDirPrefix is like TrimPrefix but panics on any error.
func (p AbsPath) MustTrimDirPrefix(dirPrefix AbsPath) RelPath {
	relPath, err := p.TrimDirPrefix(dirPrefix)
	if err != nil {
		panic(err)
	}
	return relPath
}

// Split returns p's directory and file.
func (p AbsPath) Split() (AbsPath, RelPath) {
	dir, file := path.Split(string(p))
	return AbsPath(dir), RelPath(file)
}

// TrimDirPrefix trims prefix from p.
func (p AbsPath) TrimDirPrefix(dirPrefixAbsPath AbsPath) (RelPath, error) {
	if !strings.HasPrefix(string(p), string(dirPrefixAbsPath+"/")) {
		return "", &errNotInAbsDir{
			pathAbsPath: p,
			dirAbsPath:  dirPrefixAbsPath,
		}
	}
	return RelPath(p[len(dirPrefixAbsPath)+1:]), nil
}

// AbsPaths is a slice of AbsPaths that implements sort.Interface.
type AbsPaths []AbsPath

func (ps AbsPaths) Len() int           { return len(ps) }
func (ps AbsPaths) Less(i, j int) bool { return string(ps[i]) < string(ps[j]) }
func (ps AbsPaths) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }

// A RelPath is a relative path.
type RelPath string

// Base returns p's base name.
func (p RelPath) Base() string {
	return path.Base(string(p))
}

// Dir returns p's directory.
func (p RelPath) Dir() RelPath {
	return RelPath(path.Dir(string(p)))
}

// Join appends elems to p.
func (p RelPath) Join(elems ...RelPath) RelPath {
	elemStrs := make([]string, 0, len(elems)+1)
	elemStrs = append(elemStrs, string(p))
	for _, elem := range elems {
		elemStrs = append(elemStrs, string(elem))
	}
	return RelPath(path.Join(elemStrs...))
}

// Split returns p's directory and path.
func (p RelPath) Split() (RelPath, RelPath) {
	dir, file := path.Split(string(p))
	return RelPath(dir), RelPath(file)
}

// TrimDirPrefix trims prefix from p.
func (p RelPath) TrimDirPrefix(dirPrefix RelPath) (RelPath, error) {
	if !strings.HasPrefix(string(p), string(dirPrefix+"/")) {
		return "", &errNotInRelDir{
			pathRelPath: p,
			dirRelPath:  dirPrefix,
		}
	}
	return RelPath(p[len(dirPrefix)+1:]), nil
}

// RelPaths is a slice of RelPaths that implements sort.Interface.
type RelPaths []RelPath

func (ps RelPaths) Len() int           { return len(ps) }
func (ps RelPaths) Less(i, j int) bool { return string(ps[i]) < string(ps[j]) }
func (ps RelPaths) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }

// StateData returns the state data in bucket in s.
func StateData(s PersistentState, bucket []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if err := s.ForEach(bucket, func(k, v []byte) error {
		var value map[string]interface{}
		if err := json.Unmarshal(v, &value); err != nil {
			return err
		}
		result[string(k)] = value
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// SuspiciousSourceDirEntry returns true if base is a suspicous dir entry.
func SuspiciousSourceDirEntry(base string, info os.FileInfo) bool {
	//nolint:exhaustive
	switch info.Mode() & os.ModeType {
	case 0:
		return strings.HasPrefix(base, Prefix) && !knownPrefixedFiles[base]
	case os.ModeDir:
		return strings.HasPrefix(base, Prefix) && base != templatesDirName
	case os.ModeSymlink:
		return strings.HasPrefix(base, Prefix)
	default:
		return true
	}
}

// isEmpty returns true if data is empty after trimming whitespace from both
// ends.
func isEmpty(data []byte) bool {
	return len(bytes.TrimSpace(data)) == 0
}

func modeTypeName(mode os.FileMode) string {
	if name, ok := modeTypeNames[mode&os.ModeType]; ok {
		return name
	}
	return fmt.Sprintf("0o%o: unknown type", mode&os.ModeType)
}

// mustTrimPrefix is like strings.TrimPrefix but panics if s is not prefixed by
// prefix.
func mustTrimPrefix(s, prefix string) string {
	if !strings.HasPrefix(s, prefix) {
		panic(fmt.Sprintf("%s: not prefixed by %s", s, prefix))
	}
	return s[len(prefix):]
}

// mustTrimSuffix is like strings.TrimSuffix but panics if s is not suffixed by
// suffix.
func mustTrimSuffix(s, suffix string) string {
	if !strings.HasSuffix(s, suffix) {
		panic(fmt.Sprintf("%s: not suffixed by %s", s, suffix))
	}
	return s[:len(s)-len(suffix)]
}

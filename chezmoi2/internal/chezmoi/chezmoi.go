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

type duplicateTargetError struct {
	targetName  string
	sourcePaths []string
}

func (e *duplicateTargetError) Error() string {
	return fmt.Sprintf("%s: duplicate target (%s)", e.targetName, strings.Join(e.sourcePaths, ", "))
}

type notInDirError struct {
	path string
	dir  string
}

func (e *notInDirError) Error() string {
	return fmt.Sprintf("%s: not in %s", e.path, e.dir)
}

type unsupportedFileTypeError struct {
	path string
	mode os.FileMode
}

func (e *unsupportedFileTypeError) Error() string {
	return fmt.Sprintf("%s: unsupported file type %s", e.path, modeTypeName(e.mode))
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

// MustTrimPrefix is like TrimPrefix but panics on any error.
func (p AbsPath) MustTrimPrefix(prefix AbsPath) RelPath {
	relPath, err := p.TrimPrefix(prefix)
	if err != nil {
		panic(err)
	}
	return relPath
}

func (p AbsPath) String() string { return string(p) }

// TrimPrefix trims prefix from p.
func (p AbsPath) TrimPrefix(prefix AbsPath) (RelPath, error) {
	if !strings.HasPrefix(string(p), string(prefix+"/")) {
		return "", &notInDirError{
			path: string(p),
			dir:  string(prefix),
		}
	}
	return RelPath(p[len(prefix)+1:]), nil
}

type absPathsByName []AbsPath

func (a absPathsByName) Len() int           { return len(a) }
func (a absPathsByName) Less(i, j int) bool { return string(a[i]) < string(a[j]) }
func (a absPathsByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// A RelPath is a relative path.
type RelPath string

// Dir returns p's directory.
func (p RelPath) Dir() RelPath {
	return RelPath(path.Dir(string(p)))
}

func (p RelPath) String() string { return string(p) }

// TrimPrefix trims prefix from p.
func (p RelPath) TrimPrefix(prefix RelPath) (RelPath, error) {
	if !strings.HasPrefix(string(p), string(prefix+"/")) {
		return "", &notInDirError{
			path: string(p),
			dir:  string(prefix),
		}
	}
	return RelPath(p[len(prefix)+1:]), nil
}

// MustTrimDirPrefix is like TrimDirPrefix but panics on any error.
// FIXME remove this function
func MustTrimDirPrefix(pathStr string, dir AbsPath) AbsPath {
	path, err := NewAbsPath(pathStr)
	if err != nil {
		panic(err)
	}
	result, err := TrimDirPrefix(path, dir)
	if err != nil {
		panic(err)
	}
	return result
}

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

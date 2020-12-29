package chezmoi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/coreos/go-semver/semver"
	vfs "github.com/twpayne/go-vfs"
	"go.uber.org/multierr"
)

// An Lstater implements Lstat.
type Lstater interface {
	Lstat(name string) (os.FileInfo, error)
}

// A SourceState is a source state.
type SourceState struct {
	entries              map[string]SourceStateEntry
	system               System
	sourceDir            string
	destDir              string
	umask                os.FileMode
	encryptionTool       EncryptionTool
	ignore               *patternSet
	minVersion           semver.Version
	priorityTemplateData map[string]interface{}
	templateData         map[string]interface{}
	templateFuncs        template.FuncMap
	templateOptions      []string
	templates            map[string]*template.Template
}

// A SourceStateOption sets an option on a source state.
type SourceStateOption func(*SourceState)

// WithDestDir sets the destination directory.
func WithDestDir(destDir string) SourceStateOption {
	return func(s *SourceState) {
		s.destDir = destDir
	}
}

// WithEncryptionTool sets the encryption tool.
func WithEncryptionTool(encryptionTool EncryptionTool) SourceStateOption {
	return func(s *SourceState) {
		s.encryptionTool = encryptionTool
	}
}

// WithPriorityTemplateData adds priority template data.
func WithPriorityTemplateData(priorityTemplateData map[string]interface{}) SourceStateOption {
	return func(s *SourceState) {
		recursiveMerge(s.priorityTemplateData, priorityTemplateData)
		recursiveMerge(s.templateData, s.priorityTemplateData)
	}
}

// WithSourceDir sets the source directory.
func WithSourceDir(sourceDir string) SourceStateOption {
	return func(s *SourceState) {
		s.sourceDir = sourceDir
	}
}

// WithSystem sets the system.
func WithSystem(system System) SourceStateOption {
	return func(s *SourceState) {
		s.system = system
	}
}

// WithTemplateData adds template data.
func WithTemplateData(templateData map[string]interface{}) SourceStateOption {
	return func(s *SourceState) {
		recursiveMerge(s.templateData, templateData)
		recursiveMerge(s.templateData, s.priorityTemplateData)
	}
}

// WithTemplateFuncs sets the template functions.
func WithTemplateFuncs(templateFuncs template.FuncMap) SourceStateOption {
	return func(s *SourceState) {
		s.templateFuncs = templateFuncs
	}
}

// WithTemplateOptions sets the template options.
func WithTemplateOptions(templateOptions []string) SourceStateOption {
	return func(s *SourceState) {
		s.templateOptions = templateOptions
	}
}

// WithUmask sets the umask.
func WithUmask(umask os.FileMode) SourceStateOption {
	return func(s *SourceState) {
		s.umask = umask
	}
}

// NewSourceState creates a new source state with the given options.
func NewSourceState(options ...SourceStateOption) *SourceState {
	s := &SourceState{
		entries:              make(map[string]SourceStateEntry),
		umask:                GetUmask(),
		encryptionTool:       &nullEncryptionTool{},
		ignore:               newPatternSet(),
		priorityTemplateData: make(map[string]interface{}),
		templateData:         make(map[string]interface{}),
		templateOptions:      DefaultTemplateOptions,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// AddOptions are options to SourceState.Add.
type AddOptions struct {
	AutoTemplate bool
	Empty        bool
	Encrypt      bool
	Exact        bool
	Include      *IncludeSet
	Template     bool
	umask        os.FileMode
}

// Add adds destPathInfos to s.
func (s *SourceState) Add(sourceSystem System, persistentState PersistentState, destPathInfos map[string]os.FileInfo, options *AddOptions) error {
	type update struct {
		destPath              string
		entryState            *EntryState
		sourceStateEntryNames []string
	}

	destPaths := make([]string, 0, len(destPathInfos))
	for destPath := range destPathInfos {
		destPaths = append(destPaths, destPath)
	}
	sort.Strings(destPaths)

	updates := make([]update, 0, len(destPathInfos))
	newSourceStateEntries := make(map[string]SourceStateEntry)
	newSourceStateEntriesByTargetName := make(map[string]SourceStateEntry)
	for _, destPath := range destPaths {
		destPathInfo := destPathInfos[destPath]
		if !options.Include.IncludeFileInfo(destPathInfo) {
			continue
		}
		targetName := MustTrimDirPrefix(destPath, s.destDir)

		// Find the target's parent directory.
		var parentDir string
		if parentDirTargetName := path.Dir(targetName); parentDirTargetName == "." {
			parentDir = ""
		} else if parentDirEntry, ok := newSourceStateEntriesByTargetName[parentDirTargetName]; ok {
			parentDir = parentDirEntry.Path()
		} else if parentDirEntry, ok := s.entries[parentDirTargetName]; ok {
			parentDir = parentDirEntry.Path()
		} else {
			return fmt.Errorf("%s: parent directory not in source state", destPath)
		}

		actualStateEntry, err := NewActualStateEntry(sourceSystem, destPath)
		if err != nil {
			return err
		}
		newSourceStateEntry, err := s.sourceStateEntry(actualStateEntry, destPath, destPathInfo, parentDir, options)
		if err != nil {
			return err
		}
		if newSourceStateEntry == nil {
			continue
		}

		sourceEntryName := newSourceStateEntry.Path()

		entryState, err := actualStateEntry.EntryState()
		if err != nil {
			return err
		}
		update := update{
			destPath:              destPath,
			entryState:            entryState,
			sourceStateEntryNames: []string{sourceEntryName},
		}

		if oldSourceStateEntry, ok := s.entries[targetName]; ok {
			// If both the new and old source state entries are directories but the name has changed,
			// rename to avoid losing the directory's contents. Otherwise,
			// remove the old.
			oldSourceEntryName := MustTrimDirPrefix(oldSourceStateEntry.Path(), s.sourceDir)
			if sourceEntryName != oldSourceEntryName {
				_, newIsDir := newSourceStateEntry.(*SourceStateDir)
				_, oldIsDir := oldSourceStateEntry.(*SourceStateDir)
				if newIsDir && oldIsDir {
					newSourceStateEntry = &SourceStateRenameDir{
						oldName: oldSourceEntryName,
						newName: sourceEntryName,
					}
				} else {
					newSourceStateEntries[oldSourceEntryName] = &SourceStateRemove{}
					update.sourceStateEntryNames = append(update.sourceStateEntryNames, oldSourceEntryName)
				}
			}
		}

		newSourceStateEntries[sourceEntryName] = newSourceStateEntry
		newSourceStateEntriesByTargetName[targetName] = newSourceStateEntry

		updates = append(updates, update)
	}

	targetSourceState := &SourceState{
		entries: newSourceStateEntries,
	}
	for _, update := range updates {
		for _, sourceEntryName := range update.sourceStateEntryNames {
			if err := targetSourceState.Apply(sourceSystem, NullPersistentState{}, s.sourceDir, sourceEntryName, ApplyOptions{
				Include: options.Include,
				Umask:   options.umask,
			}); err != nil {
				return err
			}
		}
		value, err := json.Marshal(update.entryState)
		if err != nil {
			return err
		}
		if err := persistentState.Set(EntryStateBucket, []byte(update.destPath), value); err != nil {
			return err
		}
	}

	return nil
}

// AddDestPathInfos adds an os.FileInfo to destPathInfos for destPath and any of
// its parents which are not already known.
func (s *SourceState) AddDestPathInfos(destPathInfos map[string]os.FileInfo, lstater Lstater, destPath string, info os.FileInfo) error {
	if _, err := TrimDirPrefix(destPath, s.destDir); err != nil {
		return err
	}

	if _, ok := destPathInfos[destPath]; ok {
		return nil
	}

	if info == nil {
		var err error
		info, err = lstater.Lstat(destPath)
		if err != nil {
			return err
		}
	}

	destPathInfos[destPath] = info
	destPath = path.Dir(destPath)
	if destPath == s.destDir {
		return nil
	}

	for {
		if _, ok := destPathInfos[destPath]; ok {
			return nil
		}
		info, err := lstater.Lstat(destPath)
		if err != nil {
			return err
		}
		destPathInfos[destPath] = info
		parentDir := path.Dir(destPath)
		if parentDir == s.destDir {
			return nil
		}
		if _, ok := s.entries[parentDir]; ok {
			return nil
		}
		destPath = parentDir
	}
}

// AllTargetNames returns all of s's target names in order.
func (s *SourceState) AllTargetNames() []string {
	targetNames := make([]string, 0, len(s.entries))
	for targetName := range s.entries {
		targetNames = append(targetNames, targetName)
	}
	sort.Slice(targetNames, func(i, j int) bool {
		orderI := s.entries[targetNames[i]].Order()
		orderJ := s.entries[targetNames[j]].Order()
		switch {
		case orderI < orderJ:
			return true
		case orderI == orderJ:
			return targetNames[i] < targetNames[j]
		default:
			return false
		}
	})
	return targetNames
}

// A PreApplyFunc is called before a target is applied.
type PreApplyFunc func(targetName string, targetEntryState, lastWrittenEntryState, actualEntryState *EntryState) error

// ApplyOptions are options to SourceState.ApplyAll and SourceState.ApplyOne.
type ApplyOptions struct {
	Include      *IncludeSet
	PreApplyFunc PreApplyFunc
	Umask        os.FileMode
}

// Apply updates targetName in targetDir in targetSystem to match s.
func (s *SourceState) Apply(targetSystem System, persistentState PersistentState, targetDir, targetName string, options ApplyOptions) error {
	targetStateEntry, err := s.entries[targetName].TargetStateEntry()
	if err != nil {
		return err
	}

	if options.Include != nil && !options.Include.IncludeTargetStateEntry(targetStateEntry) {
		return nil
	}

	targetPath := path.Join(targetDir, targetName)

	targetEntryState, err := targetStateEntry.EntryState()
	if err != nil {
		return err
	}

	actualStateEntry, err := NewActualStateEntry(targetSystem, targetPath)
	if err != nil {
		return err
	}

	if options.PreApplyFunc != nil {
		var lastWrittenEntryState *EntryState
		switch data, err := persistentState.Get(EntryStateBucket, []byte(targetPath)); {
		case err != nil:
			return err
		case data != nil:
			var entryState EntryState
			if err := json.Unmarshal(data, &entryState); err != nil {
				return err
			}
			lastWrittenEntryState = &entryState
		}

		actualEntryState, err := actualStateEntry.EntryState()
		if err != nil {
			return err
		}

		if err := options.PreApplyFunc(targetName, targetEntryState, lastWrittenEntryState, actualEntryState); err != nil {
			return err
		}
	}

	if err := targetStateEntry.Apply(targetSystem, persistentState, actualStateEntry, options.Umask); err != nil {
		return err
	}

	data, err := json.Marshal(targetEntryState)
	if err != nil {
		return err
	}
	return persistentState.Set(EntryStateBucket, []byte(targetPath), data)
}

// Entries returns s's source state entries.
func (s *SourceState) Entries() map[string]SourceStateEntry {
	return s.entries
}

// Entry returns the source state entry for targetName.
func (s *SourceState) Entry(targetName string) (SourceStateEntry, bool) {
	sourceStateEntry, ok := s.entries[targetName]
	return sourceStateEntry, ok
}

// ExecuteTemplateData returns the result of executing template data.
func (s *SourceState) ExecuteTemplateData(name string, data []byte) ([]byte, error) {
	tmpl, err := template.New(name).
		Option(s.templateOptions...).
		Funcs(s.templateFuncs).
		Parse(string(data))
	if err != nil {
		return nil, err
	}
	for name, t := range s.templates {
		tmpl, err = tmpl.AddParseTree(name, t.Tree)
		if err != nil {
			return nil, err
		}
	}
	var sb strings.Builder
	if err = tmpl.ExecuteTemplate(&sb, name, s.TemplateData()); err != nil {
		return nil, err
	}
	return []byte(sb.String()), nil
}

// Ignored returns if targetName is ignored.
func (s *SourceState) Ignored(targetName string) bool {
	return s.ignore.match(targetName)
}

// MinVersion returns the minimum version for which s is valid.
func (s *SourceState) MinVersion() semver.Version {
	return s.minVersion
}

// MustEntry returns the source state entry associated with targetName, and
// panics if it does not exist.
func (s *SourceState) MustEntry(targetName string) SourceStateEntry {
	sourceStateEntry, ok := s.entries[targetName]
	if !ok {
		panic(fmt.Sprintf("%s: no source state entry", targetName))
	}
	return sourceStateEntry
}

// Read reads a source state from sourcePath.
func (s *SourceState) Read() error {
	info, err := s.system.Lstat(s.sourceDir)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return err
	case !info.IsDir():
		return fmt.Errorf("%s: not a directory", s.sourceDir)
	}

	// Read all source entries.
	allSourceStateEntries := make(map[string][]SourceStateEntry)
	if err := vfs.WalkSlash(s.system, s.sourceDir, func(sourcePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if sourcePath == s.sourceDir {
			return nil
		}
		relPath := MustTrimDirPrefix(sourcePath, s.sourceDir)
		sourceDirName, sourceName := path.Split(relPath)
		targetDirName := getTargetDirName(sourceDirName)
		// Follow symlinks from the source directory.
		if info.Mode()&os.ModeType == os.ModeSymlink {
			info, err = s.system.Stat(sourcePath)
			if err != nil {
				return err
			}
		}
		switch {
		case strings.HasPrefix(info.Name(), dataName):
			return s.addTemplateData(sourcePath)
		case info.Name() == ignoreName:
			// .chezmoiignore is interpreted as a template. vfs.WalkSlash walks
			// in alphabetical order, so, luckily for us, .chezmoidata will be
			// read before .chezmoiignore, so data in .chezmoidata is available
			// to be used in .chezmoiignore. Unluckily for us, .chezmoitemplates
			// will be read afterwards so partial templates will not be
			// available in .chezmoiignore.
			return s.addPatterns(s.ignore, sourcePath, sourceDirName)
		case info.Name() == removeName:
			// The comment about .chezmoiignore and templates applies to
			// .chezmoiremove too.
			removePatterns := newPatternSet()
			if err := s.addPatterns(removePatterns, sourcePath, targetDirName); err != nil {
				return err
			}
			matches, err := removePatterns.glob(s.system.UnderlyingFS(), s.destDir+"/")
			if err != nil {
				return err
			}
			n := 0
			for _, match := range matches {
				if !s.Ignored(match) {
					matches[n] = match
					n++
				}
			}
			matches = matches[:n]
			sourceStateEntry := &SourceStateRemove{
				path: sourcePath,
			}
			for _, match := range matches {
				allSourceStateEntries[match] = append(allSourceStateEntries[match], sourceStateEntry)
			}
			return nil
		case info.Name() == templatesDirName:
			if err := s.addTemplatesDir(sourcePath); err != nil {
				return err
			}
			return vfs.SkipDir
		case info.Name() == versionName:
			return s.addVersionFile(sourcePath)
		case strings.HasPrefix(info.Name(), Prefix):
			fallthrough
		case strings.HasPrefix(info.Name(), ignorePrefix):
			if info.IsDir() {
				return vfs.SkipDir
			}
			return nil
		case info.IsDir():
			da := parseDirAttr(sourceName)
			targetName := path.Join(targetDirName, da.Name)
			if s.Ignored(targetName) {
				return vfs.SkipDir
			}
			sourceStateEntry := s.newSourceStateDir(sourcePath, da)
			allSourceStateEntries[targetName] = append(allSourceStateEntries[targetName], sourceStateEntry)
			return nil
		case info.Mode().IsRegular():
			fa := parseFileAttr(sourceName)
			targetName := path.Join(targetDirName, fa.Name)
			if s.Ignored(targetName) {
				return nil
			}
			sourceStateEntry := s.newSourceStateFile(sourcePath, fa, targetName)
			allSourceStateEntries[targetName] = append(allSourceStateEntries[targetName], sourceStateEntry)
			return nil
		default:
			return &unsupportedFileTypeError{
				path: sourcePath,
				mode: info.Mode(),
			}
		}
	}); err != nil {
		return err
	}

	// Remove all ignored targets.
	for targetName := range allSourceStateEntries {
		if s.Ignored(targetName) {
			delete(allSourceStateEntries, targetName)
		}
	}

	// Generate SourceStateRemoves for exact directories.
	for targetName, sourceStateEntries := range allSourceStateEntries {
		if len(sourceStateEntries) != 1 {
			continue
		}
		sourceStateDir, ok := sourceStateEntries[0].(*SourceStateDir)
		if !ok {
			continue
		}
		if !sourceStateDir.Attr.Exact {
			continue
		}
		sourceStateRemove := &SourceStateRemove{
			path: sourceStateDir.Path(),
		}
		infos, err := s.system.ReadDir(path.Join(s.destDir, targetName))
		switch {
		case err == nil:
			for _, info := range infos {
				name := info.Name()
				if name == "." || name == ".." {
					continue
				}
				targetEntryName := path.Join(targetName, name)
				if _, ok := allSourceStateEntries[targetEntryName]; ok {
					continue
				}
				if s.Ignored(targetEntryName) {
					continue
				}
				allSourceStateEntries[targetEntryName] = append(allSourceStateEntries[targetEntryName], sourceStateRemove)
			}
		case os.IsNotExist(err):
			// Do nothing.
		default:
			return err
		}
	}

	// Check for duplicate source entries with the same target name. Iterate
	// over the target names in order so that any error is deterministic.
	targetNames := make([]string, 0, len(allSourceStateEntries))
	for targetName := range allSourceStateEntries {
		targetNames = append(targetNames, targetName)
	}
	sort.Strings(targetNames)
	for _, targetName := range targetNames {
		sourceStateEntries := allSourceStateEntries[targetName]
		if len(sourceStateEntries) == 1 {
			continue
		}
		sourcePaths := make([]string, 0, len(sourceStateEntries))
		for _, sourceStateEntry := range sourceStateEntries {
			sourcePaths = append(sourcePaths, sourceStateEntry.Path())
		}
		err = multierr.Append(err, &duplicateTargetError{
			targetName:  targetName,
			sourcePaths: sourcePaths,
		})
	}
	if err != nil {
		return err
	}

	// Populate s.Entries with the unique source entry for each target.
	for targetName, sourceEntries := range allSourceStateEntries {
		s.entries[targetName] = sourceEntries[0]
	}

	return nil
}

// TargetNames returns all of s's target names in alphabetical order.
func (s *SourceState) TargetNames() []string {
	targetNames := make([]string, 0, len(s.entries))
	for targetName := range s.entries {
		targetNames = append(targetNames, targetName)
	}
	sort.Strings(targetNames)
	return targetNames
}

// TemplateData returns s's template data.
func (s *SourceState) TemplateData() map[string]interface{} {
	return s.templateData
}

// addPatterns executes the template at sourcePath, interprets the result as a
// list of patterns, and adds all patterns found to patternSet.
func (s *SourceState) addPatterns(patternSet *patternSet, sourcePath, relPath string) error {
	data, err := s.executeTemplate(sourcePath)
	if err != nil {
		return err
	}
	dir := path.Dir(relPath)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var lineNumber int
	for scanner.Scan() {
		lineNumber++
		text := scanner.Text()
		if index := strings.IndexRune(text, '#'); index != -1 {
			text = text[:index]
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		include := true
		if strings.HasPrefix(text, "!") {
			include = false
			text = mustTrimPrefix(text, "!")
		}
		pattern := path.Join(dir, text)
		if err := patternSet.add(pattern, include); err != nil {
			return fmt.Errorf("%s:%d: %w", sourcePath, lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s: %w", sourcePath, err)
	}
	return nil
}

// addTemplateData adds all template data in sourcePath to s.
func (s *SourceState) addTemplateData(sourcePath string) error {
	_, name := path.Split(sourcePath)
	suffix := mustTrimPrefix(name, dataName+".")
	format, ok := Formats[suffix]
	if !ok {
		return fmt.Errorf("%s: unknown format", sourcePath)
	}
	data, err := s.system.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("%s: %w", sourcePath, err)
	}
	var templateData map[string]interface{}
	if err := format.Decode(data, &templateData); err != nil {
		return fmt.Errorf("%s: %w", sourcePath, err)
	}
	recursiveMerge(s.templateData, templateData)
	recursiveMerge(s.templateData, s.priorityTemplateData)
	return nil
}

// addTemplatesDir adds all templates in templateDir to s.
func (s *SourceState) addTemplatesDir(templateDir string) error {
	return vfs.WalkSlash(s.system, templateDir, func(templatePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		switch {
		case info.Mode().IsRegular():
			contents, err := s.system.ReadFile(templatePath)
			if err != nil {
				return err
			}
			name := MustTrimDirPrefix(templatePath, templateDir)
			tmpl, err := template.New(name).Option(s.templateOptions...).Funcs(s.templateFuncs).Parse(string(contents))
			if err != nil {
				return err
			}
			if s.templates == nil {
				s.templates = make(map[string]*template.Template)
			}
			s.templates[name] = tmpl
			return nil
		case info.IsDir():
			return nil
		default:
			return &unsupportedFileTypeError{
				path: templatePath,
				mode: info.Mode(),
			}
		}
	})
}

// addVersionFile reads a .chezmoiversion file from source path and updates s's
// minimum version if it contains a more recent version than the current minimum
// version.
func (s *SourceState) addVersionFile(sourcePath string) error {
	data, err := s.system.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	version, err := semver.NewVersion(strings.TrimSpace(string(data)))
	if err != nil {
		return err
	}
	if s.minVersion.LessThan(*version) {
		s.minVersion = *version
	}
	return nil
}

// applyAll updates targetDir in fs to match s.
func (s *SourceState) applyAll(targetSystem System, persistentState PersistentState, targetDir string, options ApplyOptions) error {
	for _, targetName := range s.AllTargetNames() {
		switch err := s.Apply(targetSystem, persistentState, targetDir, targetName, options); {
		case errors.Is(err, Skip):
			continue
		case err != nil:
			return err
		}
	}
	return nil
}

// executeTemplate executes the template at path and returns the result.
func (s *SourceState) executeTemplate(path string) ([]byte, error) {
	data, err := s.system.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.ExecuteTemplateData(path, data)
}

// newSourceStateDir returns a new SourceStateDir.
func (s *SourceState) newSourceStateDir(sourcePath string, dirAttr DirAttr) *SourceStateDir {
	targetStateDir := &TargetStateDir{
		perm: dirAttr.perm(),
	}
	return &SourceStateDir{
		path:             sourcePath,
		Attr:             dirAttr,
		targetStateEntry: targetStateDir,
	}
}

// newSourceStateFile returns a new SourceStateFile.
func (s *SourceState) newSourceStateFile(sourcePath string, fileAttr FileAttr, targetName string) *SourceStateFile {
	lazyContents := &lazyContents{
		contentsFunc: func() ([]byte, error) {
			contents, err := s.system.ReadFile(sourcePath)
			if err != nil {
				return nil, err
			}
			if !fileAttr.Encrypted {
				return contents, nil
			}
			return s.encryptionTool.Decrypt(path.Base(targetName), contents)
		},
	}

	var targetStateEntryFunc func() (TargetStateEntry, error)
	switch fileAttr.Type {
	case SourceFileTypeFile:
		targetStateEntryFunc = func() (TargetStateEntry, error) {
			contents, err := lazyContents.Contents()
			if err != nil {
				return nil, err
			}
			if fileAttr.Template {
				contents, err = s.ExecuteTemplateData(sourcePath, contents)
				if err != nil {
					return nil, err
				}
			}
			if !fileAttr.Empty && isEmpty(contents) {
				return &TargetStateAbsent{}, nil
			}
			return &TargetStateFile{
				lazyContents: newLazyContents(contents),
				perm:         fileAttr.perm(),
			}, nil
		}
	case SourceFileTypePresent:
		targetStateEntryFunc = func() (TargetStateEntry, error) {
			contents, err := lazyContents.Contents()
			if err != nil {
				return nil, err
			}
			if fileAttr.Template {
				contents, err = s.ExecuteTemplateData(sourcePath, contents)
				if err != nil {
					return nil, err
				}
			}
			return &TargetStatePresent{
				lazyContents: newLazyContents(contents),
				perm:         fileAttr.perm(),
			}, nil
		}
	case SourceFileTypeScript:
		targetStateEntryFunc = func() (TargetStateEntry, error) {
			contents, err := lazyContents.Contents()
			if err != nil {
				return nil, err
			}
			if fileAttr.Template {
				contents, err = s.ExecuteTemplateData(sourcePath, contents)
				if err != nil {
					return nil, err
				}
			}
			return &TargetStateScript{
				lazyContents: newLazyContents(contents),
				name:         targetName,
				once:         fileAttr.Once,
			}, nil
		}
	case SourceFileTypeSymlink:
		targetStateEntryFunc = func() (TargetStateEntry, error) {
			linknameBytes, err := lazyContents.Contents()
			if err != nil {
				return nil, err
			}
			if fileAttr.Template {
				linknameBytes, err = s.ExecuteTemplateData(sourcePath, linknameBytes)
				if err != nil {
					return nil, err
				}
			}
			return &TargetStateSymlink{
				lazyLinkname: newLazyLinkname(string(bytes.TrimSpace(linknameBytes))),
			}, nil
		}
	default:
		panic(fmt.Sprintf("%d: unsupported type", fileAttr.Type))
	}

	return &SourceStateFile{
		lazyContents:         lazyContents,
		path:                 sourcePath,
		Attr:                 fileAttr,
		targetStateEntryFunc: targetStateEntryFunc,
	}
}

// sourceStateEntry returns a new SourceStateEntry based on actualStateEntry.
func (s *SourceState) sourceStateEntry(actualStateEntry ActualStateEntry, destPath string, info os.FileInfo, parentDir string, options *AddOptions) (SourceStateEntry, error) {
	switch actualStateEntry := actualStateEntry.(type) {
	case *ActualStateAbsent:
		return nil, fmt.Errorf("%s: not found", destPath)
	case *ActualStateDir:
		dirAttr := DirAttr{
			Name:    info.Name(),
			Exact:   options.Exact,
			Private: isPrivate(info),
		}
		return &SourceStateDir{
			path: path.Join(parentDir, dirAttr.BaseName()),
			Attr: dirAttr,
			targetStateEntry: &TargetStateDir{
				perm: 0o777,
			},
		}, nil
	case *ActualStateFile:
		fileAttr := FileAttr{
			Name:       info.Name(),
			Type:       SourceFileTypeFile,
			Empty:      options.Empty,
			Encrypted:  options.Encrypt,
			Executable: isExecutable(info),
			Private:    isPrivate(info),
			Template:   options.Template || options.AutoTemplate,
		}
		contents, err := actualStateEntry.Contents()
		if err != nil {
			return nil, err
		}
		if options.AutoTemplate {
			contents = autoTemplate(contents, s.TemplateData())
		}
		if len(contents) == 0 && !options.Empty {
			return nil, nil
		}
		lazyContents := &lazyContents{
			contents: contents,
		}
		return &SourceStateFile{
			path:         path.Join(parentDir, fileAttr.BaseName()),
			Attr:         fileAttr,
			lazyContents: lazyContents,
			targetStateEntry: &TargetStateFile{
				lazyContents: lazyContents,
				perm:         0o666,
			},
		}, nil
	case *ActualStateSymlink:
		fileAttr := FileAttr{
			Name:     info.Name(),
			Type:     SourceFileTypeSymlink,
			Template: options.Template || options.AutoTemplate,
		}
		linkname, err := actualStateEntry.Linkname()
		if err != nil {
			return nil, err
		}
		contents := []byte(filepath.ToSlash(linkname))
		if options.AutoTemplate {
			contents = autoTemplate(contents, s.TemplateData())
		}
		contents = append(contents, '\n')
		lazyContents := &lazyContents{
			contents: contents,
		}
		return &SourceStateFile{
			path:         path.Join(parentDir, fileAttr.BaseName()),
			Attr:         fileAttr,
			lazyContents: lazyContents,
			targetStateEntry: &TargetStateFile{
				lazyContents: lazyContents,
				perm:         0o666,
			},
		}, nil
	default:
		panic(fmt.Sprintf("%T: unsupported type", actualStateEntry))
	}
}

// getTargetDirName returns the target directory name of sourceDirName.
func getTargetDirName(sourceDirName string) string {
	sourceNames := strings.Split(sourceDirName, "/")
	targetNames := make([]string, 0, len(sourceNames))
	for _, sourceName := range sourceNames {
		da := parseDirAttr(sourceName)
		targetNames = append(targetNames, da.Name)
	}
	return strings.Join(targetNames, "/")
}

package chezmoi

import (
	"os"
	"testing"
	"text/template"

	"github.com/coreos/go-semver/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vfs "github.com/twpayne/go-vfs"
	"github.com/twpayne/go-vfs/vfst"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoitest"
)

func TestSourceStateAdd(t *testing.T) {
	for _, tc := range []struct {
		name       string
		destPaths  []string
		addOptions AddOptions
		extraRoot  interface{}
		tests      []interface{}
	}{
		{
			name: "dir",
			destPaths: []string{
				"/home/user/.dir",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/file",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/subdir",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "dir_change_attributes",
			destPaths: []string{
				"/home/user/.dir",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			extraRoot: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi/exact_dot_dir/file": "# contents of file\n",
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/exact_dot_dir",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/file",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of file\n"),
				),
			},
		},
		{
			name: "dir_file",
			destPaths: []string{
				"/home/user/.dir/file",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/file",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .dir/file\n"),
				),
			},
		},
		{
			name: "dir_file_existing_dir",
			destPaths: []string{
				"/home/user/.dir/file",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			extraRoot: map[string]interface{}{
				"/home/user/.local/share/chezmoi/dot_dir": &vfst.Dir{Perm: 0o777},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/file",
					vfst.TestModeIsRegular,
					vfst.TestContentsString("# contents of .dir/file\n"),
				),
			},
		},
		{
			name: "dir_subdir",
			destPaths: []string{
				"/home/user/.dir/subdir",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/subdir",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/subdir/file",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "dir_subdir_file",
			destPaths: []string{
				"/home/user/.dir/subdir/file",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/file",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/subdir",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/subdir/file",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .dir/subdir/file\n"),
				),
			},
		},
		{
			name: "dir_subdir_file_existing_dir_subdir",
			destPaths: []string{
				"/home/user/.dir/subdir/file",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			extraRoot: map[string]interface{}{
				"/home/user/.local/share/chezmoi/dot_dir/subdir": &vfst.Dir{Perm: 0o777},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/subdir/file",
					vfst.TestModeIsRegular,
					vfst.TestContentsString("# contents of .dir/subdir/file\n"),
				),
			},
		},
		{
			name: "empty",
			destPaths: []string{
				"/home/user/.empty",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_empty",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "empty_with_empty",
			destPaths: []string{
				"/home/user/.empty",
			},
			addOptions: AddOptions{
				Empty:   true,
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/empty_dot_empty",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContents(nil),
				),
			},
		},
		{
			name: "executable",
			destPaths: []string{
				"/home/user/.executable",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/executable_dot_executable",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .executable\n"),
				),
			},
		},
		{
			name: "exists",
			destPaths: []string{
				"/home/user/.exists",
			},
			addOptions: AddOptions{
				Exists:  true,
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/exists_dot_exists",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .exists\n"),
				),
			},
		},
		{
			name: "file",
			destPaths: []string{
				"/home/user/.file",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_file",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .file\n"),
				),
			},
		},
		{
			name: "file_change_attributes",
			destPaths: []string{
				"/home/user/.file",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			extraRoot: map[string]interface{}{
				"/home/user/.local/share/chezmoi/executable_dot_file": "# contents of .file\n",
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_file",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .file\n"),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/executable_dot_file",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "file_replace_contents",
			destPaths: []string{
				"/home/user/.file",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			extraRoot: map[string]interface{}{
				"/home/user/.local/share/chezmoi/dot_file": "# old contents of .file\n",
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_file",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .file\n"),
				),
			},
		},
		{
			name: "private_unix",
			destPaths: []string{
				"/home/user/.private",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/private_dot_private",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .private\n"),
				),
			},
		},
		{
			name: "private_windows",
			destPaths: []string{
				"/home/user/.private",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_private",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .private\n"),
				),
			},
		},
		{
			name: "symlink",
			destPaths: []string{
				"/home/user/.symlink",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			extraRoot: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".symlink": &vfst.Symlink{Target: ".dir/subdir/file"},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/symlink_dot_symlink",
					vfst.TestModeIsRegular,
					vfst.TestContentsString(".dir/subdir/file\n"),
				),
			},
		},
		{
			name: "symlink_windows",
			destPaths: []string{
				"/home/user/.symlink",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			extraRoot: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".symlink": &vfst.Symlink{Target: ".dir\\subdir\\file"},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/symlink_dot_symlink",
					vfst.TestModeIsRegular,
					vfst.TestContentsString(".dir/subdir/file\n"),
				),
			},
		},
		{
			name: "template",
			destPaths: []string{
				"/home/user/.template",
			},
			addOptions: AddOptions{
				AutoTemplate: true,
				Include:      NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_template.tmpl",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("key = {{ .variable }}\n"),
				),
			},
		},
		{
			name: "dir_and_dir_file",
			destPaths: []string{
				"/home/user/.dir",
				"/home/user/.dir/file",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/file",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .dir/file\n"),
				),
			},
		},
		// FIXME enable following test which currently fails
		/*
			{
				name: "file_in_dir_exact_subdir",
				destPaths: []string{
					"/home/user/.dir/subdir/file",
				},
				addOptions: AddOptions{
					Include: NewIncludeSet(IncludeAll),
				},
				extraRoot: map[string]interface{}{
					"/home/user/.local/share/chezmoi/dot_dir/exact_subdir": &vfst.Dir{Perm: 0o777},
				},
				tests: []interface{}{
					vfst.TestPath("/home/user/.local/share/chezmoi/dot_dir/exact_subdir/file",
						vfst.TestModeIsRegular,
						vfst.TestContentsString("# contents of .dir/subdir/file\n"),
					),
				},
			},
		*/
	} {
		t.Run(tc.name, func(t *testing.T) {
			chezmoitest.SkipUnlessGOOS(t, tc.name)

			chezmoitest.WithTestFS(t, map[string]interface{}{
				"/home/user": map[string]interface{}{
					".dir": map[string]interface{}{
						"file": "# contents of .dir/file\n",
						"subdir": map[string]interface{}{
							"file": "# contents of .dir/subdir/file\n",
						},
					},
					".empty": "",
					".executable": &vfst.File{
						Perm:     0o777,
						Contents: []byte("# contents of .executable\n"),
					},
					".exists": "# contents of .exists\n",
					".file":   "# contents of .file\n",
					".local": map[string]interface{}{
						"share": map[string]interface{}{
							"chezmoi": &vfst.Dir{Perm: 0o777},
						},
					},
					".private": &vfst.File{
						Perm:     0o600,
						Contents: []byte("# contents of .private\n"),
					},
					".symlink":  &vfst.Symlink{Target: ".dir/subdir/file"},
					".template": "key = value\n",
				},
			}, func(fs vfs.FS) {
				if tc.extraRoot != nil {
					require.NoError(t, vfst.NewBuilder().Build(fs, tc.extraRoot))
				}
				system := NewRealSystem(fs)
				persistentState := NewMockPersistentState()

				s := NewSourceState(
					WithDestDir("/home/user"),
					WithSourceDir("/home/user/.local/share/chezmoi"),
					WithSystem(system),
					withUserTemplateData(map[string]interface{}{
						"variable": "value",
					}),
				)
				require.NoError(t, s.Read())
				require.NoError(t, s.evaluateAll())

				destPathInfos := make(map[string]os.FileInfo)
				for _, destPath := range tc.destPaths {
					require.NoError(t, s.AddDestPathInfos(destPathInfos, system, destPath, nil))
				}
				require.NoError(t, s.Add(system, persistentState, destPathInfos, &tc.addOptions))

				vfst.RunTests(t, fs, "", tc.tests...)
			})
		})
	}
}

func TestSourceStateApplyAll(t *testing.T) {
	// FIXME script tests
	// FIXME script template tests

	for _, tc := range []struct {
		name               string
		root               interface{}
		sourceStateOptions []SourceStateOption
		tests              []interface{}
	}{
		{
			name: "empty",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": &vfst.Dir{Perm: 0o777},
				},
			},
		},
		{
			name: "dir",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"foo": &vfst.Dir{Perm: 0o777},
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
			},
		},
		{
			name: "dir_exact",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": "",
					},
					".local/share/chezmoi": map[string]interface{}{
						"exact_foo": &vfst.Dir{Perm: 0o777},
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/foo/bar",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "file",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("bar"),
				),
			},
		},
		{
			name: "file_remove_empty",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					"foo": "",
					".local/share/chezmoi": map[string]interface{}{
						"foo": "",
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "file_create_empty",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"empty_foo": "",
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString(""),
				),
			},
		},
		{
			name: "file_template",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"foo.tmpl": "email = {{ .email }}",
					},
				},
			},
			sourceStateOptions: []SourceStateOption{
				withUserTemplateData(map[string]interface{}{
					"email": "you@example.com",
				}),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("email = you@example.com"),
				),
			},
		},
		{
			name: "exists_create",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"exists_foo": "bar",
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("bar"),
				),
			},
		},
		{
			name: "exists_no_replace",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"exists_foo": "bar",
					},
					"foo": "baz",
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("baz"),
				),
			},
		},
		{
			name: "symlink",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"symlink_foo": "bar",
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeType(os.ModeSymlink),
					vfst.TestSymlinkTarget("bar"),
				),
			},
		},
		{
			name: "symlink_template",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"symlink_foo.tmpl": "bar_{{ .os }}",
					},
				},
			},
			sourceStateOptions: []SourceStateOption{
				withUserTemplateData(map[string]interface{}{
					"os": "linux",
				}),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeType(os.ModeSymlink),
					vfst.TestSymlinkTarget("bar_linux"),
				),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chezmoitest.WithTestFS(t, tc.root, func(fs vfs.FS) {
				system := NewRealSystem(fs)
				persistentState := NewMockPersistentState()
				sourceStateOptions := []SourceStateOption{
					WithDestDir("/home/user"),
					WithSourceDir("/home/user/.local/share/chezmoi"),
					WithSystem(system),
				}
				sourceStateOptions = append(sourceStateOptions, tc.sourceStateOptions...)
				s := NewSourceState(sourceStateOptions...)
				require.NoError(t, s.Read())
				require.NoError(t, s.evaluateAll())
				require.NoError(t, s.applyAll(system, persistentState, "/home/user", ApplyOptions{
					Umask: GetUmask(),
				}))

				vfst.RunTests(t, fs, "", tc.tests...)
			})
		})
	}
}

func TestSourceStateRead(t *testing.T) {
	for _, tc := range []struct {
		name                string
		root                interface{}
		expectedError       string
		expectedSourceState *SourceState
	}{
		{
			name: "empty",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": &vfst.Dir{Perm: 0o777},
			},
			expectedSourceState: NewSourceState(),
		},
		{
			name: "dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo": &vfst.Dir{Perm: 0o777},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateDir{
						path: "/home/user/.local/share/chezmoi/foo",
						Attr: DirAttr{
							Name: "foo",
						},
						targetStateEntry: &TargetStateDir{
							perm: 0o777,
						},
					},
				}),
			),
		},
		{
			name: "file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo": "bar",
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/foo",
						Attr: FileAttr{
							Name: "foo",
							Type: SourceFileTypeFile,
						},
						lazyContents: newLazyContents([]byte("bar")),
						targetStateEntry: &TargetStateFile{
							perm:         0o666,
							lazyContents: newLazyContents([]byte("bar")),
						},
					},
				}),
			),
		},
		{
			name: "duplicate_target_file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo":      "bar",
					"foo.tmpl": "bar",
				},
			},
			expectedError: "foo: duplicate target (/home/user/.local/share/chezmoi/foo, /home/user/.local/share/chezmoi/foo.tmpl)",
		},
		{
			name: "duplicate_target_dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo":       "bar",
					"exact_foo": &vfst.Dir{Perm: 0o777},
				},
			},
			expectedError: "foo: duplicate target (/home/user/.local/share/chezmoi/exact_foo, /home/user/.local/share/chezmoi/foo)",
		},
		{
			name: "duplicate_target_script",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"run_script":      "#!/bin/sh\n",
					"run_once_script": "#!/bin/sh\n",
				},
			},
			expectedError: "script: duplicate target (/home/user/.local/share/chezmoi/run_once_script, /home/user/.local/share/chezmoi/run_script)",
		},
		{
			name: "symlink_with_attr",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"bar":            "baz",
					"executable_foo": &vfst.Symlink{Target: "bar"},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"bar": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/bar",
						Attr: FileAttr{
							Name: "bar",
							Type: SourceFileTypeFile,
						},
						lazyContents: newLazyContents([]byte("baz")),
						targetStateEntry: &TargetStateFile{
							perm:         0o666,
							lazyContents: newLazyContents([]byte("baz")),
						},
					},
					"foo": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/executable_foo",
						Attr: FileAttr{
							Name:       "foo",
							Type:       SourceFileTypeFile,
							Executable: true,
						},
						lazyContents: newLazyContents([]byte("baz")),
						targetStateEntry: &TargetStateFile{
							perm:         0o777,
							lazyContents: newLazyContents([]byte("baz")),
						},
					},
				}),
			),
		},
		{
			name: "symlink_script",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"bar":     "baz",
					"run_foo": &vfst.Symlink{Target: "bar"},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"bar": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/bar",
						Attr: FileAttr{
							Name: "bar",
							Type: SourceFileTypeFile,
						},
						lazyContents: newLazyContents([]byte("baz")),
						targetStateEntry: &TargetStateFile{
							perm:         0o666,
							lazyContents: newLazyContents([]byte("baz")),
						},
					},
					"foo": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/run_foo",
						Attr: FileAttr{
							Name: "foo",
							Type: SourceFileTypeScript,
						},
						lazyContents: newLazyContents([]byte("baz")),
						targetStateEntry: &TargetStateScript{
							name:         "foo",
							lazyContents: newLazyContents([]byte("baz")),
						},
					},
				}),
			),
		},
		{
			name: "script",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"run_foo": "bar",
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/run_foo",
						Attr: FileAttr{
							Name: "foo",
							Type: SourceFileTypeScript,
						},
						lazyContents: newLazyContents([]byte("bar")),
						targetStateEntry: &TargetStateScript{
							name:         "foo",
							lazyContents: newLazyContents([]byte("bar")),
						},
					},
				}),
			),
		},
		{
			name: "symlink",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"symlink_foo": "bar",
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/symlink_foo",
						Attr: FileAttr{
							Name: "foo",
							Type: SourceFileTypeSymlink,
						},
						lazyContents: newLazyContents([]byte("bar")),
						targetStateEntry: &TargetStateSymlink{
							lazyLinkname: newLazyLinkname("bar"),
						},
					},
				}),
			),
		},
		{
			name: "file_in_dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": "baz",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateDir{
						path: "/home/user/.local/share/chezmoi/foo",
						Attr: DirAttr{
							Name: "foo",
						},
						targetStateEntry: &TargetStateDir{
							perm: 0o777,
						},
					},
					"foo/bar": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/foo/bar",
						Attr: FileAttr{
							Name: "bar",
							Type: SourceFileTypeFile,
						},
						lazyContents: &lazyContents{
							contents: []byte("baz"),
						},
						targetStateEntry: &TargetStateFile{
							perm: 0o666,
							lazyContents: &lazyContents{
								contents: []byte("baz"),
							},
						},
					},
				}),
			),
		},
		{
			name: "chezmoiignore",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiignore": "README.md\n",
				},
			},
			expectedSourceState: NewSourceState(
				withIgnore(
					mustNewPatternSet(t, map[string]bool{
						"README.md": true,
					}),
				),
			),
		},
		{
			name: "chezmoiignore_ignore_file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiignore": "README.md\n",
					"README.md":      "",
				},
			},
			expectedSourceState: NewSourceState(
				withIgnore(
					mustNewPatternSet(t, map[string]bool{
						"README.md": true,
					}),
				),
			),
		},
		{
			name: "chezmoiignore_exact_dir",
			root: map[string]interface{}{
				"/home/user/dir": map[string]interface{}{
					"bar": "# contents of dir/bar\n",
					"baz": "# contents of dir/baz\n",
					"foo": "# contents of dir/foo\n",
				},
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiignore": "dir/baz\n",
					"exact_dir": map[string]interface{}{
						"bar": "# contents of dir/bar\n",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"dir": &SourceStateDir{
						path: "/home/user/.local/share/chezmoi/exact_dir",
						Attr: DirAttr{
							Name:  "dir",
							Exact: true,
						},
						targetStateEntry: &TargetStateDir{
							perm: 0o777,
						},
					},
					"dir/bar": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/exact_dir/bar",
						Attr: FileAttr{
							Name: "bar",
							Type: SourceFileTypeFile,
						},
						lazyContents: &lazyContents{
							contents: []byte("# contents of dir/bar\n"),
						},
						targetStateEntry: &TargetStateFile{
							perm: 0o666,
							lazyContents: &lazyContents{
								contents: []byte("# contents of dir/bar\n"),
							},
						},
					},
					"dir/foo": &SourceStateRemove{
						path: "/home/user/.local/share/chezmoi/exact_dir",
					},
				}),
				withIgnore(
					mustNewPatternSet(t, map[string]bool{
						"dir/baz": true,
					}),
				),
			),
		},
		{
			name: "chezmoiremove",
			root: map[string]interface{}{
				"/home/user/foo": "",
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiremove": "foo\n",
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateRemove{
						path: "/home/user/.local/share/chezmoi/.chezmoiremove",
					},
				}),
			),
		},
		{
			name: "chezmoiremove_and_ignore",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					"bar": "",
					"baz": "",
				},
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiignore": "baz\n",
					".chezmoiremove": "b*\n",
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"bar": &SourceStateRemove{
						path: "/home/user/.local/share/chezmoi/.chezmoiremove",
					},
				}),
				withIgnore(
					mustNewPatternSet(t, map[string]bool{
						"baz": true,
					}),
				),
			),
		},
		{
			name: "chezmoitemplates",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoitemplates": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withTemplates(
					map[string]*template.Template{
						"foo": template.Must(template.New("foo").Option("missingkey=error").Parse("bar")),
					},
				),
			),
		},
		{
			name: "chezmoiversion",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiversion": "1.2.3\n",
				},
			},
			expectedSourceState: NewSourceState(
				withMinVersion(
					semver.Version{
						Major: 1,
						Minor: 2,
						Patch: 3,
					},
				),
			),
		},
		{
			name: "chezmoiversion_multiple",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiversion": "1.2.3\n",
					"foo": map[string]interface{}{
						".chezmoiversion": "2.3.4\n",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateDir{
						path: "/home/user/.local/share/chezmoi/foo",
						Attr: DirAttr{
							Name: "foo",
						},
						targetStateEntry: &TargetStateDir{
							perm: 0o777,
						},
					},
				}),
				withMinVersion(
					semver.Version{
						Major: 2,
						Minor: 3,
						Patch: 4,
					},
				),
			),
		},
		{
			name: "ignore_dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".ignore": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expectedSourceState: NewSourceState(),
		},
		{
			name: "ignore_file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".ignore": "",
				},
			},
			expectedSourceState: NewSourceState(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chezmoitest.WithTestFS(t, tc.root, func(fs vfs.FS) {
				s := NewSourceState(
					WithDestDir("/home/user"),
					WithSourceDir("/home/user/.local/share/chezmoi"),
					WithSystem(NewRealSystem(fs)),
				)
				err := s.Read()
				if tc.expectedError != "" {
					assert.Error(t, err)
					assert.Equal(t, tc.expectedError, err.Error())
					return
				}
				require.NoError(t, err)
				require.NoError(t, s.evaluateAll())
				tc.expectedSourceState.destDir = "/home/user"
				tc.expectedSourceState.sourceDir = "/home/user/.local/share/chezmoi"
				require.NoError(t, tc.expectedSourceState.evaluateAll())
				s.system = nil
				s.templateData = nil
				assert.Equal(t, tc.expectedSourceState, s)
			})
		})
	}
}

func TestSourceStateSortedTargetNames(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		root                      interface{}
		expectedSortedTargetNames []string
	}{
		{
			name:                      "empty",
			root:                      nil,
			expectedSortedTargetNames: []string{},
		},
		{
			name: "scripts",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"run_first_1first": "",
					"run_first_2first": "",
					"run_first_3first": "",
					"run_1":            "",
					"run_2":            "",
					"run_3":            "",
					"run_last_1last":   "",
					"run_last_2last":   "",
					"run_last_3last":   "",
				},
			},
			expectedSortedTargetNames: []string{
				"1first",
				"2first",
				"3first",
				"1",
				"2",
				"3",
				"1last",
				"2last",
				"3last",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chezmoitest.WithTestFS(t, tc.root, func(fs vfs.FS) {
				s := NewSourceState(
					WithSourceDir("/home/user/.local/share/chezmoi"),
					WithSystem(NewRealSystem(fs)),
				)
				require.NoError(t, s.Read())
				assert.Equal(t, tc.expectedSortedTargetNames, s.AllTargetNames())
			})
		})
	}
}

// evaluateAll evaluates every target state entry in s.
func (s *SourceState) evaluateAll() error {
	for _, targetName := range s.AllTargetNames() {
		sourceStateEntry := s.entries[targetName]
		if err := sourceStateEntry.Evaluate(); err != nil {
			return err
		}
		targetStateEntry, err := sourceStateEntry.TargetStateEntry()
		if err != nil {
			return err
		}
		if err := targetStateEntry.Evaluate(); err != nil {
			return err
		}
	}
	return nil
}

func withEntries(sourceEntries map[string]SourceStateEntry) SourceStateOption {
	return func(s *SourceState) {
		s.entries = sourceEntries
	}
}

func withIgnore(ignore *patternSet) SourceStateOption {
	return func(s *SourceState) {
		s.ignore = ignore
	}
}

func withMinVersion(minVersion semver.Version) SourceStateOption {
	return func(s *SourceState) {
		s.minVersion = minVersion
	}
}

// withUserTemplateData adds template data.
func withUserTemplateData(templateData map[string]interface{}) SourceStateOption {
	return func(s *SourceState) {
		recursiveMerge(s.userTemplateData, templateData)
	}
}

func withTemplates(templates map[string]*template.Template) SourceStateOption {
	return func(s *SourceState) {
		s.templates = templates
	}
}

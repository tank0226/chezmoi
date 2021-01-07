package chezmoi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vfs "github.com/twpayne/go-vfs"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoitest"
)

var _ System = &DumpSystem{}

func TestDumpSystem(t *testing.T) {
	chezmoitest.WithTestFS(t, map[string]interface{}{
		"/home/user/.local/share/chezmoi": map[string]interface{}{
			".chezmoiignore":  "README.md\n",
			".chezmoiremove":  "*.txt\n",
			".chezmoiversion": "1.2.3\n",
			".chezmoitemplates": map[string]interface{}{
				"template": "# contents of .chezmoitemplates/template\n",
			},
			"README.md": "",
			"dot_dir": map[string]interface{}{
				"file": "# contents of .dir/file\n",
			},
			"run_script":      "# contents of script\n",
			"symlink_symlink": ".dir/subdir/file\n",
		},
	}, func(fs vfs.FS) {
		s := NewSourceState(
			WithSourceDir("/home/user/.local/share/chezmoi"),
			WithSystem(NewRealSystem(fs)),
		)
		require.NoError(t, s.Read())
		require.NoError(t, s.evaluateAll())

		dumpSystem := NewDumpSystem()
		persistentState := NewMockPersistentState()
		require.NoError(t, s.applyAll(dumpSystem, persistentState, "", ApplyOptions{}))
		expectedData := map[AbsPath]interface{}{
			".dir": &dirData{
				Type: dataTypeDir,
				Name: ".dir",
				Perm: 0o777,
			},
			".dir/file": &fileData{
				Type:     dataTypeFile,
				Name:     ".dir/file",
				Contents: "# contents of .dir/file\n",
				Perm:     0o666,
			},
			"script": &scriptData{
				Type:     dataTypeScript,
				Name:     "script",
				Contents: "# contents of script\n",
			},
			"symlink": &symlinkData{
				Type:     dataTypeSymlink,
				Name:     "symlink",
				Linkname: ".dir/subdir/file",
			},
		}
		actualData := dumpSystem.Data()
		assert.Equal(t, expectedData, actualData)
	})
}

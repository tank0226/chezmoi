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
				"foo": "bar",
			},
			"README.md": "",
			"dir": map[string]interface{}{
				"foo": "bar",
			},
			"run_script":      "#!/bin/sh\n",
			"symlink_symlink": "bar",
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
		expectedData := map[string]interface{}{
			"dir": &dirData{
				Type: dataTypeDir,
				Name: "dir",
				Perm: 0o777,
			},
			"dir/foo": &fileData{
				Type:     dataTypeFile,
				Name:     "dir/foo",
				Contents: "bar",
				Perm:     0o666,
			},
			"script": &scriptData{
				Type:     dataTypeScript,
				Name:     "script",
				Contents: "#!/bin/sh\n",
			},
			"symlink": &symlinkData{
				Type:     dataTypeSymlink,
				Name:     "symlink",
				Linkname: "bar",
			},
		}
		actualData := dumpSystem.Data()
		assert.Equal(t, expectedData, actualData)
	})
}

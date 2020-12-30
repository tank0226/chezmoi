package chezmoi

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vfs "github.com/twpayne/go-vfs"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoitest"
)

var _ System = &ZIPSystem{}

func TestZIPSystem(t *testing.T) {
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
			"symlink_symlink": "target",
		},
	}, func(fs vfs.FS) {
		s := NewSourceState(
			WithSourceDir("/home/user/.local/share/chezmoi"),
			WithSystem(NewRealSystem(fs)),
		)
		require.NoError(t, s.Read())
		require.NoError(t, s.evaluateAll())

		b := &bytes.Buffer{}
		zipSystem := NewZIPSystem(b)
		persistentState := NewMockPersistentState()
		require.NoError(t, s.applyAll(zipSystem, persistentState, "", ApplyOptions{}))
		require.NoError(t, zipSystem.Close())

		r, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
		require.NoError(t, err)
		expectedFiles := []struct {
			name     string
			method   uint16
			mode     os.FileMode
			contents []byte
		}{
			{
				name: "dir",
				mode: os.ModeDir | 0o777,
			},
			{
				name:     "dir/foo",
				method:   zip.Deflate,
				mode:     0o666,
				contents: []byte("bar"),
			},
			{
				name:     "script",
				method:   zip.Deflate,
				mode:     0o700,
				contents: []byte("#!/bin/sh\n"),
			},
			{
				name:     "symlink",
				mode:     os.ModeSymlink,
				contents: []byte("target"),
			},
		}
		require.Len(t, r.File, len(expectedFiles))
		for i, expectedFile := range expectedFiles {
			t.Run(expectedFile.name, func(t *testing.T) {
				actualFile := r.File[i]
				assert.Equal(t, expectedFile.name, actualFile.Name)
				assert.Equal(t, expectedFile.method, actualFile.Method)
				assert.Equal(t, expectedFile.mode, actualFile.Mode())
				if expectedFile.contents != nil {
					rc, err := actualFile.Open()
					require.NoError(t, err)
					actualContents, err := ioutil.ReadAll(rc)
					require.NoError(t, err)
					assert.Equal(t, expectedFile.contents, actualContents)
				}
			})
		}
	})
}

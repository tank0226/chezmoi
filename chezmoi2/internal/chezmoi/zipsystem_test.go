package chezmoi

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"

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

		b := &bytes.Buffer{}
		zipSystem := NewZIPSystem(b, time.Now().UTC())
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
				name: ".dir",
				mode: os.ModeDir | 0o777,
			},
			{
				name:     ".dir/file",
				method:   zip.Deflate,
				mode:     0o666,
				contents: []byte("# contents of .dir/file\n"),
			},
			{
				name:     "script",
				method:   zip.Deflate,
				mode:     0o700,
				contents: []byte("# contents of script\n"),
			},
			{
				name:     "symlink",
				mode:     os.ModeSymlink,
				contents: []byte(".dir/subdir/file"),
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

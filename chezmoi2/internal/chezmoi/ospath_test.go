package chezmoi

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoitest"
)

func TestOSPathFormat(t *testing.T) {
	type s struct {
		Dir *OSPath
	}

	for name, format := range Formats {
		t.Run(name, func(t *testing.T) {
			var dirStr string
			switch runtime.GOOS {
			case "windows":
				dirStr = `C:\home\user`
			default:
				dirStr = "/home/user"
			}
			expectedS := &s{
				Dir: NewOSPath(dirStr),
			}
			data, err := format.Marshal(expectedS)
			assert.NoError(t, err)
			actualS := &s{}
			assert.NoError(t, format.Decode(data, actualS))
			assert.Equal(t, expectedS, actualS)
		})
	}
}

func TestOSPathTildeAbsSlash(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	normalizedWD, err := NormalizePath(wd)
	require.NoError(t, err)

	for _, tc := range []struct {
		name     string
		s        string
		expected string
	}{
		{
			name:     "empty",
			expected: normalizedWD,
		},
		{
			name:     "file",
			s:        "file",
			expected: path.Join(normalizedWD, "file"),
		},
		{
			name:     "tilde",
			s:        "~",
			expected: chezmoitest.NormalizedHomeDir(),
		},
		{
			name:     "tilde_home_file",
			s:        "~/file",
			expected: chezmoitest.NormalizedHomeDir() + "/file",
		},
		{
			name:     "tilde_home_file_windows",
			s:        `~\file`,
			expected: chezmoitest.NormalizedHomeDir() + "/file",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chezmoitest.SkipUnlessGOOS(t, tc.name)

			actual, err := NewOSPath(tc.s).Normalize(chezmoitest.NormalizedHomeDir())
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

package chezmoi

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoitest"
)

func TestOSPathFormat(t *testing.T) {
	type s struct {
		Dir OSPath
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
	wdAbsPath, err := NormalizePath(wd)
	require.NoError(t, err)

	for _, tc := range []struct {
		name     string
		s        string
		expected AbsPath
	}{
		{
			name:     "empty",
			expected: wdAbsPath,
		},
		{
			name:     "file",
			s:        "file",
			expected: wdAbsPath.Join("file"),
		},
		{
			name:     "tilde",
			s:        "~",
			expected: AbsPath(chezmoitest.HomeDir()),
		},
		{
			name:     "tilde_home_file",
			s:        "~/file",
			expected: AbsPath(chezmoitest.HomeDir()) + "/file",
		},
		{
			name:     "tilde_home_file_windows",
			s:        `~\file`,
			expected: AbsPath(chezmoitest.HomeDir()) + "/file",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chezmoitest.SkipUnlessGOOS(t, tc.name)

			actual, err := NewOSPath(tc.s).Normalize(AbsPath(chezmoitest.HomeDir()))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

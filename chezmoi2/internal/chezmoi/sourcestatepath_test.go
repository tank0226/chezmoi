package chezmoi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSourceStatePathRelPath(t *testing.T) {
	for _, tc := range []struct {
		name            string
		sourceStatePath SourceStatePath
		expectedDirPath SourceStatePath
		expectedRelPath RelPath
	}{
		{
			name:            "empty",
			expectedDirPath: NewSourceStateDirPath("."),
		},
		{
			name:            "dir",
			sourceStatePath: NewSourceStateDirPath("dir"),
			expectedDirPath: NewSourceStateDirPath("."),
			expectedRelPath: "dir",
		},
		{
			name:            "exact_dir",
			sourceStatePath: NewSourceStateDirPath("exact_dir"),
			expectedDirPath: NewSourceStateDirPath("."),
			expectedRelPath: "dir",
		},
		{
			name:            "exact_dir_private_dir",
			sourceStatePath: NewSourceStateDirPath("exact_dir/private_dir"),
			expectedDirPath: NewSourceStateDirPath("exact_dir"),
			expectedRelPath: "dir/dir",
		},
		{
			name:            "file",
			sourceStatePath: NewSourceStatePath("file"),
			expectedDirPath: NewSourceStateDirPath("."),
			expectedRelPath: "file",
		},
		{
			name:            "dot_file",
			sourceStatePath: NewSourceStatePath("dot_file"),
			expectedDirPath: NewSourceStateDirPath("."),
			expectedRelPath: ".file",
		},
		{
			name:            "exact_dir_executable_file",
			sourceStatePath: NewSourceStatePath("exact_dir/executable_file"),
			expectedDirPath: NewSourceStateDirPath("exact_dir"),
			expectedRelPath: "dir/file",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedDirPath, tc.sourceStatePath.Dir())
			assert.Equal(t, tc.expectedRelPath, tc.sourceStatePath.RelPath())
		})
	}
}

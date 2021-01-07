package cmd

import (
	"io"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vfs "github.com/twpayne/go-vfs"
	xdg "github.com/twpayne/go-xdg/v3"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoi"
	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoitest"
)

func TestAddTemplateFuncPanic(t *testing.T) {
	chezmoitest.WithTestFS(t, nil, func(fs vfs.FS) {
		c := newTestConfig(t, fs)
		assert.NotPanics(t, func() {
			c.addTemplateFunc("func", nil)
		})
		assert.Panics(t, func() {
			c.addTemplateFunc("func", nil)
		})
	})
}

func TestParseConfig(t *testing.T) {
	for _, tc := range []struct {
		name          string
		filename      string
		contents      string
		expectedColor bool
	}{
		{
			name:     "json_bool",
			filename: "chezmoi.json",
			contents: chezmoitest.JoinLines(
				`{`,
				`  "color":true`,
				`}`,
			),
			expectedColor: true,
		},
		{
			name:     "json_string",
			filename: "chezmoi.json",
			contents: chezmoitest.JoinLines(
				`{`,
				`  "color":"on"`,
				`}`,
			),
			expectedColor: true,
		},
		{
			name:     "toml_bool",
			filename: "chezmoi.toml",
			contents: chezmoitest.JoinLines(
				`color = true`,
			),
			expectedColor: true,
		},
		{
			name:     "toml_string",
			filename: "chezmoi.toml",
			contents: chezmoitest.JoinLines(
				`color = "y"`,
			),
			expectedColor: true,
		},
		{
			name:     "yaml_bool",
			filename: "chezmoi.yaml",
			contents: chezmoitest.JoinLines(
				`color: true`,
			),
			expectedColor: true,
		},
		{
			name:     "yaml_string",
			filename: "chezmoi.yaml",
			contents: chezmoitest.JoinLines(
				`color: "yes"`,
			),
			expectedColor: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chezmoitest.WithTestFS(t, map[string]interface{}{
				"/home/user/.config/chezmoi/" + tc.filename: tc.contents,
			}, func(fs vfs.FS) {
				c := newTestConfig(t, fs)
				require.NoError(t, c.execute([]string{"init"}))
				assert.Equal(t, tc.expectedColor, c.color)
			})
		})
	}
}

func TestUpperSnakeCaseToCamelCase(t *testing.T) {
	for s, expected := range map[string]string{
		"BUG_REPORT_URL":   "bugReportURL",
		"ID":               "id",
		"ID_LIKE":          "idLike",
		"NAME":             "name",
		"VERSION_CODENAME": "versionCodename",
		"VERSION_ID":       "versionID",
	} {
		assert.Equal(t, expected, upperSnakeCaseToCamelCase(s))
	}
}

func TestValidateKeys(t *testing.T) {
	for _, tc := range []struct {
		data        interface{}
		expectedErr bool
	}{
		{
			data:        nil,
			expectedErr: false,
		},
		{
			data: map[string]interface{}{
				"foo":                    "bar",
				"a":                      0,
				"_x9":                    false,
				"ThisVariableIsExported": nil,
				"αβ":                     "",
			},
			expectedErr: false,
		},
		{
			data: map[string]interface{}{
				"foo-foo": "bar",
			},
			expectedErr: true,
		},
		{
			data: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar-bar": "baz",
				},
			},
			expectedErr: true,
		},
		{
			data: map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"bar-bar": "baz",
					},
				},
			},
			expectedErr: true,
		},
	} {
		if tc.expectedErr {
			assert.Error(t, validateKeys(tc.data, identifierRx))
		} else {
			assert.NoError(t, validateKeys(tc.data, identifierRx))
		}
	}
}

func newTestConfig(t *testing.T, fs vfs.FS, options ...configOption) *Config {
	t.Helper()
	system := chezmoi.NewRealSystem(fs)
	c, err := newConfig(
		append([]configOption{
			withBaseSystem(system),
			withDestSystem(system),
			withSourceSystem(system),
			withTestFS(fs),
			withTestUser("user"),
		}, options...)...,
	)
	require.NoError(t, err)
	return c
}

func withBaseSystem(baseSystem chezmoi.System) configOption {
	return func(c *Config) error {
		c.baseSystem = baseSystem
		return nil
	}
}

func withDestSystem(destSystem chezmoi.System) configOption {
	return func(c *Config) error {
		c.destSystem = destSystem
		return nil
	}
}

func withSourceSystem(sourceSystem chezmoi.System) configOption {
	return func(c *Config) error {
		c.sourceSystem = sourceSystem
		return nil
	}
}

func withStdout(stdout io.Writer) configOption {
	return func(c *Config) error {
		c.stdout = stdout
		return nil
	}
}

func withTestFS(fs vfs.FS) configOption {
	return func(c *Config) error {
		c.fs = fs
		return nil
	}
}

func withTestUser(username string) configOption {
	return func(c *Config) error {
		var homeDirStr string
		switch runtime.GOOS {
		case "windows":
			homeDirStr = `c:\home\user`
		default:
			homeDirStr = "/home/user"
		}
		c.HomeDir = homeDirStr
		c.SourceDir = filepath.Join(homeDirStr, ".local", "share", "chezmoi")
		c.DestDir = homeDirStr
		c.Umask = 0o22
		configHome := filepath.Join(homeDirStr, ".config")
		dataHome := filepath.Join(homeDirStr, ".local", "share")
		c.bds = &xdg.BaseDirectorySpecification{
			ConfigHome: configHome,
			ConfigDirs: []string{configHome},
			DataHome:   dataHome,
			DataDirs:   []string{dataHome},
			CacheHome:  filepath.Join(homeDirStr, ".cache"),
			RuntimeDir: filepath.Join(homeDirStr, ".run"),
		}
		return nil
	}
}

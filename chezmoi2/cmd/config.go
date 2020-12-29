package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/coreos/go-semver/semver"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/twpayne/go-vfs"
	vfsafero "github.com/twpayne/go-vfsafero"
	"github.com/twpayne/go-xdg/v3"
	"golang.org/x/term"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoi"
	"github.com/twpayne/chezmoi/internal/git"
)

type purgeOptions struct {
	binary bool
}

type templateConfig struct {
	Options []string `mapstructure:"options"`
}

// A Config represents a configuration.
type Config struct {
	version     *semver.Version
	versionInfo VersionInfo
	versionStr  string

	logger zerolog.Logger

	bds *xdg.BaseDirectorySpecification

	fs              vfs.FS
	configFile      string
	baseSystem      chezmoi.System
	sourceSystem    chezmoi.System
	destSystem      chezmoi.System
	persistentState chezmoi.PersistentState
	color           bool

	// Global configuration, settable in the config file.
	HomeDir       string                 `mapstructure:"homeDir"`
	SourceDir     string                 `mapstructure:"sourceDir"`
	DestDir       string                 `mapstructure:"destDir"`
	Umask         fileMode               `mapstructure:"umask"`
	Format        string                 `mapstructure:"format"`
	Remove        bool                   `mapstructure:"remove"`
	Color         string                 `mapstructure:"color"`
	Data          map[string]interface{} `mapstructure:"data"`
	Template      templateConfig         `mapstructure:"template"`
	UseBuiltinGit string                 `mapstructure:"useBuiltinGit"`

	// Global configuration, not settable in the config file.
	debug         bool
	dryRun        bool
	force         bool
	keepGoing     bool
	output        string
	verbose       bool
	templateFuncs template.FuncMap

	// Password manager configurations, settable in the config file.
	Bitwarden   bitwardenConfig   `mapstructure:"bitwarden"`
	Gopass      gopassConfig      `mapstructure:"gopass"`
	Keepassxc   keepassxcConfig   `mapstructure:"keepassxc"`
	Lastpass    lastpassConfig    `mapstructure:"lastpass"`
	Onepassword onepasswordConfig `mapstructure:"onepassword"`
	Pass        passConfig        `mapstructure:"pass"`
	Secret      secretConfig      `mapstructure:"secret"`
	Vault       vaultConfig       `mapstructure:"vault"`

	// Password manager data.
	gitHub  gitHubData
	keyring keyringData

	// Command configurations, settable in the config file.
	CD    cdCmdConfig    `mapstructure:"cd"`
	Diff  diffCmdConfig  `mapstructure:"diff"`
	Edit  editCmdConfig  `mapstructure:"edit"`
	Git   gitCmdConfig   `mapstructure:"git"`
	Merge mergeCmdConfig `mapstructure:"merge"`

	// Command configurations, not settable in the config file.
	add             addCmdConfig
	apply           applyCmdConfig
	archive         archiveCmdConfig
	dump            dumpCmdConfig
	executeTemplate executeTemplateCmdConfig
	init            initCmdConfig
	managed         managedCmdConfig
	purge           purgeCmdConfig
	status          statusCmdConfig
	update          updateCmdConfig
	verify          verifyCmdConfig

	// Computed configuration.
	normalizedConfigFile string
	normalizedHomeDir    string
	normalizedSourceDir  string
	normalizedDestDir    string

	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	tty       io.ReadWriter
	ttyReader *bufio.Reader
	ttyWriter io.Writer

	ioregData ioregData
}

// A configOption sets and option on a Config.
type configOption func(*Config) error

var (
	persistentStateFilename    = "chezmoistate.boltdb"
	commitMessageTemplateAsset = "assets/templates/COMMIT_MESSAGE.tmpl"

	identifierRx = regexp.MustCompile(`\A[\pL_][\pL\p{Nd}_]*\z`)
	whitespaceRx = regexp.MustCompile(`\s+`)

	assets = make(map[string][]byte)
)

// newConfig creates a new Config with the given options.
func newConfig(options ...configOption) (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	normalizedHomeDir, err := chezmoi.NormalizePath(homeDir)
	if err != nil {
		return nil, err
	}

	bds, err := xdg.NewBaseDirectorySpecification()
	if err != nil {
		return nil, err
	}

	c := &Config{
		bds:     bds,
		fs:      vfs.OSFS,
		HomeDir: homeDir,
		DestDir: homeDir,
		Umask:   fileMode(chezmoi.GetUmask()),
		Color:   "auto",
		Format:  "json",
		Diff: diffCmdConfig{
			include: chezmoi.NewIncludeSet(chezmoi.IncludeAll &^ chezmoi.IncludeScripts),
		},
		Edit: editCmdConfig{
			include: chezmoi.NewIncludeSet(chezmoi.IncludeDirs | chezmoi.IncludeFiles | chezmoi.IncludeSymlinks),
		},
		Git: gitCmdConfig{
			Command: "git",
		},
		Merge: mergeCmdConfig{
			Command: "vimdiff",
		},
		Template: templateConfig{
			Options: chezmoi.DefaultTemplateOptions,
		},
		templateFuncs: sprig.TxtFuncMap(),
		Bitwarden: bitwardenConfig{
			Command: "bw",
		},
		Gopass: gopassConfig{
			Command: "gopass",
		},
		Keepassxc: keepassxcConfig{
			Command: "keepassxc-cli",
		},
		Lastpass: lastpassConfig{
			Command: "lpass",
		},
		Onepassword: onepasswordConfig{
			Command: "op",
		},
		Pass: passConfig{
			Command: "pass",
		},
		Vault: vaultConfig{
			Command: "vault",
		},
		add: addCmdConfig{
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive: true,
		},
		apply: applyCmdConfig{
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive: true,
		},
		archive: archiveCmdConfig{
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive: true,
		},
		dump: dumpCmdConfig{
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive: true,
		},
		managed: managedCmdConfig{
			include: chezmoi.NewIncludeSet(chezmoi.IncludeDirs | chezmoi.IncludeFiles | chezmoi.IncludeSymlinks),
		},
		status: statusCmdConfig{
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive: true,
		},
		update: updateCmdConfig{
			apply:     true,
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive: true,
		},
		verify: verifyCmdConfig{
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll &^ chezmoi.IncludeScripts),
			recursive: true,
		},

		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,

		normalizedHomeDir: normalizedHomeDir,
	}

	for key, value := range map[string]interface{}{
		"bitwarden":                c.bitwardenTemplateFunc,
		"bitwardenFields":          c.bitwardenFieldsTemplateFunc,
		"gitHubKeys":               c.gitHubKeysTemplateFunc,
		"gopass":                   c.gopassTemplateFunc,
		"include":                  c.includeTemplateFunc,
		"ioreg":                    c.ioregTemplateFunc,
		"joinPath":                 c.joinPathTemplateFunc,
		"keepassxc":                c.keepassxcTemplateFunc,
		"keepassxcAttribute":       c.keepassxcAttributeTemplateFunc,
		"keyring":                  c.keyringTemplateFunc,
		"lastpass":                 c.lastpassTemplateFunc,
		"lastpassRaw":              c.lastpassRawTemplateFunc,
		"lookPath":                 c.lookPathTemplateFunc,
		"onepassword":              c.onepasswordTemplateFunc,
		"onepasswordDetailsFields": c.onepasswordDetailsFieldsTemplateFunc,
		"onepasswordDocument":      c.onepasswordDocumentTemplateFunc,
		"pass":                     c.passTemplateFunc,
		"secret":                   c.secretTemplateFunc,
		"secretJSON":               c.secretJSONTemplateFunc,
		"stat":                     c.statTemplateFunc,
		"vault":                    c.vaultTemplateFunc,
	} {
		c.addTemplateFunc(key, value)
	}

	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	c.configFile = defaultConfigFile(c.fs, c.bds).String()
	c.SourceDir = defaultSourceDir(c.fs, c.bds).String()

	c.normalizedHomeDir, err = chezmoi.NormalizePath(c.HomeDir)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) addTemplateFunc(key string, value interface{}) {
	if _, ok := c.templateFuncs[key]; ok {
		panic(fmt.Sprintf("%s: already defined", key))
	}
	c.templateFuncs[key] = value
}

func (c *Config) applyArgs(targetSystem chezmoi.System, targetDir string, args []string, include *chezmoi.IncludeSet, recursive bool, umask os.FileMode, preApplyFunc chezmoi.PreApplyFunc) error {
	s, err := c.sourceState()
	if err != nil {
		return err
	}

	applyOptions := chezmoi.ApplyOptions{
		Include:      include,
		PreApplyFunc: preApplyFunc,
		Umask:        umask,
	}

	var targetNames []string
	if len(args) == 0 {
		targetNames = s.AllTargetNames()
	} else {
		targetNames, err = c.targetNames(s, args, targetNamesOptions{
			mustBeInSourceState: true,
			recursive:           recursive,
		})
		if err != nil {
			return err
		}
	}

	for _, targetName := range targetNames {
		switch err := s.Apply(targetSystem, c.persistentState, targetDir, targetName, applyOptions); {
		case errors.Is(err, chezmoi.Skip):
			continue
		case err != nil && c.keepGoing:
			c.errorf("%v", err)
		case err != nil:
			return err
		}
	}

	return nil
}

func (c *Config) cmdOutput(dir, name string, args []string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		var err error
		cmd.Dir, err = c.baseSystem.RawPath(dir)
		if err != nil {
			return nil, err
		}
	}
	return c.baseSystem.IdempotentCmdOutput(cmd)
}

func (c *Config) defaultTemplateData() map[string]interface{} {
	data := map[string]interface{}{
		"arch":      runtime.GOARCH,
		"homeDir":   c.HomeDir,
		"os":        runtime.GOOS,
		"sourceDir": c.normalizedSourceDir,
		"version": map[string]interface{}{
			"builtBy": c.versionInfo.BuiltBy,
			"commit":  c.versionInfo.Commit,
			"date":    c.versionInfo.Date,
			"version": c.versionInfo.Version,
		},
	}

	// Determine the user's username and group, if possible.
	//
	// user.Current and user.LookupGroupId in Go's standard library are
	// generally unreliable, so work around errors if possible, or ignore them.
	//
	// If CGO is disabled, then the Go standard library falls back to parsing
	// /etc/passwd and /etc/group, which will return incorrect results without
	// error if the system uses an alternative password database such as NIS or
	// LDAP.
	//
	// If CGO is enabled then user.Current and user.LookupGroupId will use the
	// underlying libc functions, namely getpwuid_r and getgrnam_r. If linked
	// with glibc this will return the correct result. If linked with musl then
	// they will use musl's implementation which, like Go's non-CGO
	// implementation, also only parses /etc/passwd and /etc/group and so also
	// returns incorrect results without error if NIS or LDAP are being used.
	//
	// Since neither the username nor the group are likely widely used in
	// templates, leave these variables unset if their values cannot be
	// determined. Unset variables will trigger template errors if used,
	// alerting the user to the problem and allowing them to find alternative
	// solutions.
	if currentUser, err := user.Current(); err == nil {
		data["username"] = currentUser.Username
		if group, err := user.LookupGroupId(currentUser.Gid); err == nil {
			data["group"] = group.Name
		} else {
			c.logger.Debug().
				Str("gid", currentUser.Gid).
				Err(err).
				Msg("user.LookupGroupId")
		}
	} else {
		c.logger.Debug().
			Err(err).
			Msg("user.Current")
		user, ok := os.LookupEnv("USER")
		if ok {
			data["username"] = user
		} else {
			c.logger.Debug().
				Bool("ok", ok).
				Msg("os.LookupEnv")
		}
	}

	if hostname, err := os.Hostname(); err == nil {
		data["fullHostname"] = hostname
		data["hostname"] = strings.SplitN(hostname, ".", 2)[0]
	} else {
		c.logger.Debug().
			Err(err).
			Msg("os.Hostname")
	}

	if kernelInfo, err := chezmoi.KernelInfo(c.fs); err == nil {
		data["kernel"] = kernelInfo
	} else {
		c.logger.Debug().
			Err(err).
			Msg("chezmoi.KernelInfo(...)")
	}

	if osRelease, err := chezmoi.OSRelease(c.fs); err == nil {
		data["osRelease"] = upperSnakeCaseToCamelCaseMap(osRelease)
	} else {
		c.logger.Debug().
			Err(err).
			Msg("chezmoi.OSRelease(...)")
	}

	return map[string]interface{}{
		"chezmoi": data,
	}
}

func (c *Config) destPathInfos(sourceState *chezmoi.SourceState, args []string, recursive bool) (map[string]os.FileInfo, error) {
	destPathInfos := make(map[string]os.FileInfo)
	for _, arg := range args {
		destPath, err := c.normalizedDestPath(chezmoi.NewOSPath(arg))
		if err != nil {
			return nil, err
		}
		if recursive {
			if err := vfs.WalkSlash(c.destSystem, destPath, func(destPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				return sourceState.AddDestPathInfos(destPathInfos, c.destSystem, destPath, info)
			}); err != nil {
				return nil, err
			}
		} else {
			if err := sourceState.AddDestPathInfos(destPathInfos, c.destSystem, destPath, nil); err != nil {
				return nil, err
			}
		}
	}
	return destPathInfos, nil
}

func (c *Config) doPurge(purgeOptions *purgeOptions) error {
	if c.persistentState != nil {
		if err := c.persistentState.Close(); err != nil {
			return err
		}
	}

	absSlashPersistentStateFile, err := c.persistentStateFile().Normalize(c.normalizedHomeDir)
	if err != nil {
		return err
	}

	paths := []string{
		path.Dir(c.normalizedConfigFile),
		c.normalizedConfigFile,
		absSlashPersistentStateFile,
		c.normalizedSourceDir,
	}
	if purgeOptions != nil && purgeOptions.binary {
		executable, err := os.Executable()
		if err == nil {
			paths = append(paths, executable)
		}
	}

	// Remove all paths that exist.
	for _, path := range paths {
		switch _, err := c.baseSystem.Stat(path); {
		case os.IsNotExist(err):
			continue
		case err != nil:
			return err
		}

		if !c.force {
			switch choice, err := c.prompt(fmt.Sprintf("Remove %s", path), "ynqa"); {
			case err != nil:
				return err
			case choice == 'a':
				c.force = true
			case choice == 'n':
				continue
			case choice == 'q':
				return nil
			}
		}

		switch err := c.baseSystem.RemoveAll(path); {
		case os.IsPermission(err):
			continue
		case err != nil:
			return err
		}
	}

	return nil
}

// editor returns the path to the user's editor and any extra arguments.
func (c *Config) editor() (string, []string) {
	// If the user has set and edit command then use it.
	if c.Edit.Command != "" {
		return c.Edit.Command, c.Edit.Args
	}

	// Prefer $VISUAL over $EDITOR and fallback to vi.
	editor := firstNonEmptyString(
		os.Getenv("VISUAL"),
		os.Getenv("EDITOR"),
		"vi",
	)

	// If editor is found, return it.
	if path, err := exec.LookPath(editor); err == nil {
		return path, nil
	}

	// Otherwise, if editor contains spaces, then assume that the first word is
	// the editor and the rest are arguments.
	components := whitespaceRx.Split(editor, -1)
	if len(components) > 1 {
		if path, err := exec.LookPath(components[0]); err == nil {
			return path, components[1:]
		}
	}

	// Fallback to editor only.
	return editor, nil
}

func (c *Config) errorf(format string, args ...interface{}) {
	fmt.Fprintf(c.stderr, "chezmoi: "+format, args...)
}

func (c *Config) execute(args []string) error {
	rootCmd, err := c.newRootCmd()
	if err != nil {
		return err
	}
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func (c *Config) getTargetName(arg *chezmoi.OSPath) (string, error) {
	destPath, err := c.normalizedDestPath(arg)
	if err != nil {
		return "", err
	}
	return chezmoi.TrimDirPrefix(destPath, c.normalizedDestDir)
}

func (c *Config) getTTY() (*bufio.Reader, io.Writer, error) {
	if c.ttyReader == nil {
		// FIXME find out how to get a tty on Windows
		if runtime.GOOS == "windows" {
			c.ttyReader = bufio.NewReader(c.stdin)
			c.ttyWriter = c.stdout
		} else {
			var err error
			c.tty, err = os.OpenFile("/dev/tty", os.O_RDWR, 0)
			if err != nil {
				return nil, nil, err
			}
			c.ttyReader = bufio.NewReader(c.tty)
			c.ttyWriter = c.tty
		}
	}
	return c.ttyReader, c.ttyWriter, nil
}

func (c *Config) gitAutoAdd() (*git.Status, error) {
	if err := c.run(c.normalizedSourceDir, c.Git.Command, []string{"add", "."}); err != nil {
		return nil, err
	}
	output, err := c.cmdOutput(c.normalizedSourceDir, c.Git.Command, []string{"status", "--porcelain=v2"})
	if err != nil {
		return nil, err
	}
	return git.ParseStatusPorcelainV2(output)
}

func (c *Config) gitAutoCommit(status *git.Status) error {
	if status.Empty() {
		return nil
	}
	commitMessageText, err := asset(commitMessageTemplateAsset)
	if err != nil {
		return err
	}
	commitMessageTmpl, err := template.New("commit_message").Funcs(c.templateFuncs).Parse(string(commitMessageText))
	if err != nil {
		return err
	}
	commitMessage := strings.Builder{}
	if err := commitMessageTmpl.Execute(&commitMessage, status); err != nil {
		return err
	}
	return c.run(c.normalizedSourceDir, c.Git.Command, []string{"commit", "--message", commitMessage.String()})
}

func (c *Config) gitAutoPush(status *git.Status) error {
	if status.Empty() {
		return nil
	}
	return c.run(c.normalizedSourceDir, c.Git.Command, []string{"push"})
}

func (c *Config) makeRunEWithSourceState(runE func(*cobra.Command, []string, *chezmoi.SourceState) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		sourceState, err := c.sourceState()
		if err != nil {
			return err
		}
		return runE(cmd, args, sourceState)
	}
}

func (c *Config) marshal(data interface{}) error {
	format, ok := chezmoi.Formats[c.Format]
	if !ok {
		return fmt.Errorf("%s: unknown format", c.Format)
	}
	marshaledData, err := format.Marshal(data)
	if err != nil {
		return err
	}
	return c.writeOutput(marshaledData)
}

func (c *Config) newRootCmd() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:                "chezmoi",
		Short:              "Manage your dotfiles across multiple diverse machines, securely",
		Version:            c.versionStr,
		PersistentPreRunE:  c.persistentPreRunRootE,
		PersistentPostRunE: c.persistentPostRunRootE,
		SilenceErrors:      true,
		SilenceUsage:       true,
	}

	persistentFlags := rootCmd.PersistentFlags()

	persistentFlags.StringVar(&c.Color, "color", c.Color, "colorize diffs")
	persistentFlags.StringVarP(&c.DestDir, "destination", "D", c.DestDir, "destination directory")
	persistentFlags.StringVar(&c.Format, "format", c.Format, "format ("+serializationFormatNamesStr()+")")
	persistentFlags.BoolVar(&c.Remove, "remove", c.Remove, "remove targets")
	persistentFlags.StringVarP(&c.SourceDir, "source", "S", c.SourceDir, "source directory")
	persistentFlags.StringVar(&c.UseBuiltinGit, "use-builtin-git", c.UseBuiltinGit, "use builtin git")
	for _, key := range []string{
		"color",
		"destination",
		"format",
		"remove",
		"source",
	} {
		if err := viper.BindPFlag(key, persistentFlags.Lookup(key)); err != nil {
			return nil, err
		}
	}

	persistentFlags.StringVarP(&c.configFile, "config", "c", c.configFile, "config file")
	persistentFlags.BoolVarP(&c.dryRun, "dry-run", "n", c.dryRun, "dry run")
	persistentFlags.BoolVar(&c.force, "force", c.force, "force")
	persistentFlags.BoolVarP(&c.keepGoing, "keep-going", "k", c.keepGoing, "keep going as far as possible after an error")
	persistentFlags.BoolVarP(&c.verbose, "verbose", "v", c.verbose, "verbose")
	persistentFlags.StringVarP(&c.output, "output", "o", c.output, "output file")
	persistentFlags.BoolVar(&c.debug, "debug", c.debug, "write debug logs")

	for _, err := range []error{
		rootCmd.MarkPersistentFlagFilename("config"),
		rootCmd.MarkPersistentFlagDirname("destination"),
		rootCmd.MarkPersistentFlagFilename("output"),
		rootCmd.MarkPersistentFlagDirname("source"),
	} {
		if err != nil {
			return nil, err
		}
	}

	rootCmd.SetHelpCommand(c.newHelpCmd())
	for _, newCmdFunc := range []func() *cobra.Command{
		c.newAddCmd,
		c.newApplyCmd,
		c.newArchiveCmd,
		c.newCatCmd,
		c.newCDCmd,
		c.newChattrCmd,
		c.newCompletionCmd,
		c.newDataCmd,
		c.newDiffCmd,
		c.newDocsCmd,
		c.newDoctorCmd,
		c.newDumpCmd,
		c.newEditCmd,
		c.newEditConfigCmd,
		c.newExecuteTemplateCmd,
		c.newForgetCmd,
		c.newGitCmd,
		// c.newImportCmd, // FIXME
		c.newInitCmd,
		c.newManagedCmd,
		c.newMergeCmd,
		c.newPurgeCmd,
		c.newRemoveCmd,
		c.newSourcePathCmd,
		c.newStateCmd,
		c.newStatusCmd,
		c.newUnmanagedCmd,
		c.newUpdateCmd,
		c.newVerifyCmd,
	} {
		rootCmd.AddCommand(newCmdFunc())
	}

	return rootCmd, nil
}

func (c *Config) normalizedDestPath(arg *chezmoi.OSPath) (string, error) {
	normalizedPath, err := arg.Normalize(c.normalizedHomeDir)
	if err != nil {
		return "", err
	}
	if _, err := chezmoi.TrimDirPrefix(normalizedPath, c.normalizedDestDir); err != nil {
		return "", fmt.Errorf("%s: not in destination directory (%s)", arg, c.normalizedDestDir)
	}
	return normalizedPath, nil
}

func (c *Config) persistentPostRunRootE(cmd *cobra.Command, args []string) error {
	if c.persistentState != nil {
		if err := c.persistentState.Close(); err != nil {
			return err
		}
	}

	if boolAnnotation(cmd, modifiesConfigFile) {
		// Warn the user of any errors reading the config file.
		v := viper.New()
		v.SetFs(vfsafero.NewAferoFS(c.fs))
		v.SetConfigFile(c.normalizedConfigFile)
		err := v.ReadInConfig()
		if err == nil {
			err = v.Unmarshal(&Config{})
		}
		if err != nil {
			cmd.Printf("warning: %s: %v\n", c.normalizedConfigFile, err)
		}
	}

	if boolAnnotation(cmd, modifiesSourceDirectory) {
		var status *git.Status
		if c.Git.AutoAdd || c.Git.AutoCommit || c.Git.AutoPush {
			var err error
			status, err = c.gitAutoAdd()
			if err != nil {
				return err
			}
		}
		if c.Git.AutoCommit || c.Git.AutoPush {
			if err := c.gitAutoCommit(status); err != nil {
				return err
			}
		}
		if c.Git.AutoPush {
			if err := c.gitAutoPush(status); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Config) persistentPreRunRootE(cmd *cobra.Command, args []string) error {
	var err error
	c.normalizedConfigFile, err = chezmoi.NewOSPath(c.configFile).Normalize(c.normalizedHomeDir)
	if err != nil {
		return err
	}

	if err := c.readConfig(); err != nil {
		if !boolAnnotation(cmd, doesNotRequireValidConfig) {
			return fmt.Errorf("invalid config: %s: %w", c.configFile, err)
		}
		cmd.Printf("warning: %s: %v\n", c.configFile, err)
	}

	if c.Color == "" || strings.ToLower(c.Color) == "auto" {
		if _, ok := os.LookupEnv("NO_COLOR"); ok {
			c.color = false
		} else if stdout, ok := c.stdout.(*os.File); ok {
			c.color = term.IsTerminal(int(stdout.Fd()))
		} else {
			c.color = false
		}
	} else if color, err := parseBool(c.Color); err == nil {
		c.color = color
	} else if !boolAnnotation(cmd, doesNotRequireValidConfig) {
		return fmt.Errorf("%s: invalid color value", c.Color)
	}

	if c.color {
		if err := enableVirtualTerminalProcessing(c.stdout); err != nil {
			return err
		}
	}

	if c.normalizedSourceDir, err = chezmoi.NewOSPath(c.SourceDir).Normalize(c.normalizedHomeDir); err != nil {
		return err
	}
	if c.normalizedDestDir, err = chezmoi.NewOSPath(c.DestDir).Normalize(c.normalizedHomeDir); err != nil {
		return err
	}

	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        c.stderr,
		NoColor:    !c.color,
		TimeFormat: time.RFC3339,
	})
	if c.debug {
		logger = logger.Level(zerolog.DebugLevel)
	}
	c.logger = logger.With().Timestamp().Logger()

	switch {
	case cmd.Annotations[persistentStateMode] == persistentStateModeEmpty:
		c.persistentState = chezmoi.NewMockPersistentState()
	case cmd.Annotations[persistentStateMode] == persistentStateModeReadOnly:
		persistentStateFile := c.persistentStateFile().String()
		c.persistentState, err = chezmoi.NewBoltPersistentState(c.fs, persistentStateFile, chezmoi.BoltPersistentStateReadOnly)
		if err != nil {
			return err
		}
	case cmd.Annotations[persistentStateMode] == persistentStateModeReadMockWrite:
		fallthrough
	case cmd.Annotations[persistentStateMode] == persistentStateModeReadWrite && c.dryRun:
		persistentStateFile := c.persistentStateFile().String()
		persistentState, err := chezmoi.NewBoltPersistentState(c.fs, persistentStateFile, chezmoi.BoltPersistentStateReadOnly)
		if err != nil {
			return err
		}
		dryRunPeristentState := chezmoi.NewMockPersistentState()
		if err := persistentState.CopyTo(dryRunPeristentState); err != nil {
			return err
		}
		if err := persistentState.Close(); err != nil {
			return err
		}
		c.persistentState = dryRunPeristentState
	case cmd.Annotations[persistentStateMode] == persistentStateModeReadWrite:
		persistentStateFile := c.persistentStateFile().String()
		c.persistentState, err = chezmoi.NewBoltPersistentState(c.fs, persistentStateFile, chezmoi.BoltPersistentStateReadWrite)
		if err != nil {
			return err
		}
	default:
		c.persistentState = nil
	}
	if c.debug && c.persistentState != nil {
		c.persistentState = chezmoi.NewDebugPersistentState(c.persistentState, c.logger)
	}

	c.baseSystem = chezmoi.NewRealSystem(c.fs)
	if c.debug {
		c.baseSystem = chezmoi.NewDebugSystem(c.baseSystem, c.logger)
	}

	c.sourceSystem = c.baseSystem
	c.destSystem = c.baseSystem
	if !boolAnnotation(cmd, modifiesDestinationDirectory) {
		c.destSystem = chezmoi.NewReadOnlySystem(c.destSystem)
	}
	if !boolAnnotation(cmd, modifiesSourceDirectory) {
		c.sourceSystem = chezmoi.NewReadOnlySystem(c.sourceSystem)
	}
	if c.dryRun {
		c.sourceSystem = chezmoi.NewDryRunSystem(c.sourceSystem)
		c.destSystem = chezmoi.NewDryRunSystem(c.destSystem)
	}
	if c.verbose {
		c.sourceSystem = chezmoi.NewGitDiffSystem(c.sourceSystem, c.stdout, c.normalizedSourceDir, c.color)
		c.destSystem = chezmoi.NewGitDiffSystem(c.destSystem, c.stdout, c.normalizedDestDir, c.color)
	}

	if boolAnnotation(cmd, requiresConfigDirectory) {
		if err := vfs.MkdirAll(c.baseSystem, path.Dir(c.normalizedConfigFile), 0o777); err != nil {
			return err
		}
	}

	if boolAnnotation(cmd, requiresSourceDirectory) {
		if err := vfs.MkdirAll(c.baseSystem, c.normalizedSourceDir, 0o777); err != nil {
			return err
		}
	}

	if boolAnnotation(cmd, runsCommands) {
		if runtime.GOOS == "linux" && c.bds.RuntimeDir != "" {
			// Snap sets the $XDG_RUNTIME_DIR environment variable to
			// /run/user/$uid/snap.$snap_name, but does not create this
			// directory. Consequently, any spawned processes that need
			// $XDG_DATA_DIR will fail. As a work-around, create the directory
			// if it does not exist. See
			// https://forum.snapcraft.io/t/wayland-dconf-and-xdg-runtime-dir/186/13.
			if err := vfs.MkdirAll(c.baseSystem, c.bds.RuntimeDir, 0o700); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Config) persistentStateFile() *chezmoi.OSPath {
	if c.configFile != "" {
		return chezmoi.NewOSPath(c.configFile).Dir().Join(persistentStateFilename)
	}
	for _, configDir := range c.bds.ConfigDirs {
		persistentStateFile := filepath.Join(configDir, "chezmoi", persistentStateFilename)
		if _, err := os.Stat(persistentStateFile); err == nil {
			return chezmoi.NewOSPath(persistentStateFile)
		}
	}
	return defaultConfigFile(c.fs, c.bds).Dir().Join(persistentStateFilename)
}

func (c *Config) preApply(targetName string, targetEntryState, lastWrittenEntryState, actualEntryState *chezmoi.EntryState) error {
	switch {
	case c.force:
		return nil
	case lastWrittenEntryState == nil:
		return nil
	case lastWrittenEntryState.Equivalent(actualEntryState, c.Umask.FileMode()):
		return nil
	}
	switch choice, err := c.prompt(fmt.Sprintf("%s has changed since chezmoi last wrote it, overwrite", targetName), "ynqa"); {
	case err != nil:
		return err
	case choice == 'a':
		c.force = true
		return nil
	case choice == 'n':
		return chezmoi.Skip
	case choice == 'q':
		return ErrExitCode(1)
	default:
		return nil
	}
}

func (c *Config) prompt(s, choices string) (byte, error) {
	ttyReader, ttyWriter, err := c.getTTY()
	if err != nil {
		return 0, err
	}
	for {
		_, err := fmt.Fprintf(ttyWriter, "%s [%s]? ", s, strings.Join(strings.Split(choices, ""), ","))
		if err != nil {
			return 0, err
		}
		line, err := ttyReader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimSpace(line)
		if len(line) == 1 && strings.IndexByte(choices, line[0]) != -1 {
			return line[0], nil
		}
	}
}

func (c *Config) readConfig() error {
	v := viper.New()
	v.SetConfigFile(c.normalizedConfigFile)
	v.SetFs(vfsafero.NewAferoFS(c.fs))
	switch err := v.ReadInConfig(); {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return err
	}
	if err := v.Unmarshal(c); err != nil {
		return err
	}
	if err := c.validateData(); err != nil {
		return err
	}
	return nil
}

func (c *Config) run(dir, name string, args []string) error {
	cmd := exec.Command(name, args...)
	if dir != "" {
		var err error
		cmd.Dir, err = c.baseSystem.RawPath(dir)
		if err != nil {
			return err
		}
	}
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	return c.baseSystem.RunCmd(cmd)
}

func (c *Config) runEditor(args []string) error {
	editor, editorArgs := c.editor()
	return c.run("", editor, append(editorArgs, args...))
}

func (c *Config) sourcePaths(s *chezmoi.SourceState, args []string) ([]string, error) {
	targetNames, err := c.targetNames(s, args, targetNamesOptions{
		mustBeInSourceState: true,
		recursive:           false,
	})
	if err != nil {
		return nil, err
	}
	sourcePaths := make([]string, 0, len(targetNames))
	for _, targetName := range targetNames {
		sourcePath := s.MustEntry(targetName).Path()
		sourcePaths = append(sourcePaths, sourcePath)
	}
	return sourcePaths, nil
}

func (c *Config) sourceState() (*chezmoi.SourceState, error) {
	s := chezmoi.NewSourceState(
		chezmoi.WithDestDir(c.normalizedDestDir),
		chezmoi.WithPriorityTemplateData(c.Data),
		chezmoi.WithSourceDir(c.normalizedSourceDir),
		chezmoi.WithSystem(c.sourceSystem),
		chezmoi.WithTemplateData(c.defaultTemplateData()),
		chezmoi.WithTemplateFuncs(c.templateFuncs),
		chezmoi.WithTemplateOptions(c.Template.Options),
	)

	if err := s.Read(); err != nil {
		return nil, err
	}

	if minVersion := s.MinVersion(); c.version != nil && c.version.LessThan(minVersion) {
		return nil, fmt.Errorf("source state requires version %s or later, chezmoi is version %s", minVersion, c.version)
	}

	return s, nil
}

type targetNamesOptions struct {
	mustBeInSourceState bool
	recursive           bool
}

func (c *Config) targetNames(s *chezmoi.SourceState, args []string, options targetNamesOptions) ([]string, error) {
	targetNames := make([]string, 0, len(args))
	for _, arg := range args {
		targetName, err := c.getTargetName(chezmoi.NewOSPath(arg))
		if err != nil {
			return nil, err
		}
		if options.mustBeInSourceState {
			if _, ok := s.Entry(targetName); !ok {
				return nil, fmt.Errorf("%s: not in source state", arg)
			}
		}
		targetNames = append(targetNames, targetName)
		if options.recursive {
			targetNamePrefix := targetName + "/"
			for _, targetName := range s.TargetNames() {
				if strings.HasPrefix(targetName, targetNamePrefix) {
					targetNames = append(targetNames, targetName)
				}
			}
		}
	}

	if len(targetNames) == 0 {
		return nil, nil
	}

	// Sort and de-duplicate targetNames in place.
	sort.Strings(targetNames)
	n := 1
	for i := 1; i < len(targetNames); i++ {
		if targetNames[i] != targetNames[i-1] {
			targetNames[n] = targetNames[i]
			n++
		}
	}
	return targetNames[:n], nil
}

func (c *Config) useBuiltinGit() (bool, error) {
	if c.UseBuiltinGit == "" || strings.ToLower(c.UseBuiltinGit) == "auto" {
		if _, err := exec.LookPath(c.Git.Command); err == nil {
			return false, nil
		}
		return true, nil
	}
	return parseBool(c.UseBuiltinGit)
}

func (c *Config) validateData() error {
	return validateKeys(c.Data, identifierRx)
}

func (c *Config) writeOutput(data []byte) error {
	if c.output == "" || c.output == "-" {
		_, err := c.stdout.Write(data)
		return err
	}
	return c.baseSystem.WriteFile(c.output, data, 0o666)
}

func (c *Config) writeOutputString(data string) error {
	return c.writeOutput([]byte(data))
}

// withVersionInfo sets the version information.
func withVersionInfo(versionInfo VersionInfo) configOption {
	return func(c *Config) error {
		var version *semver.Version
		var versionElems []string
		if versionInfo.Version != "" {
			var err error
			version, err = semver.NewVersion(strings.TrimPrefix(versionInfo.Version, "v"))
			if err != nil {
				return err
			}
			versionElems = append(versionElems, version.String())
		} else {
			versionElems = append(versionElems, "dev")
		}
		if versionInfo.Commit != "" {
			versionElems = append(versionElems, "commit "+versionInfo.Commit)
		}
		if versionInfo.Date != "" {
			versionElems = append(versionElems, "built at "+versionInfo.Date)
		}
		if versionInfo.BuiltBy != "" {
			versionElems = append(versionElems, "built by "+versionInfo.BuiltBy)
		}
		c.version = version
		c.versionInfo = versionInfo
		c.versionStr = strings.Join(versionElems, ", ")
		return nil
	}
}

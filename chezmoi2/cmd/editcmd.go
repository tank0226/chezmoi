package cmd

import (
	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoi"
)

type editCmdConfig struct {
	Command   string
	Args      []string
	apply     bool
	include   *chezmoi.IncludeSet
	recursive bool
}

func (c *Config) newEditCmd() *cobra.Command {
	editCmd := &cobra.Command{
		Use:     "edit targets...",
		Short:   "Edit the source state of a target",
		Long:    mustLongHelp("edit"),
		Example: example("edit"),
		RunE:    c.makeRunEWithSourceState(c.runEditCmd),
		Annotations: map[string]string{
			modifiesDestinationDirectory: "true",
			modifiesSourceDirectory:      "true",
			persistentStateMode:          persistentStateModeReadWrite,
			requiresSourceDirectory:      "true",
			runsCommands:                 "true",
		},
	}

	flags := editCmd.Flags()
	flags.BoolVarP(&c.Edit.apply, "apply", "a", c.Edit.apply, "apply edit after editing")

	return editCmd
}

func (c *Config) runEditCmd(cmd *cobra.Command, args []string, s *chezmoi.SourceState) error {
	var sourceAbsPaths chezmoi.AbsPaths
	if len(args) == 0 {
		sourceAbsPaths = chezmoi.AbsPaths{c.normalizedSourceDir}
	} else {
		var err error
		sourceAbsPaths, err = c.sourceAbsPaths(s, args)
		if err != nil {
			return err
		}
	}

	// FIXME transparently decrypt encrypted files

	sourceAbsPathStrs := make([]string, 0, len(sourceAbsPaths))
	for _, sourceAbsPath := range sourceAbsPaths {
		sourceAbsPathStrs = append(sourceAbsPathStrs, sourceAbsPath.String())
	}
	if err := c.runEditor(sourceAbsPathStrs); err != nil {
		return err
	}

	if !c.Edit.apply {
		return nil
	}

	return c.applyArgs(c.destSystem, c.normalizedDestDir, args, c.Edit.include, c.Edit.recursive, c.Umask.FileMode(), c.preApply)
}

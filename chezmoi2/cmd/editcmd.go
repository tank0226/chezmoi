package cmd

import (
	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoi"
)

type editCmdConfig struct {
	Command string
	Args    []string
	apply   bool
	include *chezmoi.IncludeSet
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
	flags.VarP(c.Edit.include, "include", "i", "include entry types")

	return editCmd
}

func (c *Config) runEditCmd(cmd *cobra.Command, args []string, s *chezmoi.SourceState) error {
	if len(args) == 0 {
		if err := c.runEditor([]string{string(c.sourceDirAbsPath)}); err != nil {
			return err
		}
		if c.Edit.apply {
			if err := c.applyArgs(c.destSystem, c.destDirAbsPath, nil, c.Edit.include, recursive, c.Umask.FileMode(), c.preApply); err != nil {
				return err
			}
		}
		return nil
	}

	sourceAbsPaths, err := c.sourceAbsPaths(s, args)
	if err != nil {
		return err
	}

	// FIXME transparently decrypt encrypted files

	sourceAbsPathStrs := make([]string, 0, len(sourceAbsPaths))
	for _, sourceAbsPath := range sourceAbsPaths {
		sourceAbsPathStrs = append(sourceAbsPathStrs, string(sourceAbsPath))
	}
	if err := c.runEditor(sourceAbsPathStrs); err != nil {
		return err
	}

	if !c.Edit.apply {
		return nil
	}

	return c.applyArgs(c.destSystem, c.destDirAbsPath, args, c.Edit.include, nonRecursive, c.Umask.FileMode(), c.preApply)
}

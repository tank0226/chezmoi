package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoi"
)

func (c *Config) newRemoveCmd() *cobra.Command {
	removeCmd := &cobra.Command{
		Use:     "remove target...",
		Aliases: []string{"rm"},
		Short:   "Remove a target from the source state and the destination directory",
		Long:    mustLongHelp("remove"),
		Example: example("remove"),
		Args:    cobra.MinimumNArgs(1),
		RunE:    c.makeRunEWithSourceState(c.runRemoveCmd),
		Annotations: map[string]string{
			modifiesDestinationDirectory: "true",
			modifiesSourceDirectory:      "true",
		},
	}

	return removeCmd
}

func (c *Config) runRemoveCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	targetRelPaths, err := c.targetRelPaths(sourceState, args, targetRelPathsOptions{
		recursive:           false,
		mustBeInSourceState: true,
	})
	if err != nil {
		return err
	}

	for _, targetRelPath := range targetRelPaths {
		destAbsPath := c.normalizedDestDir.Join(targetRelPath)
		sourceAbsPath := sourceState.MustEntry(targetRelPath).SourceRelPath()
		if !c.force {
			choice, err := c.prompt(fmt.Sprintf("Remove %s and %s", destAbsPath, sourceAbsPath), "ynqa")
			if err != nil {
				return err
			}
			switch choice {
			case 'y':
			case 'n':
				continue
			case 'q':
				return nil
			case 'a':
				c.force = true
			}
		}
		if err := c.destSystem.RemoveAll(destAbsPath.String()); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := c.sourceSystem.RemoveAll(sourceAbsPath.String()); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
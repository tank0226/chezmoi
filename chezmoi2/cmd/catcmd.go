package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoi"
)

func (c *Config) newCatCmd() *cobra.Command {
	catCmd := &cobra.Command{
		Use:     "cat target...",
		Short:   "Print the target contents of a file or symlink",
		Long:    mustLongHelp("cat"),
		Example: example("cat"),
		Args:    cobra.MinimumNArgs(1),
		RunE:    c.makeRunEWithSourceState(c.runCatCmd),
	}

	return catCmd
}

func (c *Config) runCatCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	targetRelPaths, err := c.targetRelPaths(sourceState, args, targetRelPathsOptions{
		mustBeInSourceState: true,
	})
	if err != nil {
		return err
	}

	sb := strings.Builder{}
	for _, targetRelPath := range targetRelPaths {
		targetStateEntry, err := sourceState.MustEntry(targetRelPath).TargetStateEntry()
		if err != nil {
			return fmt.Errorf("%s: %w", targetRelPath, err)
		}
		switch targetStateEntry := targetStateEntry.(type) {
		case *chezmoi.TargetStateFile:
			contents, err := targetStateEntry.Contents()
			if err != nil {
				return fmt.Errorf("%s: %w", targetRelPath, err)
			}
			sb.Write(contents)
		case *chezmoi.TargetStatePresent:
			contents, err := targetStateEntry.Contents()
			if err != nil {
				return fmt.Errorf("%s: %w", targetRelPath, err)
			}
			sb.Write(contents)
		case *chezmoi.TargetStateSymlink:
			linkname, err := targetStateEntry.Linkname()
			if err != nil {
				return fmt.Errorf("%s: %w", targetRelPath, err)
			}
			sb.WriteString(linkname + "\n")
		default:
			return fmt.Errorf("%s: not a file or symlink", targetRelPath)
		}
	}
	return c.writeOutputString(sb.String())
}

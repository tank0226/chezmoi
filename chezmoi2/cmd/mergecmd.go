package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoi"
)

type mergeCmdConfig struct {
	Command string
	Args    []string
}

func (c *Config) newMergeCmd() *cobra.Command {
	mergeCmd := &cobra.Command{
		Use:     "merge target...",
		Args:    cobra.MinimumNArgs(1),
		Short:   "Perform a three-way merge between the destination state, the source state, and the target state",
		Long:    mustLongHelp("merge"),
		Example: example("merge"),
		RunE:    c.makeRunEWithSourceState(c.runMergeCmd),
		Annotations: map[string]string{
			modifiesSourceDirectory: "true",
			requiresSourceDirectory: "true",
		},
	}

	return mergeCmd
}

func (c *Config) runMergeCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	targetNames, err := c.targetNames(sourceState, args, targetNamesOptions{
		mustBeInSourceState: false,
		recursive:           true,
	})
	if err != nil {
		return err
	}

	// Create a temporary directory to store the target state and ensure that it
	// is removed afterwards. We cannot use fs as it lacks TempDir
	// functionality.
	tempDir, err := ioutil.TempDir("", "chezmoi")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	tempDirAbsPath := chezmoi.AbsPath(tempDir)

	for _, targetName := range targetNames {
		sourceStateEntry := sourceState.MustEntry(targetName)
		// FIXME sourceStateEntry.TargetStateEntry eagerly evaluates the return
		// targetStateEntry's contents, which means that we cannot fallback to a
		// two-way merge if the source state's contents cannot be decrypted or
		// are an invalid template
		targetStateEntry, err := sourceStateEntry.TargetStateEntry()
		if err != nil {
			return fmt.Errorf("%s: %w", targetName, err)
		}
		targetStateFile, ok := targetStateEntry.(*chezmoi.TargetStateFile)
		if !ok {
			// LATER consider handling symlinks?
			return fmt.Errorf("%s: not a file", targetName)
		}
		contents, err := targetStateFile.Contents()
		if err != nil {
			return err
		}
		targetStatePath := tempDirAbsPath.Join(chezmoi.RelPath(targetName.Base()))
		if err := c.baseSystem.WriteFile(targetStatePath.String(), contents, 0o600); err != nil {
			return err
		}
		args := append(
			append([]string{}, c.Merge.Args...),
			c.normalizedDestDir.Join(targetName).String(),
			c.normalizedSourceDir.Join(sourceStateEntry.SourceRelPath().RelPath()).String(),
			targetStatePath.String(),
		)
		if err := c.run(c.normalizedDestDir, c.Merge.Command, args); err != nil {
			return fmt.Errorf("%s: %w", targetName, err)
		}
	}

	return nil
}

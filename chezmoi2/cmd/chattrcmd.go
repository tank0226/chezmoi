package cmd

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoi"
)

type boolModifier int

const (
	boolModifierSet            boolModifier = 1
	boolModifierLeaveUnchanged boolModifier = 0
	boolModifierClear          boolModifier = -1
)

type orderModifier int

const (
	orderModifierSetFirst       orderModifier = -2
	orderModifierClearFirst     orderModifier = -1
	orderModifierLeaveUnchanged orderModifier = 0
	orderModifierClearLast      orderModifier = 1
	orderModifierSetLast        orderModifier = 2
)

type attrModifier struct {
	empty      boolModifier
	encrypted  boolModifier
	exact      boolModifier
	executable boolModifier
	once       boolModifier
	order      orderModifier
	private    boolModifier
	template   boolModifier
}

func (c *Config) newChattrCmd() *cobra.Command {
	attrs := []string{
		"empty", "e",
		"encrypted",
		"exact",
		"executable", "x",
		"first", "f",
		"last", "l",
		"once", "o",
		"private", "p",
		"template", "t",
	}
	validArgs := make([]string, 0, 4*len(attrs))
	for _, attribute := range attrs {
		validArgs = append(validArgs, attribute, "-"+attribute, "+"+attribute, "no"+attribute)
	}

	chattrCmd := &cobra.Command{
		Use:       "chattr attributes target...",
		Short:     "Change the attributes of a target in the source state",
		Long:      mustLongHelp("chattr"),
		Example:   example("chattr"),
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: validArgs,
		RunE:      c.makeRunEWithSourceState(c.runChattrCmd),
		Annotations: map[string]string{
			modifiesSourceDirectory: "true",
		},
	}

	return chattrCmd
}

func (c *Config) runChattrCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	// LATER should the core functionality of chattr move to chezmoi.SourceState?

	am, err := parseAttrModifier(args[0])
	if err != nil {
		return err
	}

	targetNames, err := c.targetNames(sourceState, args[1:], targetNamesOptions{
		mustBeInSourceState: true,
		recursive:           false,
	})
	if err != nil {
		return err
	}

	// Sort targets in reverse so we update children before their parent
	// directories.
	sort.Sort(sort.Reverse(sort.StringSlice(targetNames)))

	for _, targetName := range targetNames {
		sourceStateEntry := sourceState.MustEntry(targetName)
		sourceName, err := chezmoi.TrimDirPrefix(sourceStateEntry.Path(), c.normalizedSourceDir)
		if err != nil {
			return err
		}
		parentDirName, baseName := path.Split(sourceName)
		switch sourceStateEntry := sourceStateEntry.(type) {
		case *chezmoi.SourceStateDir:
			if newBaseName := am.modifyDirAttr(sourceStateEntry.Attr).BaseName(); newBaseName != baseName {
				newSourcePath := path.Join(c.normalizedSourceDir, parentDirName, newBaseName)
				if err := c.sourceSystem.Rename(sourceStateEntry.Path(), newSourcePath); err != nil {
					return err
				}
			}
		case *chezmoi.SourceStateFile:
			// FIXME encrypted attribute changes
			// FIXME when changing encrypted attribute add new file before removing old one
			if newBaseName := am.modifyFileAttr(sourceStateEntry.Attr).BaseName(); newBaseName != baseName {
				newSourcePath := path.Join(c.normalizedSourceDir, parentDirName, newBaseName)
				if err := c.sourceSystem.Rename(sourceStateEntry.Path(), newSourcePath); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (m boolModifier) modify(b bool) bool {
	switch m {
	case boolModifierSet:
		return true
	case boolModifierLeaveUnchanged:
		return b
	case boolModifierClear:
		return false
	default:
		panic(fmt.Sprintf("%d: unknown bool modifier", m))
	}
}

func (m orderModifier) modify(order int) int {
	switch m {
	case orderModifierSetFirst:
		return -1
	case orderModifierClearFirst:
		if order < 0 {
			return 0
		}
		return order
	case orderModifierLeaveUnchanged:
		return order
	case orderModifierClearLast:
		if order > 0 {
			return 0
		}
		return order
	case orderModifierSetLast:
		return 1
	default:
		panic(fmt.Sprintf("%d: unknown order modifier", m))
	}
}

func parseAttrModifier(s string) (*attrModifier, error) {
	am := &attrModifier{}
	for _, modifierStr := range strings.Split(s, ",") {
		modifierStr = strings.TrimSpace(modifierStr)
		if modifierStr == "" {
			continue
		}
		var bm boolModifier
		var attribute string
		switch {
		case modifierStr[0] == '-':
			bm = boolModifierClear
			attribute = modifierStr[1:]
		case modifierStr[0] == '+':
			bm = boolModifierSet
			attribute = modifierStr[1:]
		case strings.HasPrefix(modifierStr, "no"):
			bm = boolModifierClear
			attribute = modifierStr[2:]
		default:
			bm = boolModifierSet
			attribute = modifierStr
		}
		switch attribute {
		case "empty", "e":
			am.empty = bm
		case "encrypted":
			am.encrypted = bm
		case "exact":
			am.exact = bm
		case "executable", "x":
			am.executable = bm
		case "first", "f":
			switch bm {
			case boolModifierClear:
				am.order = orderModifierClearFirst
			case boolModifierLeaveUnchanged:
				am.order = orderModifierLeaveUnchanged
			case boolModifierSet:
				am.order = orderModifierSetFirst
			}
		case "last", "l":
			switch bm {
			case boolModifierClear:
				am.order = orderModifierClearLast
			case boolModifierLeaveUnchanged:
				am.order = orderModifierLeaveUnchanged
			case boolModifierSet:
				am.order = orderModifierSetLast
			}
		case "once", "o":
			am.once = bm
		case "private", "p":
			am.private = bm
		case "template", "t":
			am.template = bm
		default:
			return nil, fmt.Errorf("%s: unknown attribute", attribute)
		}
	}
	return am, nil
}

func (am *attrModifier) modifyDirAttr(dirAttr chezmoi.DirAttr) chezmoi.DirAttr {
	return chezmoi.DirAttr{
		Name:    dirAttr.Name,
		Exact:   am.exact.modify(dirAttr.Exact),
		Private: am.private.modify(dirAttr.Private),
	}
}

func (am *attrModifier) modifyFileAttr(fileAttr chezmoi.FileAttr) chezmoi.FileAttr {
	switch fileAttr.Type {
	case chezmoi.SourceFileTypeFile:
		return chezmoi.FileAttr{
			Name:       fileAttr.Name,
			Type:       chezmoi.SourceFileTypeFile,
			Empty:      am.empty.modify(fileAttr.Empty),
			Encrypted:  am.encrypted.modify(fileAttr.Encrypted),
			Executable: am.executable.modify(fileAttr.Executable),
			Private:    am.private.modify(fileAttr.Private),
			Template:   am.template.modify(fileAttr.Template),
		}
	case chezmoi.SourceFileTypePresent:
		return chezmoi.FileAttr{
			Name:       fileAttr.Name,
			Type:       chezmoi.SourceFileTypePresent,
			Encrypted:  am.encrypted.modify(fileAttr.Encrypted),
			Executable: am.executable.modify(fileAttr.Executable),
			Private:    am.private.modify(fileAttr.Private),
			Template:   am.template.modify(fileAttr.Template),
		}
	case chezmoi.SourceFileTypeScript:
		return chezmoi.FileAttr{
			Name:  fileAttr.Name,
			Type:  chezmoi.SourceFileTypeScript,
			Once:  am.once.modify(fileAttr.Once),
			Order: am.order.modify(fileAttr.Order),
		}
	case chezmoi.SourceFileTypeSymlink:
		return chezmoi.FileAttr{
			Name:     fileAttr.Name,
			Type:     chezmoi.SourceFileTypeSymlink,
			Template: am.template.modify(fileAttr.Template),
		}
	default:
		panic(fmt.Sprintf("%d: unknown source file type", fileAttr.Type))
	}
}

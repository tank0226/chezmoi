package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAttrModifier(t *testing.T) {
	for _, tc := range []struct {
		s           string
		expected    *attrModifier
		expectedErr bool
	}{
		{
			s: "empty",
			expected: &attrModifier{
				empty: boolModifierSet,
			},
		},
		{
			s: "+empty",
			expected: &attrModifier{
				empty: boolModifierSet,
			},
		},
		{
			s: "-empty",
			expected: &attrModifier{
				empty: boolModifierClear,
			},
		},
		{
			s: "noempty",
			expected: &attrModifier{
				empty: boolModifierClear,
			},
		},
		{
			s: "e",
			expected: &attrModifier{
				empty: boolModifierSet,
			},
		},
		{
			s: "encrypted",
			expected: &attrModifier{
				encrypted: boolModifierSet,
			},
		},
		{
			s: "executable",
			expected: &attrModifier{
				executable: boolModifierSet,
			},
		},
		{
			s: "x",
			expected: &attrModifier{
				executable: boolModifierSet,
			},
		},
		{
			s: "f",
			expected: &attrModifier{
				order: orderModifierSetFirst,
			},
		},
		{
			s: "-f",
			expected: &attrModifier{
				order: orderModifierClearFirst,
			},
		},
		{
			s: "last",
			expected: &attrModifier{
				order: orderModifierSetLast,
			},
		},
		{
			s: "nolast",
			expected: &attrModifier{
				order: orderModifierClearLast,
			},
		},
		{
			s: "once",
			expected: &attrModifier{
				once: boolModifierSet,
			},
		},
		{
			s: "private",
			expected: &attrModifier{
				private: boolModifierSet,
			},
		},
		{
			s: "p",
			expected: &attrModifier{
				private: boolModifierSet,
			},
		},
		{
			s: "template",
			expected: &attrModifier{
				template: boolModifierSet,
			},
		},
		{
			s: "t",
			expected: &attrModifier{
				template: boolModifierSet,
			},
		},
		{
			s: "empty,+executable,noprivate,-t",
			expected: &attrModifier{
				empty:      boolModifierSet,
				executable: boolModifierSet,
				private:    boolModifierClear,
				template:   boolModifierClear,
			},
		},
		{
			s: " empty , -private, notemplate ",
			expected: &attrModifier{
				empty:    boolModifierSet,
				private:  boolModifierClear,
				template: boolModifierClear,
			},
		},
		{
			s: "p,,-t",
			expected: &attrModifier{
				private:  boolModifierSet,
				template: boolModifierClear,
			},
		},
		{
			s:           "unknown",
			expectedErr: true,
		},
	} {
		t.Run(tc.s, func(t *testing.T) {
			actual, err := parseAttrModifier(tc.s)
			if tc.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, actual)
			}
		})
	}
}

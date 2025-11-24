package dotenv_test

import (
	"context"
	"strings"
	"testing"

	"github.com/compose-spec/dotenv"
	"gotest.tools/v3/assert"
)

func TestExplain(t *testing.T) {
	type test struct {
		name         string
		input        string
		variableName string
		expected     string
	}

	tests := []test{
		{
			name:         "no expansion",
			input:        "BASE=/usr",
			variableName: "BASE",
			expected: `Variable: BASE
Location: :1
Raw Value: /usr
Final Value: /usr
`,
		},
		{
			name:         "simple expansion with $VAR",
			input:        "BASE=/usr\nPATH=$BASE/bin",
			variableName: "PATH",
			expected: `Variable: PATH
Location: :2
Raw Value: $BASE/bin
Final Value: /usr/bin
Expanded from:
  - BASE=/usr at :1
`,
		},
		{
			name:         "simple expansion with ${VAR}",
			input:        "BASE=/usr\nPATH=${BASE}/bin",
			variableName: "PATH",
			expected: `Variable: PATH
Location: :2
Raw Value: ${BASE}/bin
Final Value: /usr/bin
Expanded from:
  - BASE=/usr at :1
`,
		},
		{
			name:         "chained expansion",
			input:        "BASE=/usr\nBIN=${BASE}/bin\nFULL=${BIN}:/opt/bin",
			variableName: "FULL",
			expected: `Variable: FULL
Location: :3
Raw Value: ${BIN}:/opt/bin
Final Value: /usr/bin:/opt/bin
Expanded from:
  - BIN=/usr/bin at :2
`,
		},
		{
			name:         "multiple expansions in one line",
			input:        "A=foo\nB=bar\nC=$A-$B",
			variableName: "C",
			expected: `Variable: C
Location: :3
Raw Value: $A-$B
Final Value: foo-bar
Expanded from:
  - A=foo at :1
  - B=bar at :2
`,
		},
		{
			name:         "expansion with default value",
			input:        "BASE=/usr\nPATH=${BASE:-/default}/bin",
			variableName: "PATH",
			expected: `Variable: PATH
Location: :2
Raw Value: ${BASE:-/default}/bin
Final Value: /usr/bin
Expanded from:
  - BASE=/usr at :1
`,
		},
		{
			name:         "expansion with replacement value",
			input:        "BASE=/usr\nPATH=${BASE:+/replaced}/bin",
			variableName: "PATH",
			expected: `Variable: PATH
Location: :2
Raw Value: ${BASE:+/replaced}/bin
Final Value: /replaced/bin
Expanded from:
  - BASE=/usr at :1
`,
		},
		{
			name:         "non-existent variable",
			input:        "FOO=bar",
			variableName: "NONEXISTENT",
			expected:     "Variable not found",
		},
		{
			name:         "required variable with question mark",
			input:        "BASE=/usr\nPATH=${BASE?BASE is required}/bin",
			variableName: "PATH",
			expected: `Variable: PATH
Location: :2
Raw Value: ${BASE?BASE is required}/bin
Final Value: /usr/bin
Expanded from:
  - BASE=/usr at :1
`,
		},
		{
			name:         "required variable with colon question mark",
			input:        "BASE=/usr\nPATH=${BASE:?BASE is required}/bin",
			variableName: "PATH",
			expected: `Variable: PATH
Location: :2
Raw Value: ${BASE:?BASE is required}/bin
Final Value: /usr/bin
Expanded from:
  - BASE=/usr at :1
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env, err := dotenv.Parse(context.TODO(), strings.NewReader(test.input))
			assert.NilError(t, err)

			explanation := env.Explain(test.variableName)
			assert.Equal(t, test.expected, explanation)
		})
	}
}

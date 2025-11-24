package dotenv_test

import (
	"context"
	"strings"
	"testing"

	"github.com/compose-spec/dotenv"
	"gotest.tools/v3/assert"
)

func TestParse(t *testing.T) {
	type test struct {
		name   string
		input  string
		expect map[string]string
		err    string
	}
	tests := []test{
		{
			name:  "unquoted",
			input: "FOO=BAR",
			expect: map[string]string{
				"FOO": "BAR",
			},
		},
		{
			name:  "export prefix",
			input: "export FOO=BAR",
			expect: map[string]string{
				"FOO": "BAR",
			},
		},
		{
			name:  "with comments",
			input: "# comment before\nFOO=BAR\n# comment after",
			expect: map[string]string{
				"FOO": "BAR",
			},
		},
		{
			name:  "space before equal",
			input: "FOO =bar",
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "space after equal",
			input: "FOO= bar",
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "colon separator",
			input: "FOO:bar",
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "colon with spaces",
			input: "FOO : bar",
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "leading spaces",
			input: "  FOO=bar",
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "trailing spaces",
			input: "FOO=bar  ",
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "leading and trailing spaces",
			input: "  FOO=bar  ",
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "double quoted value",
			input: `FOO="bar"`,
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "single quoted value",
			input: `FOO='bar'`,
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "double quoted with spaces",
			input: `FOO="hello world"`,
			expect: map[string]string{
				"FOO": "hello world",
			},
		},
		{
			name:  "single quoted with spaces",
			input: `FOO='hello world'`,
			expect: map[string]string{
				"FOO": "hello world",
			},
		},
		{
			name:  "double quoted with trailing spaces",
			input: `FOO="bar"  `,
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "single quoted with trailing spaces",
			input: `FOO='bar'  `,
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "double quoted with escaped quote",
			input: `FOO="bar \"baz\""`,
			expect: map[string]string{
				"FOO": `bar "baz"`,
			},
		},
		{
			name:  "double quoted with escaped backslash",
			input: `FOO="bar\\baz"`,
			expect: map[string]string{
				"FOO": `bar\baz`,
			},
		},
		{
			name:  "double quoted with newline",
			input: `FOO="bar\nbaz"`,
			expect: map[string]string{
				"FOO": "bar\nbaz",
			},
		},
		{
			name:  "double quoted with tab",
			input: `FOO="bar\tbaz"`,
			expect: map[string]string{
				"FOO": "bar\tbaz",
			},
		},
		{
			name:  "variable expansion with $VAR",
			input: "BASE=/usr\nPATH=$BASE/bin",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "/usr/bin",
			},
		},
		{
			name:  "variable expansion with ${VAR}",
			input: "BASE=/usr\nPATH=${BASE}/bin",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "/usr/bin",
			},
		},
		{
			name:  "variable expansion undefined variable",
			input: "PATH=$UNDEFINED/bin",
			expect: map[string]string{
				"PATH": "/bin",
			},
		},
		{
			name:  "variable expansion multiple references",
			input: "A=foo\nB=bar\nC=$A-$B",
			expect: map[string]string{
				"A": "foo",
				"B": "bar",
				"C": "foo-bar",
			},
		},
		{
			name:  "variable expansion in quoted value",
			input: "BASE=/usr\nPATH=\"$BASE/bin\"",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "/usr/bin",
			},
		},
		{
			name:  "blank value",
			input: "EMPTY=",
			expect: map[string]string{
				"EMPTY": "",
			},
		},
		{
			name:  "blank value with spaces",
			input: "EMPTY=  ",
			expect: map[string]string{
				"EMPTY": "",
			},
		},
		{
			name:  "default value with dash unset variable",
			input: "FOO=${UNSET-default}",
			expect: map[string]string{
				"FOO": "default",
			},
		},
		{
			name:  "default value with dash set variable",
			input: "BAR=value\nFOO=${BAR-default}",
			expect: map[string]string{
				"BAR": "value",
				"FOO": "value",
			},
		},
		{
			name:  "default value with dash empty variable",
			input: "BAR=\nFOO=${BAR-default}",
			expect: map[string]string{
				"BAR": "",
				"FOO": "",
			},
		},
		{
			name:  "default value with colon dash unset variable",
			input: "FOO=${UNSET:-default}",
			expect: map[string]string{
				"FOO": "default",
			},
		},
		{
			name:  "default value with colon dash empty variable",
			input: "BAR=\nFOO=${BAR:-default}",
			expect: map[string]string{
				"BAR": "",
				"FOO": "default",
			},
		},
		{
			name:  "default value with colon dash set variable",
			input: "BAR=value\nFOO=${BAR:-default}",
			expect: map[string]string{
				"BAR": "value",
				"FOO": "value",
			},
		},
		{
			name:  "replacement value with plus unset variable",
			input: "FOO=${UNSET+replacement}",
			expect: map[string]string{
				"FOO": "",
			},
		},
		{
			name:  "replacement value with plus set variable",
			input: "BAR=value\nFOO=${BAR+replacement}",
			expect: map[string]string{
				"BAR": "value",
				"FOO": "replacement",
			},
		},
		{
			name:  "replacement value with plus empty variable",
			input: "BAR=\nFOO=${BAR+replacement}",
			expect: map[string]string{
				"BAR": "",
				"FOO": "replacement",
			},
		},
		{
			name:  "replacement value with colon plus unset variable",
			input: "FOO=${UNSET:+replacement}",
			expect: map[string]string{
				"FOO": "",
			},
		},
		{
			name:  "replacement value with colon plus empty variable",
			input: "BAR=\nFOO=${BAR:+replacement}",
			expect: map[string]string{
				"BAR": "",
				"FOO": "",
			},
		},
		{
			name:  "replacement value with colon plus set variable",
			input: "BAR=value\nFOO=${BAR:+replacement}",
			expect: map[string]string{
				"BAR": "value",
				"FOO": "replacement",
			},
		},
		{
			name:  "required variable with question mark set variable",
			input: "BAR=value\nFOO=${BAR?BAR is required}",
			expect: map[string]string{
				"BAR": "value",
				"FOO": "value",
			},
		},
		{
			name:  "required variable with question mark empty variable",
			input: "BAR=\nFOO=${BAR?BAR is required}",
			expect: map[string]string{
				"BAR": "",
				"FOO": "",
			},
		},
		{
			name:  "required variable with colon question mark set variable",
			input: "BAR=value\nFOO=${BAR:?BAR is required}",
			expect: map[string]string{
				"BAR": "value",
				"FOO": "value",
			},
		},
		{
			name:  "required variable with question mark unset variable",
			input: "FOO=${UNSET?UNSET is required}",
			err:   "UNSET is required",
		},
		{
			name:  "required variable with question mark unset variable no message",
			input: "FOO=${UNSET?}",
			err:   "UNSET: required variable is not set",
		},
		{
			name:  "required variable with colon question mark unset variable",
			input: "FOO=${UNSET:?UNSET is required}",
			err:   "UNSET is required",
		},
		{
			name:  "required variable with colon question mark empty variable",
			input: "BAR=\nFOO=${BAR:?BAR is required}",
			err:   "BAR is required",
		},
		{
			name:  "required variable with colon question mark unset variable no message",
			input: "FOO=${UNSET:?}",
			err:   "UNSET: required variable is not set",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env, err := dotenv.Parse(context.TODO(), strings.NewReader(test.input))
			if test.err == "" {
				assert.NilError(t, err)
				assert.DeepEqual(t, test.expect, env.Variables())
			} else {
				assert.Error(t, err, test.err)
			}
		})
	}
}

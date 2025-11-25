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
			name:  "variable name with underscore",
			input: "FOO_BAR=value",
			expect: map[string]string{
				"FOO_BAR": "value",
			},
		},
		{
			name:  "variable name with dot",
			input: "foo.bar=value",
			expect: map[string]string{
				"foo.bar": "value",
			},
		},
		{
			name:  "variable name with hyphen",
			input: "foo-bar=value",
			expect: map[string]string{
				"foo-bar": "value",
			},
		},
		{
			name:  "variable name with numbers",
			input: "VAR123=value",
			expect: map[string]string{
				"VAR123": "value",
			},
		},
		{
			name:  "variable name with mixed characters",
			input: "foo.bar-baz_123=value",
			expect: map[string]string{
				"foo.bar-baz_123": "value",
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
			name:  "export without assignment",
			input: "FOO=BAR\nexport FOO",
			expect: map[string]string{
				"FOO": "BAR",
			},
		},
		{
			name:  "export without assignment is ignored",
			input: "BASE=/usr\nexport BASE\nPATH=$BASE/bin",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "/usr/bin",
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
			name:  "inline comment with unquoted value",
			input: "FOO=bar # this is a comment",
			expect: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:  "inline comment with double quoted value",
			input: "FOO=\"bar # not a comment\"",
			expect: map[string]string{
				"FOO": "bar # not a comment",
			},
		},
		{
			name:  "inline comment with single quoted value",
			input: "FOO='bar # not a comment'",
			expect: map[string]string{
				"FOO": "bar # not a comment",
			},
		},
		{
			name:  "multiple inline comments",
			input: "FOO=bar # comment\nBAZ=qux # another comment",
			expect: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
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
			name:  "yaml style",
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
			name:  "double quoted multi-line value",
			input: "FOO=\"line1\nline2\nline3\"",
			expect: map[string]string{
				"FOO": "line1\nline2\nline3",
			},
		},
		{
			name:  "single quoted multi-line value",
			input: "FOO='line1\nline2\nline3'",
			expect: map[string]string{
				"FOO": "line1\nline2\nline3",
			},
		},
		{
			name:  "multi-line value with multiple variables",
			input: "FOO=\"multi\nline\"\nBAR=single",
			expect: map[string]string{
				"FOO": "multi\nline",
				"BAR": "single",
			},
		},
		{
			name:  "multi-line value preserves indentation",
			input: "FOO=\"line1\n  indented\n    more indented\"",
			expect: map[string]string{
				"FOO": "line1\n  indented\n    more indented",
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
			name:  "variable expansion undefined variable with braces",
			input: "PATH=${UNDEFINED}/bin",
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
			name:  "variable expansion in double quoted value",
			input: "BASE=/usr\nPATH=\"$BASE/bin\"",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "/usr/bin",
			},
		},
		{
			name:  "no variable expansion in single quoted value",
			input: "BASE=/usr\nPATH='$BASE/bin'",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "$BASE/bin",
			},
		},
		{
			name:  "no variable expansion in single quoted value with braces",
			input: "BASE=/usr\nPATH='${BASE}/bin'",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "${BASE}/bin",
			},
		},
		{
			name:  "variable expansion in unquoted value",
			input: "BASE=/usr\nPATH=$BASE/bin",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "/usr/bin",
			},
		},
		{
			name:  "escaped dollar sign with $VAR syntax",
			input: "BASE=/usr\nPATH=\\$BASE/bin",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "$BASE/bin",
			},
		},
		{
			name:  "escaped dollar sign with ${VAR} syntax",
			input: "BASE=/usr\nPATH=\\${BASE}/bin",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "${BASE}/bin",
			},
		},
		{
			name:  "escaped dollar sign in double quoted value",
			input: "BASE=/usr\nPATH=\"\\$BASE/bin\"",
			expect: map[string]string{
				"BASE": "/usr",
				"PATH": "$BASE/bin",
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
		{
			name:  "nested expansion in default value with dash",
			input: "BAR=hello\nFOO=${UNSET-$BAR}",
			expect: map[string]string{
				"BAR": "hello",
				"FOO": "hello",
			},
		},
		{
			name:  "nested expansion in default value with colon dash",
			input: "BAR=world\nFOO=${UNSET:-$BAR}",
			expect: map[string]string{
				"BAR": "world",
				"FOO": "world",
			},
		},
		{
			name:  "nested expansion with braces in default value",
			input: "BAR=test\nFOO=${UNSET:-${BAR}}",
			expect: map[string]string{
				"BAR": "test",
				"FOO": "test",
			},
		},
		{
			name:  "nested expansion in replacement value with plus",
			input: "BAR=replaced\nBIZ=set\nFOO=${BIZ+$BAR}",
			expect: map[string]string{
				"BAR": "replaced",
				"BIZ": "set",
				"FOO": "replaced",
			},
		},
		{
			name:  "nested expansion in replacement value with colon plus",
			input: "BAR=value\nBIZ=set\nFOO=${BIZ:+$BAR}",
			expect: map[string]string{
				"BAR": "value",
				"BIZ": "set",
				"FOO": "value",
			},
		},
		{
			name:  "nested expansion with undefined variable in default",
			input: "FOO=${UNSET:-$UNDEFINED}",
			expect: map[string]string{
				"FOO": "",
			},
		},
		{
			name:  "nested expansion with multiple variables in default",
			input: "A=hello\nB=world\nFOO=${UNSET:-$A $B}",
			expect: map[string]string{
				"A": "hello",
				"B": "world",
				"FOO": "hello world",
			},
		},
		{
			name:  "multiple levels of nested expansion",
			input: "A=final\nB=${UNSET:-$A}\nC=${UNSET:-$B}",
			expect: map[string]string{
				"A": "final",
				"B": "final",
				"C": "final",
			},
		},
		{
			name:  "nested expansion with literal text",
			input: "BAR=value\nFOO=${UNSET:-prefix-$BAR-suffix}",
			expect: map[string]string{
				"BAR": "value",
				"FOO": "prefix-value-suffix",
			},
		},
		{
			name:  "default value not expanded when variable is set",
			input: "VAR=value\nFOO=${VAR-${BAR?BAR is required}}",
			expect: map[string]string{
				"VAR": "value",
				"FOO": "value",
			},
		},
		{
			name:  "default value with error expanded when variable is unset",
			input: "FOO=${VAR-${BAR?BAR is required}}",
			err:   "BAR is required",
		},
		{
			name:  "default value not expanded when variable is set with colon dash",
			input: "VAR=value\nFOO=${VAR:-${BAR?BAR is required}}",
			expect: map[string]string{
				"VAR": "value",
				"FOO": "value",
			},
		},
		{
			name:  "default value with error expanded when variable is empty with colon dash",
			input: "VAR=\nFOO=${VAR:-${BAR?BAR is required}}",
			err:   "BAR is required",
		},
		{
			name:  "line without separator",
			input: "FOO=BAR\nINVALIDLINE",
			err:   "line 2: no separator found in line: INVALIDLINE",
		},
		{
			name:  "export undefined variable",
			input: "FOO=BAR\nexport UNDEFINED",
			err:   "line 2 \"UNDEFINED\" has an unset variable",
		},
		{
			name:  "invalid variable name with space",
			input: "FOO BAR=value",
			err:   "line 1: invalid variable name \"FOO BAR\"",
		},
		{
			name:  "invalid variable name with special character",
			input: "FOO@BAR=value",
			err:   "line 1: invalid variable name \"FOO@BAR\"",
		},
		{
			name:  "invalid variable name with dollar sign",
			input: "FOO$BAR=value",
			err:   "line 1: invalid variable name \"FOO$BAR\"",
		},
		{
			name:  "invalid variable name with bracket",
			input: "FOO[BAR]=value",
			err:   "line 1: invalid variable name \"FOO[BAR]\"",
		},
		{
			name:  "invalid variable name starting with digit",
			input: "123VAR=value",
			err:   "line 1: invalid variable name \"123VAR\"",
		},
		{
			name:  "invalid variable name starting with digit and underscore",
			input: "1_VAR=value",
			err:   "line 1: invalid variable name \"1_VAR\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env, err := dotenv.Parse(context.TODO(), strings.NewReader(test.input))
			if test.err == "" {
				assert.NilError(t, err)
				vars, err := env.Resolve()
				assert.NilError(t, err)
				assert.DeepEqual(t, test.expect, vars)
			} else if strings.Contains(test.err, "required") {
				// Error expected from Resolve() for required variable checks
				assert.NilError(t, err)
				_, err = env.Resolve()
				assert.Error(t, err, test.err)
			} else {
				// Error expected from Parse()
				assert.Error(t, err, test.err)
			}
		})
	}
}

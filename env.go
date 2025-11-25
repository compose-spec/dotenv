package dotenv

import (
	"fmt"
	"strings"
)

// Location tracks the source file and line number of an environment variable in the format "file:line"
type Location string

// QuoteStyle represents the quoting style of a variable value
type QuoteStyle int

const (
	Unquoted QuoteStyle = iota
	Quoted
	DoubleQuoted
)

// EnvFile represents a parsed .env file containing a list of variables
type EnvFile struct {
	Variables []Variable
	expanded  bool
}

// Resolve performs variable expansion and returns the environment variables as a map[string]string
func (e *EnvFile) Resolve() (map[string]string, error) {
	if !e.expanded {
		if err := e.expand(); err != nil {
			return nil, err
		}
		e.expanded = true
	}

	result := make(map[string]string, len(e.Variables))
	for _, variable := range e.Variables {
		result[variable.Name] = variable.Value
	}
	return result, nil
}

// expand processes variable expansion in the EnvFile
// It replaces $VARIABLE and ${VARIABLE} references with values from previously declared variables
func (e *EnvFile) expand() error {
	// Build a map of variables as we go for lookups
	vars := make(map[string]Variable)

	for i := range e.Variables {
		// Skip expansion for single-quoted variables
		if e.Variables[i].Quoted == Quoted {
			// For single-quoted variables, just copy RawValue to Value
			e.Variables[i].Value = e.Variables[i].RawValue
			vars[e.Variables[i].Name] = e.Variables[i]
			continue
		}

		// Expand the current variable's value using previously declared variables
		lookup := func(name string) (Variable, bool) {
			v, ok := vars[name]
			return v, ok
		}
		if err := e.Variables[i].expandValue(lookup); err != nil {
			return err
		}

		// Add the current variable to the map for future expansions
		vars[e.Variables[i].Name] = e.Variables[i]
	}
	return nil
}

// findClosingBrace finds the index of the closing brace that matches the opening brace
// at the given start position, accounting for nested braces
func findClosingBrace(value string, start int) int {
	depth := 1
	for i := start; i < len(value); i++ {
		if value[i] == '{' {
			depth++
		} else if value[i] == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1 // no matching closing brace found
}

// findOperator finds the first occurrence of the operator at the top level (not inside nested braces)
func findOperator(content string, operator string) int {
	depth := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '$' && i+1 < len(content) && content[i+1] == '{' {
			depth++
			i++ // skip the '{'
		} else if content[i] == '}' && depth > 0 {
			depth--
		} else if depth == 0 {
			// We're at the top level, check for operator
			if i+len(operator) <= len(content) && content[i:i+len(operator)] == operator {
				return i
			}
		}
	}
	return -1
}

// expandString expands variable references in a string value
func expandString(value string, lookup LookupFn) (string, map[string]Location, error) {
	expanded := make(map[string]Location)
	var result strings.Builder
	result.Grow(len(value))

	for i := 0; i < len(value); i++ {
		// Check for escaped dollar sign \$
		if value[i] == '\\' && i+1 < len(value) && value[i+1] == '$' {
			// Write literal $ and skip both characters
			result.WriteByte('$')
			i++ // skip the $
			continue
		}

		if value[i] == '$' && i+1 < len(value) {
			if value[i+1] == '{' {
				// ${VARIABLE} syntax with possible default/replacement value
				// Find the matching closing brace, accounting for nested braces
				endIdx := findClosingBrace(value, i+2)
				if endIdx != -1 {
					content := value[i+2 : endIdx]

					// Check for ${VAR:?error} (error if unset or empty)
					if colonQuestionIdx := findOperator(content, ":?"); colonQuestionIdx != -1 {
						varName := content[:colonQuestionIdx]
						errorMsg := content[colonQuestionIdx+2:]
						variable, ok := lookup(varName)
						if !ok || variable.Value == "" {
							if errorMsg == "" {
								return "", nil, fmt.Errorf("%s: required variable is not set", varName)
							}
							return "", nil, fmt.Errorf("%s", errorMsg)
						}
						result.WriteString(variable.Value)
						expanded[varName] = variable.Location
					} else if colonDashIdx := findOperator(content, ":-"); colonDashIdx != -1 {
						// Check for ${VAR:-default} (use default if unset or empty)
						varName := content[:colonDashIdx]
						defaultValue := content[colonDashIdx+2:]
						if variable, ok := lookup(varName); ok && variable.Value != "" {
							result.WriteString(variable.Value)
							expanded[varName] = variable.Location
						} else {
							// Recursively expand the default value
							expandedDefault, nestedExpanded, err := expandString(defaultValue, lookup)
							if err != nil {
								return "", nil, err
							}
							result.WriteString(expandedDefault)
							// Merge nested expanded variables
							for k, v := range nestedExpanded {
								expanded[k] = v
							}
						}
					} else if colonPlusIdx := findOperator(content, ":+"); colonPlusIdx != -1 {
						// Check for ${VAR:+replacement} (use replacement if set and non-empty)
						varName := content[:colonPlusIdx]
						replacement := content[colonPlusIdx+2:]
						if variable, ok := lookup(varName); ok && variable.Value != "" {
							// Recursively expand the replacement value
							expandedReplacement, nestedExpanded, err := expandString(replacement, lookup)
							if err != nil {
								return "", nil, err
							}
							result.WriteString(expandedReplacement)
							expanded[varName] = variable.Location
							// Merge nested expanded variables
							for k, v := range nestedExpanded {
								expanded[k] = v
							}
						}
						// Otherwise leave empty
					} else if questionIdx := findOperator(content, "?"); questionIdx != -1 {
						// Check for ${VAR?error} (error if unset, but can be empty)
						varName := content[:questionIdx]
						errorMsg := content[questionIdx+1:]
						if variable, ok := lookup(varName); !ok {
							if errorMsg == "" {
								return "", nil, fmt.Errorf("%s: required variable is not set", varName)
							}
							return "", nil, fmt.Errorf("%s", errorMsg)
						} else {
							result.WriteString(variable.Value)
							expanded[varName] = variable.Location
						}
					} else if dashIdx := findOperator(content, "-"); dashIdx != -1 {
						// Check for ${VAR-default} (use default if unset)
						varName := content[:dashIdx]
						defaultValue := content[dashIdx+1:]
						if variable, ok := lookup(varName); ok {
							result.WriteString(variable.Value)
							expanded[varName] = variable.Location
						} else {
							// Recursively expand the default value
							expandedDefault, nestedExpanded, err := expandString(defaultValue, lookup)
							if err != nil {
								return "", nil, err
							}
							result.WriteString(expandedDefault)
							// Merge nested expanded variables
							for k, v := range nestedExpanded {
								expanded[k] = v
							}
						}
					} else if plusIdx := findOperator(content, "+"); plusIdx != -1 {
						// Check for ${VAR+replacement} (use replacement if set)
						varName := content[:plusIdx]
						replacement := content[plusIdx+1:]
						if variable, ok := lookup(varName); ok {
							// Recursively expand the replacement value
							expandedReplacement, nestedExpanded, err := expandString(replacement, lookup)
							if err != nil {
								return "", nil, err
							}
							result.WriteString(expandedReplacement)
							expanded[varName] = variable.Location
							// Merge nested expanded variables
							for k, v := range nestedExpanded {
								expanded[k] = v
							}
						}
						// Otherwise leave empty
					} else {
						// Simple ${VAR} syntax
						if variable, ok := lookup(content); ok {
							result.WriteString(variable.Value)
							expanded[content] = variable.Location
						}
						// If variable not found, leave it empty (standard behavior)
					}
					i = endIdx // skip past the }
				} else {
					// No closing brace, write literal
					result.WriteByte(value[i])
				}
			} else if isVarNameChar(value[i+1]) {
				// $VARIABLE syntax
				j := i + 1
				for j < len(value) && isVarNameChar(value[j]) {
					j++
				}
				varName := value[i+1 : j]
				if variable, ok := lookup(varName); ok {
					result.WriteString(variable.Value)
					expanded[varName] = variable.Location
				}
				// If variable not found, leave it empty
				i = j - 1 // will be incremented by loop
			} else {
				// $ followed by non-variable char, write literal
				result.WriteByte(value[i])
			}
		} else {
			result.WriteByte(value[i])
		}
	}

	return result.String(), expanded, nil
}

// isVarNameChar returns true if the character is valid in a variable name
func isVarNameChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
}

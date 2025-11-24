package dotenv

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// unescapeDoubleQuoted processes escape sequences in a double-quoted string
func unescapeDoubleQuoted(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			// Handle escape sequences
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
				i++
			case 't':
				result.WriteByte('\t')
				i++
			case 'r':
				result.WriteByte('\r')
				i++
			case '\\':
				result.WriteByte('\\')
				i++
			case '"':
				result.WriteByte('"')
				i++
			default:
				// Unknown escape sequence, keep the backslash
				result.WriteByte(s[i])
			}
		} else {
			result.WriteByte(s[i])
		}
	}

	return result.String()
}

// Parse reads an .env file from the provided reader and returns a parsed EnvFile
func Parse(ctx context.Context, reader io.Reader) (*EnvFile, error) {
	envFile := &EnvFile{
		variables: []Variable{},
	}

	scanner := bufio.NewScanner(reader)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove 'export ' prefix if present
		if strings.HasPrefix(line, "export ") {
			line = line[7:]
		}

		// Find the separator (= or :)
		equalIdx := strings.Index(line, "=")
		colonIdx := strings.Index(line, ":")

		var separatorIdx int
		if equalIdx == -1 && colonIdx == -1 {
			// No separator found, skip line
			continue
		} else if equalIdx == -1 {
			separatorIdx = colonIdx
		} else if colonIdx == -1 {
			separatorIdx = equalIdx
		} else {
			// Both found, use the first one
			if equalIdx < colonIdx {
				separatorIdx = equalIdx
			} else {
				separatorIdx = colonIdx
			}
		}

		// Split on the separator
		name := strings.TrimSpace(line[:separatorIdx])
		value := strings.TrimSpace(line[separatorIdx+1:])

		// Remove surrounding quotes if present
		if len(value) >= 2 {
			if value[0] == '"' && value[len(value)-1] == '"' {
				// Double-quoted: remove quotes and process escape sequences
				value = unescapeDoubleQuoted(value[1 : len(value)-1])
			} else if value[0] == '\'' && value[len(value)-1] == '\'' {
				// Single-quoted: just remove quotes, no escape processing
				value = value[1 : len(value)-1]
			}
		}

		variable := Variable{
			Name:     name,
			Value:    value,
			RawValue: value,
			Location: Location(fmt.Sprintf(":%d", lineNumber)),
			Expanded: make(map[string]Location),
		}

		envFile.variables = append(envFile.variables, variable)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Perform variable expansion
	if err := expand(envFile); err != nil {
		return nil, err
	}

	return envFile, nil
}

// expand processes variable expansion in the EnvFile
// It replaces $VARIABLE and ${VARIABLE} references with values from previously declared variables
func expand(envFile *EnvFile) error {
	// Build a map of variables as we go for lookups
	varMap := make(map[string]string)
	locMap := make(map[string]Location)

	for i := range envFile.variables {
		// Track which variables are expanded
		expanded := make(map[string]Location)

		// Expand the current variable's value using previously declared variables
		val, err := expandValue(envFile.variables[i].Value, varMap, locMap, expanded)
		if err != nil {
			return err
		}
		envFile.variables[i].Value = val
		envFile.variables[i].Expanded = expanded

		// Add the current variable to the map for future expansions
		varMap[envFile.variables[i].Name] = envFile.variables[i].Value
		locMap[envFile.variables[i].Name] = envFile.variables[i].Location
	}
	return nil
}

// expandValue replaces $VAR and ${VAR} references in the value
func expandValue(value string, varMap map[string]string, locMap map[string]Location, expanded map[string]Location) (string, error) {
	var result strings.Builder
	result.Grow(len(value))

	for i := 0; i < len(value); i++ {
		if value[i] == '$' && i+1 < len(value) {
			if value[i+1] == '{' {
				// ${VARIABLE} syntax with possible default/replacement value
				endIdx := strings.Index(value[i+2:], "}")
				if endIdx != -1 {
					content := value[i+2 : i+2+endIdx]

					// Check for ${VAR:?error} (error if unset or empty)
					if colonQuestionIdx := strings.Index(content, ":?"); colonQuestionIdx != -1 {
						varName := content[:colonQuestionIdx]
						errorMsg := content[colonQuestionIdx+2:]
						val, ok := varMap[varName]
						if !ok || val == "" {
							if errorMsg == "" {
								return "", fmt.Errorf("%s: required variable is not set", varName)
							}
							return "", fmt.Errorf("%s", errorMsg)
						}
						result.WriteString(val)
						expanded[varName] = locMap[varName]
					} else if colonDashIdx := strings.Index(content, ":-"); colonDashIdx != -1 {
						// Check for ${VAR:-default} (use default if unset or empty)
						varName := content[:colonDashIdx]
						defaultValue := content[colonDashIdx+2:]
						if val, ok := varMap[varName]; ok && val != "" {
							result.WriteString(val)
							expanded[varName] = locMap[varName]
						} else {
							result.WriteString(defaultValue)
						}
					} else if colonPlusIdx := strings.Index(content, ":+"); colonPlusIdx != -1 {
						// Check for ${VAR:+replacement} (use replacement if set and non-empty)
						varName := content[:colonPlusIdx]
						replacement := content[colonPlusIdx+2:]
						if val, ok := varMap[varName]; ok && val != "" {
							result.WriteString(replacement)
							expanded[varName] = locMap[varName]
						}
						// Otherwise leave empty
					} else if questionIdx := strings.Index(content, "?"); questionIdx != -1 {
						// Check for ${VAR?error} (error if unset, but can be empty)
						varName := content[:questionIdx]
						errorMsg := content[questionIdx+1:]
						if _, ok := varMap[varName]; !ok {
							if errorMsg == "" {
								return "", fmt.Errorf("%s: required variable is not set", varName)
							}
							return "", fmt.Errorf("%s", errorMsg)
						}
						result.WriteString(varMap[varName])
						expanded[varName] = locMap[varName]
					} else if dashIdx := strings.Index(content, "-"); dashIdx != -1 {
						// Check for ${VAR-default} (use default if unset)
						varName := content[:dashIdx]
						defaultValue := content[dashIdx+1:]
						if val, ok := varMap[varName]; ok {
							result.WriteString(val)
							expanded[varName] = locMap[varName]
						} else {
							result.WriteString(defaultValue)
						}
					} else if plusIdx := strings.Index(content, "+"); plusIdx != -1 {
						// Check for ${VAR+replacement} (use replacement if set)
						varName := content[:plusIdx]
						replacement := content[plusIdx+1:]
						if _, ok := varMap[varName]; ok {
							result.WriteString(replacement)
							expanded[varName] = locMap[varName]
						}
						// Otherwise leave empty
					} else {
						// Simple ${VAR} syntax
						if val, ok := varMap[content]; ok {
							result.WriteString(val)
							expanded[content] = locMap[content]
						}
						// If variable not found, leave it empty (standard behavior)
					}
					i = i + 2 + endIdx // skip past the }
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
				if val, ok := varMap[varName]; ok {
					result.WriteString(val)
					expanded[varName] = locMap[varName]
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

	return result.String(), nil
}

// isVarNameChar returns true if the character is valid in a variable name
func isVarNameChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
}

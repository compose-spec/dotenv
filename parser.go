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
		Variables: []Variable{},
	}

	scanner := bufio.NewScanner(reader)
	lineNumber := 0
	// Track defined variable names
	definedVars := make(map[string]bool)

	for scanner.Scan() {
		lineNumber++

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := scanner.Text()
		originalLine := line

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove 'export ' prefix if present
		isExportLine := strings.HasPrefix(line, "export ")
		if isExportLine {
			line = line[7:]
		}

		// Find the separator (= or :)
		equalIdx := strings.Index(line, "=")
		colonIdx := strings.Index(line, ":")

		var separatorIdx int
		if equalIdx == -1 && colonIdx == -1 {
			// No separator found
			// Allow "export VARIABLE" if VARIABLE is already defined
			if isExportLine {
				varName := strings.TrimSpace(line)
				if definedVars[varName] {
					// Valid export of existing variable, skip line
					continue
				}
				return nil, fmt.Errorf("line %d %q has an unset variable", lineNumber, varName)
			}
			return nil, fmt.Errorf("line %d: no separator found in line: %s", lineNumber, originalLine)
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

		// Validate variable name - must match [A-Za-z0-9_.-]
		if !isValidVariableName(name) {
			return nil, fmt.Errorf("line %d: invalid variable name %q", lineNumber, name)
		}

		// Handle inline comments: strip # comment from unquoted values
		// But preserve # in quoted values
		quoteStyle := Unquoted
		if len(value) > 0 && value[0] != '"' && value[0] != '\'' {
			// Unquoted value: look for # comment marker
			if commentIdx := strings.Index(value, "#"); commentIdx != -1 {
				value = strings.TrimSpace(value[:commentIdx])
			}
		}

		// Handle multi-line quoted values
		if len(value) > 0 && (value[0] == '"' || value[0] == '\'') {
			quoteChar := value[0]
			// Check if quote is closed on the same line
			closingQuoteIdx := -1
			for i := 1; i < len(value); i++ {
				if value[i] == quoteChar {
					// Check if it's escaped (for double quotes)
					if quoteChar == '"' && i > 0 && value[i-1] == '\\' {
						continue
					}
					closingQuoteIdx = i
					break
				}
			}

			// If quote is not closed, read more lines
			if closingQuoteIdx == -1 {
				var multilineValue strings.Builder
				multilineValue.WriteString(value)

				for scanner.Scan() {
					lineNumber++
					nextLine := scanner.Text()
					multilineValue.WriteString("\n")
					multilineValue.WriteString(nextLine)

					// Look for closing quote in this line
					for i := 0; i < len(nextLine); i++ {
						if nextLine[i] == quoteChar {
							// Check if it's escaped (for double quotes)
							if quoteChar == '"' && i > 0 && nextLine[i-1] == '\\' {
								continue
							}
							closingQuoteIdx = i
							break
						}
					}

					if closingQuoteIdx != -1 {
						break
					}
				}

				value = multilineValue.String()
			}
		}

		// Track quote style and remove surrounding quotes if present
		if len(value) >= 2 {
			if value[0] == '"' && value[len(value)-1] == '"' {
				// Double-quoted: remove quotes and process escape sequences
				quoteStyle = DoubleQuoted
				value = unescapeDoubleQuoted(value[1 : len(value)-1])
			} else if value[0] == '\'' && value[len(value)-1] == '\'' {
				// Single-quoted: just remove quotes, no escape processing
				quoteStyle = Quoted
				value = value[1 : len(value)-1]
			}
		}

		variable := Variable{
			Name:     name,
			RawValue: value,
			Location: Location(fmt.Sprintf(":%d", lineNumber)),
			Quoted:   quoteStyle,
			Expanded: make(map[string]Location),
		}

		envFile.Variables = append(envFile.Variables, variable)
		definedVars[name] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envFile, nil
}

// isValidVariableName returns true if the variable name matches [A-Za-z0-9_.-] and doesn't start with a digit
func isValidVariableName(name string) bool {
	if len(name) == 0 {
		return false
	}
	// First character cannot be a digit
	if name[0] >= '0' && name[0] <= '9' {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '.' || c == '-') {
			return false
		}
	}
	return true
}

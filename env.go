package dotenv

import "sort"

// Location tracks the source file and line number of an environment variable in the format "file:line"
type Location string

// Variable represents a single environment variable with its metadata
type Variable struct {
	Name     string
	Value    string
	RawValue string
	Location Location
	Expanded map[string]Location // tracks which variables were expanded and where they came from
}

// EnvFile represents a parsed .env file containing a list of variables
type EnvFile struct {
	variables []Variable
}

// Variables returns the environment variables as a map[string]string
func (e *EnvFile) Variables() map[string]string {
	result := make(map[string]string, len(e.variables))
	for _, variable := range e.variables {
		result[variable.Name] = variable.Value
	}
	return result
}

// Explain returns a detailed explanation of how a variable was set
func (e *EnvFile) Explain(name string) string {
	// Find the variable
	var variable *Variable
	for i := range e.variables {
		if e.variables[i].Name == name {
			variable = &e.variables[i]
			break
		}
	}

	if variable == nil {
		return "Variable not found"
	}

	explanation := "Variable: " + variable.Name + "\n"
	explanation += "Location: " + string(variable.Location) + "\n"
	explanation += "Raw Value: " + variable.RawValue + "\n"
	explanation += "Final Value: " + variable.Value + "\n"

	if len(variable.Expanded) > 0 {
		explanation += "Expanded from:\n"
		// Build a map of variable values for lookup
		varMap := make(map[string]string)
		for _, v := range e.variables {
			varMap[v.Name] = v.Value
		}
		// Sort variable names for deterministic output
		varNames := make([]string, 0, len(variable.Expanded))
		for varName := range variable.Expanded {
			varNames = append(varNames, varName)
		}
		sort.Strings(varNames)
		for _, varName := range varNames {
			explanation += "  - " + varName + "=" + varMap[varName] + " at " + string(variable.Expanded[varName]) + "\n"
		}
	}

	return explanation
}

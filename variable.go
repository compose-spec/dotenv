package dotenv

// Variable represents a single environment variable with its metadata
type Variable struct {
	Name     string
	Value    string
	RawValue string
	Location Location
	Quoted   QuoteStyle
	Expanded map[string]Location // tracks which variables were expanded and where they came from
}

// expandValue replaces $VAR and ${VAR} references in the value
func (v *Variable) expandValue(lookup LookupFn) error {
	val, exp, err := expandString(v.RawValue, lookup)
	if err != nil {
		return err
	}
	v.Value = val
	v.Expanded = exp
	return nil
}

package dotenv

import (
	"os"
	"sort"
)

// LookupFn is a function that looks up a variable by name and returns the Variable and whether it was found
type LookupFn func(string) (Variable, bool)

// CompositeLookup manages multiple LookupFn with explicit priorities
type CompositeLookup struct {
	lookups []prioritizedLookup
}

func WithPriority(Lookup LookupFn, priority int) prioritizedLookup {
	return prioritizedLookup{
		Lookup:   Lookup,
		Priority: priority,
	}
}

// NewCompositeLookup creates a new CompositeLookup with the given prioritized lookup functions
// Higher priority values are tried first
func NewCompositeLookup(lookups ...prioritizedLookup) *CompositeLookup {
	// Make a copy and sort by priority (highest first)
	sorted := make([]prioritizedLookup, len(lookups))
	copy(sorted, lookups)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})
	return &CompositeLookup{
		lookups: sorted,
	}
}

// PrioritizedLookup associates a lookup function with a priority value
type prioritizedLookup struct {
	Priority int
	Lookup   LookupFn
}

// Lookup implements the LookupFn signature by trying each lookup function in priority order
func (c *CompositeLookup) Lookup(name string) (Variable, bool) {
	for _, pl := range c.lookups {
		if v, ok := pl.Lookup(name); ok {
			return v, true
		}
	}
	return Variable{}, false
}

// Lookup implements the LookupFn signature by looking up variables in the OS environment
var OSEnv LookupFn = func(name string) (Variable, bool) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return Variable{}, false
	}
	return Variable{
		Name:     name,
		Value:    value,
		RawValue: value,
		Location: ":os",
	}, true
}

// Package flagx holds the command-line flag types shared by antislop
// analyzers whose options are lists of names.
package flagx

import "strings"

// List binds a comma-separated command-line flag to a []string option.
// Setting the flag replaces the list rather than adding to it, so a value
// given on the command line (or in the host's settings) fully describes what
// the analyzer looks for.
type List struct{ target *[]string }

// NewList returns a flag.Value that writes the parsed list to target.
func NewList(target *[]string) *List { return &List{target: target} }

// String renders the current list the way Set accepts it.
func (l *List) String() string {
	if l == nil || l.target == nil {
		return ""
	}
	return strings.Join(*l.target, ",")
}

// Set replaces the list with the comma-separated entries of value. Blank
// entries are dropped, so "a, ,b" and "a,b" mean the same thing.
func (l *List) Set(value string) error {
	out := []string{}
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	*l.target = out
	return nil
}

// Package cli provides a set of utilities for building command-line interfaces (CLI) in Go. It includes support for parsing command-line arguments, flags, and options, as well as handling user input and output. The package is designed to be flexible and extensible, allowing developers to create custom CLI configurations
package cli

import (
	"slices"
	"strings"
)

type OptionalStringMap map[string]string

func (m *OptionalStringMap) String() string {
	if m == nil || len(*m) == 0 {
		return ""
	}

	keys := make([]string, 0, len(*m))
	for k := range *m {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		v := (*m)[k]
		if v != "" {
			pairs = append(pairs, k+"="+v)
		} else {
			pairs = append(pairs, k)
		}
	}
	return strings.Join(pairs, ",")
}

func (m *OptionalStringMap) Type() string {
	return "key[=value]"
}

func (m *OptionalStringMap) Set(val string) error {
	if *m == nil {
		*m = make(OptionalStringMap)
	}

	for item := range strings.SplitSeq(val, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		key, value, _ := strings.Cut(item, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		(*m)[key] = value
	}
	return nil
}

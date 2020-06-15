// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"strings"
)

const (
	filterTypeSimple = iota
	filterTypeJQ
)

type filter struct {
	ftype      int
	simpleSpec filterSimple
	priority   int
}

func (f *filter) filter(line map[string]interface{}) (map[string]interface{}, error) {
	switch f.ftype {
	case filterTypeSimple:
		if f.simpleSpec.isMatch(line) {
			return line, nil
		}
		return nil, nil
	}
	panic("BUG: invalid filter type")
}

func determineFilterType(spec string) int {
	return filterTypeSimple
}

type filterSimple struct {
	filename     string
	components   []string
	messageTypes []string
}

func parseSimpleFilter(filterexpr string) (*filter, error) {
	var (
		res   filterSimple
		parts = strings.SplitN(filterexpr, ":", 3)
	)
	switch len(parts) {
	// Only a filename ist specified, no filters.
	case 1:
		res.filename = parts[0]
	// Filters and filename is availabe.
	case 2:
		res.messageTypes = removeEmpy(strings.Split(parts[0], ","))
		res.filename = parts[1]
	// Components, filters, and a filename specified.
	case 3:
		res.components = removeEmpy(strings.Split(parts[0], ","))
		res.messageTypes = removeEmpy(strings.Split(parts[1], ","))
		res.filename = parts[2]
	// Filter expression is invalid.
	default:
		return nil, fmt.Errorf("invalid filter expression")
	}
	return &filter{ftype: filterTypeSimple, simpleSpec: res}, nil
}

func compare(candidate string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	c := strings.ToLower(candidate)
	for _, filter := range filters {
		f := strings.ToLower(filter)
		if c == f {
			return true
		}
	}
	return false
}

// FIXME: exclusive is broken, thus missing
func (f *filterSimple) isMatch(data map[string]interface{}) bool {
	comp, err := castField(data, "component")
	if err != nil {
		return false
	}
	msgType, err := castField(data, "type")
	if err != nil {
		return false
	}
	if !compare(comp, f.components) {
		return false
	}
	if !compare(msgType, f.messageTypes) {
		return false
	}
	return true
}

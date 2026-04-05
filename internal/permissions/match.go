package permissions

import (
	"path"
	"regexp"
	"strings"
)

// matchElement checks whether a single pattern element matches a single arg.
func matchElement(elem PatternElement, arg string) bool {
	switch elem.Type {
	case ElementLiteral:
		if strings.Contains(elem.Value, "*") || strings.Contains(elem.Value, "?") || strings.Contains(elem.Value, ".") {
			matchPattern := strings.ReplaceAll(elem.Value, ".", "?")
			matched, _ := path.Match(matchPattern, arg)
			return matched
		}
		return arg == elem.Value
	case ElementSingleChar:
		return len(arg) == 1
	case ElementWildcard:
		return true
	case ElementQuoted:
		return elem.Value == arg
	case ElementRegex:
		re, err := regexp.Compile(elem.Value)
		if err != nil {
			return false
		}
		return re.MatchString(arg)
	default:
		return false
	}
}

// matchGroup tries to match group sub-elements against consecutive args starting at startIdx.
// Returns the number of args consumed, or -1 if no match.
func matchGroup(group []PatternElement, args []string, startIdx int) int {
	idx := startIdx
	for _, sub := range group {
		if idx >= len(args) || !matchElement(sub, args[idx]) {
			return -1
		}
		idx++
	}
	return idx - startIdx
}

// allUnconsumed returns true if all indices from start to start+count-1 are unconsumed.
func allUnconsumed(consumed map[int]bool, start, count int) bool {
	for i := start; i < start+count; i++ {
		if consumed[i] {
			return false
		}
	}
	return true
}

// markConsumed marks indices from start to start+count-1 as consumed.
func markConsumed(consumed map[int]bool, start, count int) {
	for i := start; i < start+count; i++ {
		consumed[i] = true
	}
}

// MatchPattern checks whether args match the given pattern.
// It walks pattern elements and args in parallel.
// Default behavior is prefix matching: if all pattern elements are
// consumed, extra args are allowed.
func MatchPattern(pattern *ArgPattern, args []string) bool {
	// Track which arg indices have been consumed by non-positional elements.
	consumed := make(map[int]bool)

	argIdx := 0
	for elemIdx := 0; elemIdx < len(pattern.Elements); elemIdx++ {
		elem := pattern.Elements[elemIdx]

		if elem.NonPositional {
			if elem.Type == ElementGroup {
				// Non-positional group: search for contiguous match anywhere.
				found := false
				groupLen := len(elem.Group)
				if groupLen > 0 {
					for i := argIdx; i <= len(args)-groupLen; i++ {
						if allUnconsumed(consumed, i, groupLen) {
							n := matchGroup(elem.Group, args, i)
							if n > 0 {
								markConsumed(consumed, i, n)
								found = true
								break
							}
						}
					}
				}
				if !found {
					if elem.Optional {
						continue
					}
					return false
				}
				continue
			}
			// Search all remaining args (from argIdx onward) for a match.
			found := false
			for i := argIdx; i < len(args); i++ {
				if !consumed[i] && matchElement(elem, args[i]) {
					consumed[i] = true
					found = true
					break
				}
			}
			if !found {
				if elem.Optional {
					continue
				}
				return false
			}
			continue
		}

		// Skip consumed args for positional matching.
		for argIdx < len(args) && consumed[argIdx] {
			argIdx++
		}

		// Handle positional groups.
		if elem.Type == ElementGroup {
			n := matchGroup(elem.Group, args, argIdx)
			if n >= 0 {
				argIdx += n
			} else if !elem.Optional {
				return false
			}
			continue
		}

		// Optional positional elements: try to match, skip if no match.
		if elem.Optional {
			if argIdx < len(args) && matchElement(elem, args[argIdx]) {
				argIdx++
			}
			// Either way, continue to next element.
			continue
		}

		switch elem.Type {
		case ElementLiteral:
			if argIdx >= len(args) {
				return false
			}
			if !matchElement(elem, args[argIdx]) {
				return false
			}
			argIdx++
		case ElementSingleChar:
			if argIdx >= len(args) {
				return false
			}
			if len(args[argIdx]) != 1 {
				return false
			}
			argIdx++
		case ElementWildcard:
			if argIdx >= len(args) {
				return false
			}
			argIdx++
		case ElementQuoted:
			if argIdx >= len(args) {
				return false
			}
			if args[argIdx] != elem.Value {
				return false
			}
			argIdx++
		case ElementDoubleWildcard:
			// ** matches zero or more remaining args.
			// If last element, consume everything.
			if elemIdx == len(pattern.Elements)-1 {
				return true
			}
			// If not last, try matching remaining elements at every position.
			remaining := &ArgPattern{
				Elements:   pattern.Elements[elemIdx+1:],
				ExactMatch: pattern.ExactMatch,
			}
			for tryIdx := argIdx; tryIdx <= len(args); tryIdx++ {
				if MatchPattern(remaining, args[tryIdx:]) {
					return true
				}
			}
			return false
		case ElementRegex:
			if argIdx >= len(args) {
				return false
			}
			re, err := regexp.Compile(elem.Value)
			if err != nil {
				return false
			}
			if !re.MatchString(args[argIdx]) {
				return false
			}
			argIdx++
		case ElementRegexMulti:
			// /regex/** scans all remaining args for at least one match.
			if argIdx >= len(args) {
				return false
			}
			re, err := regexp.Compile(elem.Value)
			if err != nil {
				return false
			}
			found := false
			for _, arg := range args[argIdx:] {
				if re.MatchString(arg) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
			// Consume all remaining args — regex multi is a greedy scanner.
			return true
		default:
			return false
		}
	}
	if pattern.ExactMatch {
		// Count unconsumed args.
		unconsumed := 0
		for i := argIdx; i < len(args); i++ {
			if !consumed[i] {
				unconsumed++
			}
		}
		return unconsumed == 0
	}
	return true
}

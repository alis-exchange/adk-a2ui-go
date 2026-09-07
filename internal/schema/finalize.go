package schema

import "sort"

// Finalize drops problems whose (Path, Message) pair already appeared, keeping the first
// occurrence, and sorts the result stably by Path. Two independent passes (e.g. a version check
// and the schema formatter) can report the same finding for one message, and every caller's
// return path should apply this the same way so the returned error is deterministic regardless
// of which pass produced the problems.
func Finalize(problems []Problem) []Problem {
	seen := make(map[Problem]bool, len(problems))
	out := problems[:0:0]
	for _, p := range problems {
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

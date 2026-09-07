package schema

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// maxProblems caps the list handed to the model: past twenty findings the list stops being a
// fix-list and starts being noise, and the trailer tells the model more remain.
const maxProblems = 20

// Finalize is the single exit path every validator uses to turn collected problems into the
// list the model sees. It drops problems whose (Path, Message) pair already appeared, keeping
// the first occurrence; sorts what is left into the order the paths appear in the batch (see
// [naturalLess]); and caps the result at maxProblems, appending a "... and N more" trailer when
// it had to cut. Two independent passes (e.g. a version check and the schema formatter) can
// report the same finding for one message, and every caller's return path applies this the same
// way so the returned error is deterministic regardless of which pass produced the problems.
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
	sort.SliceStable(out, func(i, j int) bool { return naturalLess(out[i].Path, out[j].Path) })
	if len(out) > maxProblems {
		extra := len(out) - maxProblems
		out = append(out[:maxProblems:maxProblems], Problem{Message: fmt.Sprintf("... and %d more", extra)})
	}
	return out
}

// naturalLess orders two rendered paths the way a reader scans the batch: segment by segment,
// with array indices compared as numbers. Plain lexical order would put "messages[10]" before
// "messages[2]" and "components[10]" before "components[9]", so a long batch's problem list
// would jump around instead of running front to back.
func naturalLess(a, b string) bool {
	ta, tb := pathTokens(a), pathTokens(b)
	for i := 0; i < len(ta) && i < len(tb); i++ {
		x, y := ta[i], tb[i]
		switch {
		case x.isIndex && y.isIndex:
			if x.index != y.index {
				return x.index < y.index
			}
		case x.isIndex != y.isIndex:
			return x.isIndex // an index sorts before a property name at the same depth
		default:
			if x.name != y.name {
				return x.name < y.name
			}
		}
	}
	return len(ta) < len(tb)
}

// pathToken is one segment of a rendered path: a property name, or an array index.
type pathToken struct {
	name    string
	index   int
	isIndex bool
}

// pathTokens splits a rendered path into its segments: "messages[10].updateComponents" becomes
// ["messages", 10, "updateComponents"]. A bracketed segment that is not a number (nothing the
// formatter emits today) is kept as a name, so ordering stays defined for any input.
func pathTokens(p string) []pathToken {
	var out []pathToken
	for i := 0; i < len(p); {
		switch p[i] {
		case '.':
			i++
		case '[':
			end := strings.IndexByte(p[i:], ']')
			if end < 0 {
				return append(out, pathToken{name: p[i:]})
			}
			seg := p[i+1 : i+end]
			if n, err := strconv.Atoi(seg); err == nil {
				out = append(out, pathToken{index: n, isIndex: true})
			} else {
				out = append(out, pathToken{name: seg})
			}
			i += end + 1
		default:
			next := strings.IndexAny(p[i:], ".[")
			if next < 0 {
				return append(out, pathToken{name: p[i:]})
			}
			out = append(out, pathToken{name: p[i : i+next]})
			i += next
		}
	}
	return out
}

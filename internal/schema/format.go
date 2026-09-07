package schema

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const maxProblems = 20

var printer = message.NewPrinter(language.English)

// Format turns a validator error into model-facing problems. instance is the value that was
// validated and prefix is its rendered path ("messages", or "messages[1].updateComponents.components[0]"
// for a single component). oneOf fan-outs are pruned to the branch matching the instance's
// "component" (catalog components) or present message key (envelopes).
func Format(err error, instance any, prefix string) []Problem {
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []Problem{{Path: prefix, Message: err.Error()}}
	}
	f := &formatter{instance: instance, prefix: prefix, seen: map[string]bool{}}
	f.walk(ve)
	if len(f.problems) == 0 {
		f.problems = []Problem{{Path: prefix, Message: strings.SplitN(ve.Error(), "\n", 2)[0]}}
	}
	if len(f.problems) > maxProblems {
		extra := len(f.problems) - maxProblems
		f.problems = append(f.problems[:maxProblems], Problem{Message: fmt.Sprintf("... and %d more", extra)})
	}
	return f.problems
}

type formatter struct {
	instance any
	prefix   string
	problems []Problem
	seen     map[string]bool
}

func (f *formatter) walk(e *jsonschema.ValidationError) {
	switch k := e.ErrorKind.(type) {
	case *kind.OneOf:
		if f.walkOneOf(e) {
			return
		}
	case *kind.Required:
		for _, m := range k.Missing {
			f.add(e.InstanceLocation, fmt.Sprintf("missing property %q", m))
		}
		return
	case *kind.FalseSchema:
		// unevaluatedProperties:false reports the offending property as a false schema at its own location.
		if n := len(e.InstanceLocation); n > 0 {
			f.add(e.InstanceLocation[:n-1], fmt.Sprintf("unknown property %q", e.InstanceLocation[n-1]))
			return
		}
	case *kind.AdditionalProperties:
		f.add(e.InstanceLocation, "unknown properties "+quoteList(k.Properties))
		return
	case *kind.Const:
		f.add(e.InstanceLocation, "must be "+jsonText(k.Want))
		return
	case *kind.Enum:
		f.add(e.InstanceLocation, "must be one of "+jsonList(k.Want))
		return
	case *kind.Type:
		f.add(e.InstanceLocation, fmt.Sprintf("must be of type %s, got %s", strings.Join(k.Want, " or "), k.Got))
		return
	}
	if len(e.Causes) == 0 {
		f.add(e.InstanceLocation, e.ErrorKind.LocalizedString(printer))
		return
	}
	for _, c := range pruneCollateralFalseSchema(e.Causes) {
		f.walk(c)
	}
}

// pruneCollateralFalseSchema drops *kind.FalseSchema causes that are collateral fallout from a
// sibling failure in the same set, rather than a real "unknown property" finding.
//
// The catalog schemas bundle a component's own required/const/enum checks and its
// unevaluatedProperties:false check as siblings under one allOf branch. When a sibling
// keyword (required, const, enum, type, ...) fails, the whole clause is not satisfied, so none
// of its declared properties count as "evaluated" -- unevaluatedProperties then reports every
// declared property in that clause as false schema too, alongside the genuine failure. That
// makes known properties (e.g. the "component" discriminator itself) look unknown.
//
// The only case where a set of sibling FalseSchema causes is a genuine unknown-property finding
// is when there is no other kind of failure alongside it: a lone (or all-FalseSchema) cause set
// means every other keyword in the clause was satisfied and the flagged property really is
// undeclared.
func pruneCollateralFalseSchema(causes []*jsonschema.ValidationError) []*jsonschema.ValidationError {
	hasOther := false
	for _, c := range causes {
		if _, ok := c.ErrorKind.(*kind.FalseSchema); !ok {
			hasOther = true
			break
		}
	}
	if !hasOther {
		return causes
	}
	out := causes[:0:0]
	for _, c := range causes {
		if _, ok := c.ErrorKind.(*kind.FalseSchema); !ok {
			out = append(out, c)
		}
	}
	return out
}

// walkOneOf selects the branch that matches the instance. Returns false when no selection
// strategy applies, so the caller falls back to walking every cause.
func (f *formatter) walkOneOf(e *jsonschema.ValidationError) bool {
	inst, ok := f.valueAt(e.InstanceLocation).(map[string]any)
	if !ok {
		return false
	}
	var branchNames []string
	for _, c := range e.Causes {
		ref, ok := c.ErrorKind.(*kind.Reference)
		if !ok {
			return false
		}
		name := path.Base(ref.URL)
		branchNames = append(branchNames, name)
		if comp, _ := inst["component"].(string); comp != "" && name == comp {
			f.walk(c)
			return true
		}
		if key, isMsg := strings.CutSuffix(name, "Message"); isMsg {
			if _, present := inst[lowerFirst(key)]; present {
				f.walk(c)
				return true
			}
		}
	}
	sort.Strings(branchNames)
	if comp, _ := inst["component"].(string); comp != "" {
		f.add(e.InstanceLocation, fmt.Sprintf("unknown component %q (catalog components: %s)", comp, strings.Join(branchNames, ", ")))
		return true
	}
	if strings.HasSuffix(branchNames[0], "Message") {
		var keys, unknown []string
		for _, n := range branchNames {
			keys = append(keys, lowerFirst(strings.TrimSuffix(n, "Message")))
		}
		for k := range inst {
			if k != "version" && !contains(keys, k) {
				unknown = append(unknown, k)
			}
		}
		sort.Strings(unknown)
		msg := "a message must contain exactly one of " + quoteList(keys)
		if len(unknown) > 0 {
			msg = "unknown message key " + quoteList(unknown) + "; " + msg
		}
		f.add(e.InstanceLocation, msg)
		return true
	}
	return false
}

func (f *formatter) add(loc []string, msg string) {
	p := Problem{Path: renderPath(f.prefix, loc), Message: msg}
	key := p.Path + "\x00" + p.Message
	if f.seen[key] {
		return
	}
	f.seen[key] = true
	f.problems = append(f.problems, p)
}

func (f *formatter) valueAt(loc []string) any {
	cur := f.instance
	for _, seg := range loc {
		switch v := cur.(type) {
		case map[string]any:
			cur = v[seg]
		case []any:
			i, err := strconv.Atoi(seg)
			if err != nil || i < 0 || i >= len(v) {
				return nil
			}
			cur = v[i]
		default:
			return nil
		}
	}
	return cur
}

func renderPath(prefix string, loc []string) string {
	var b strings.Builder
	b.WriteString(prefix)
	for _, seg := range loc {
		if _, err := strconv.Atoi(seg); err == nil {
			b.WriteString("[" + seg + "]")
			continue
		}
		if b.Len() > 0 {
			b.WriteString(".")
		}
		b.WriteString(seg)
	}
	return b.String()
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func quoteList(items []string) string {
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = strconv.Quote(s)
	}
	return strings.Join(q, ", ")
}

func jsonText(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func jsonList(vs []any) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = jsonText(v)
	}
	return strings.Join(parts, ", ")
}

package v10

// Function caller roles, as catalog_definition.json's allowedCallers names them.
const (
	callerRendererOnly    = "rendererOnly"
	callerAgentOnly       = "agentOnly"
	callerRendererOrAgent = "rendererOrAgent"
)

// allowedCallers returns the allowedCallers a v1.0 catalog declares for the named function, or
// "" when the catalog's functions object does not define that name (a hand-written anyFunction
// union the library cannot read a policy from). An absent value on a defined function is the
// spec default, rendererOnly.
func allowedCallers(catalog map[string]any, name string) string {
	fns, _ := catalog["functions"].(map[string]any)
	def, ok := fns[name].(map[string]any)
	if !ok {
		return ""
	}
	if v, ok := def["allowedCallers"].(string); ok && v != "" {
		return v
	}
	return callerRendererOnly
}

// agentMayCall reports whether the agent may send callRendererFunction for a function: only
// agentOnly and rendererOrAgent ones, per the spec; the default rendererOnly is not.
func agentMayCall(allowed string) bool {
	return allowed == callerAgentOnly || allowed == callerRendererOrAgent
}

// rendererMayCall reports whether the renderer may send callAgentFunction for a function:
// agentOnly functions are reserved for agent-initiated callRendererFunction.
func rendererMayCall(allowed string) bool {
	return allowed != callerAgentOnly
}

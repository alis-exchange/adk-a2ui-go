package toolkit

import (
	"fmt"
	"strconv"
	"strings"
)

// Description renders the generate tool's instructions for one negotiated version. The example
// uses the real catalog id when one is known so the model has nothing to guess.
func Description(s Spec, exampleCatalogID string) string {
	if exampleCatalogID == "" {
		exampleCatalogID = "<a catalogId from a2ui_catalog>"
	}
	keys := make([]string, len(s.MessageKeys))
	for i, k := range s.MessageKeys {
		keys[i] = strconv.Quote(k)
	}
	var notes strings.Builder
	for _, n := range s.Notes {
		notes.WriteString("- " + n + "\n")
	}
	return fmt.Sprintf(descriptionTemplate, s.Version, strings.Join(keys, ", "), notes.String(), s.Version, exampleCatalogID, s.Version)
}

const descriptionTemplate = `Renders interactive UI in the chat by sending validated A2UI %[1]s messages to the client. This is the ONLY way to display structured UI (forms, cards, dashboards, buttons, etc.). Do not describe UI in plain text when this tool is available.

WHEN TO USE:
- The user asks to show, build, display, or update a UI, form, widget, or visual layout.
- You need to create a surface, change its components, update bound data, or remove it.

WHEN NOT TO USE:
- A text-only answer is sufficient.
- You have not called a2ui_catalog yet. It tells you the component names, their properties, and the valid catalogId values.
- A previous call returned status "success" for the same UI task. Your work is done.

WORKFLOW:
1. Call a2ui_catalog and read the catalogs and instructions it returns.
2. Call this tool once with a messages array. Every message MUST include "version": "%[1]s" and exactly one of %[2]s.
3. On status "success", stop. Do not call again for the same UI.
4. On error, fix exactly the problems listed and retry with the corrected messages.

NEW SURFACE (send together in one call):
1. createSurface with a surfaceId and catalogId.
2. updateComponents for the same surfaceId with a component tree that includes a component with "id": "root".

CRITICAL: the client mounts the component whose id is "root". A surface without one renders nothing.

UPDATING EXISTING UI:
- Rebuild layout: updateComponents (include id "root" again when replacing the tree).
- Change bound values only: updateDataModel.
- Remove UI: deleteSurface.

VERSION NOTES (%[4]s):
%[3]s
Example (new surface, minimal):
{
  "messages": [
    {"version": "%[6]s", "createSurface": {"surfaceId": "signup-form", "catalogId": "%[5]s"}},
    {"version": "%[6]s", "updateComponents": {"surfaceId": "signup-form", "components": [
      {"id": "root", "component": "Card", "child": "title"},
      {"id": "title", "component": "Text", "text": "Sign up"}
    ]}}
  ]
}
`

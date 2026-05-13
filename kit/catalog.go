package kit

import "encoding/json"

// GetCatalogs extracts catalog-related fields from an A2UI extension params map (or similar
// capability object):
//
//   - supportedCatalogIds: optional array of catalog ID strings.
//
//   - inlineCatalogs: optional array of arbitrary JSON objects; each element is re-marshaled and
//     unmarshaled into map[string]any for a stable map shape.
//
// Either slice may be empty if the key is missing or not of the expected type. An error is
// returned only if JSON remarshal of an inline catalog fails.
func GetCatalogs(capabilities map[string]any) ([]string, []map[string]any, error) {
	var supportedCatalogIds []string
	if supportedCatalogs, ok := capabilities["supportedCatalogIds"].([]any); ok {
		for _, catalogIDAny := range supportedCatalogs {
			if catalogID, ok := catalogIDAny.(string); ok {
				supportedCatalogIds = append(supportedCatalogIds, catalogID)
			}
		}
	}

	var catalogs []map[string]any
	if inlineCatalogs, ok := capabilities["inlineCatalogs"].([]any); ok {
		for _, inlineCatalogAny := range inlineCatalogs {
			inlineCatalogBytes, err := json.Marshal(inlineCatalogAny)
			if err != nil {
				return nil, nil, err
			}

			var inlineCatalog map[string]any
			if err := json.Unmarshal(inlineCatalogBytes, &inlineCatalog); err != nil {
				return nil, nil, err
			}

			catalogs = append(catalogs, inlineCatalog)
		}
	}
	return supportedCatalogIds, catalogs, nil
}

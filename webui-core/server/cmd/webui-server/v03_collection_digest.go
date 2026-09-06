package main

import "encoding/json"

// MarshalJSON intentionally serializes only the stable collection payload.
// Preview/apply control metadata (mode, preview token and confirmation) is
// server-owned and must not change the digest or the adapter request payload.
// The adapter receives the operation name separately and revalidates all typed
// record values from this canonical request file.
func (r v03CollectionRequest) MarshalJSON() ([]byte, error) {
	type stableCollectionPayload struct {
		Name    string           `json:"name"`
		Records []map[string]any `json:"records"`
	}
	return json.Marshal(stableCollectionPayload{Name: r.Name, Records: r.Records})
}

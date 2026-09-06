package main

import (
	"encoding/json"
	"testing"
)

func int64ptr(value int64) *int64 { return &value }

func testV03Collection() v03CollectionDefinition {
	return v03CollectionDefinition{
		Name:        "profiles",
		Label:       "Profiles",
		Risk:        "caution",
		IdentityKey: "name",
		MaxRecords:  4,
		Fields: []v03FieldDefinition{
			{Key: "name", Label: "Name", Type: "string", Required: true, MaxLength: 32, Pattern: `^[a-z][a-z0-9_-]*$`, ExportPolicy: "public"},
			{Key: "enabled", Label: "Enabled", Type: "boolean", Required: true, ExportPolicy: "public"},
			{Key: "port", Label: "Port", Type: "integer", Required: true, Min: int64ptr(1), Max: int64ptr(65535), ExportPolicy: "public"},
			{Key: "mode", Label: "Mode", Type: "enum", Required: true, ExportPolicy: "public", Options: []optionDefinition{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}},
		},
	}
}

func TestV03CollectionDefinition(t *testing.T) {
	if err := validateV03CollectionDefinition(testV03Collection()); err != nil {
		t.Fatalf("valid collection rejected: %v", err)
	}
	invalid := testV03Collection()
	invalid.Fields[0].Secret = true
	if err := validateV03CollectionDefinition(invalid); err == nil {
		t.Fatal("secret identity field must be rejected")
	}
}

func TestV03CollectionRecords(t *testing.T) {
	definition := testV03Collection()
	records := []map[string]any{
		{"name": "alpha", "enabled": true, "port": json.Number("22"), "mode": "a"},
		{"name": "beta", "enabled": false, "port": json.Number("2222"), "mode": "b"},
	}
	if _, err := validateV03Records(definition, records); err != nil {
		t.Fatalf("valid records rejected: %v", err)
	}
	duplicate := []map[string]any{
		{"name": "alpha", "enabled": true, "port": json.Number("22"), "mode": "a"},
		{"name": "alpha", "enabled": false, "port": json.Number("23"), "mode": "b"},
	}
	if _, err := validateV03Records(definition, duplicate); err == nil {
		t.Fatal("duplicate stable identity must be rejected")
	}
	unknown := []map[string]any{{"name": "alpha", "enabled": true, "port": json.Number("22"), "mode": "a", "command": "id"}}
	if _, err := validateV03Records(definition, unknown); err == nil {
		t.Fatal("undeclared field must be rejected")
	}
}

func TestV03CollectionDigestIgnoresControlMetadata(t *testing.T) {
	records := []map[string]any{{"name": "alpha", "enabled": true, "port": json.Number("22"), "mode": "a"}}
	preview := v03CollectionRequest{Name: "profiles", Mode: "preview", Records: records}
	apply := v03CollectionRequest{Name: "profiles", Mode: "apply", Records: records, PreviewToken: "0123456789abcdef0123456789abcdef", Confirmation: "APPLY"}
	previewJSON, err := json.Marshal(preview)
	if err != nil {
		t.Fatal(err)
	}
	applyJSON, err := json.Marshal(apply)
	if err != nil {
		t.Fatal(err)
	}
	if string(previewJSON) != string(applyJSON) {
		t.Fatalf("stable collection payload changed across preview/apply:\npreview=%s\napply=%s", previewJSON, applyJSON)
	}
	if string(previewJSON) != `{"name":"profiles","records":[{"enabled":true,"mode":"a","name":"alpha","port":22}]}` {
		t.Fatalf("unexpected canonical payload: %s", previewJSON)
	}
}

func TestV03ExportSecurityPolicy(t *testing.T) {
	valid := v03ExportDefinition{Name: "config", Label: "Config", Format: "json", Risk: "safe", MaxBytes: 4096, Filename: "config.json", SecretPolicy: "redacted"}
	if err := validateV03ExportDefinition(valid); err != nil {
		t.Fatalf("valid export rejected: %v", err)
	}
	unsafe := valid
	unsafe.SecretPolicy = "credential_material"
	if err := validateV03ExportDefinition(unsafe); err == nil {
		t.Fatal("credential material export policy must be rejected")
	}
	unsafe = valid
	unsafe.Filename = "../secret.json"
	if err := validateV03ExportDefinition(unsafe); err == nil {
		t.Fatal("path-bearing export filename must be rejected")
	}
}

func TestV03ImportBoundsAndConfirmation(t *testing.T) {
	valid := v03ImportDefinition{Name: "config", Label: "Config", Format: "json", Risk: "caution", MaxBytes: 65536, RequiresConfirmation: true, ConfirmationText: "IMPORT"}
	if err := validateV03ImportDefinition(valid); err != nil {
		t.Fatalf("valid import rejected: %v", err)
	}
	tooLarge := valid
	tooLarge.MaxBytes = maxV03UploadBytes + 1
	if err := validateV03ImportDefinition(tooLarge); err == nil {
		t.Fatal("oversized import declaration must be rejected")
	}
	missingConfirmation := valid
	missingConfirmation.ConfirmationText = ""
	if err := validateV03ImportDefinition(missingConfirmation); err == nil {
		t.Fatal("required confirmation without exact text must be rejected")
	}
}

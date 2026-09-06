package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestV04CapabilityValidation(t *testing.T) {
	app := &application{inventoryIndex: map[string]inventoryDefinition{"returns": {Name: "returns"}}}
	doc := v04CapabilityDocument{
		Schema:     v04CapabilitySchema,
		Module:     moduleDefinition{ID: "example"},
		Features:   map[string]bool{"typed_jobs": true, "inventory_operations": true, "references": true},
		References: []v04ReferenceDefinition{{Name: "target", SourceCollection: "targets", SourceIdentityKey: "name"}},
		Jobs: []v04JobDefinition{{
			Name: "collect", Risk: "caution", DedupeKeys: []string{"return_id"},
			Parameters: []v04ParameterDefinition{{Key: "return_id", Type: "string", Required: true, MaxLength: 64}},
		}},
		InventoryOperations: []v04InventoryOperationDefinition{{Name: "collect", Inventory: "returns", IdentityField: "return_id", Job: "collect", JobParameter: "return_id"}},
	}
	if err := validateV04Capabilities(doc, app); err != nil {
		t.Fatalf("valid document rejected: %v", err)
	}
	doc.Jobs[0].DedupeKeys = []string{"missing"}
	if err := validateV04Capabilities(doc, app); err == nil {
		t.Fatal("unknown dedupe key accepted")
	}
}

func TestV04ParameterValidation(t *testing.T) {
	min := int64(1)
	max := int64(10)
	parameter := v04ParameterDefinition{Key: "count", Type: "integer", Required: true, Min: &min, Max: &max}
	value, err := validateV04ParameterValue(parameter, json.Number("4"))
	if err != nil || value != int64(4) {
		t.Fatalf("integer validation failed value=%#v err=%v", value, err)
	}
	if _, err := validateV04ParameterValue(parameter, json.Number("11")); err == nil {
		t.Fatal("integer maximum not enforced")
	}
	text := v04ParameterDefinition{Key: "return_id", Type: "string", Required: true, MaxLength: 8, Pattern: "^[a-z0-9]+$"}
	if _, err := validateV04ParameterValue(text, "abc123"); err != nil {
		t.Fatalf("valid text rejected: %v", err)
	}
	if _, err := validateV04ParameterValue(text, "../bad"); err == nil {
		t.Fatal("pattern bypass accepted")
	}
}

func TestV04DedupeKeyStable(t *testing.T) {
	definition := v04JobDefinition{Name: "collect", DedupeKeys: []string{"return_id", "target"}}
	a := v04DedupeKey(definition, map[string]any{"return_id": "SDR-a", "target": "pi4", "ignored": "x"})
	b := v04DedupeKey(definition, map[string]any{"target": "pi4", "return_id": "SDR-a", "ignored": "y"})
	if a == "" || a != b {
		t.Fatalf("dedupe key is not stable: %q %q", a, b)
	}
	c := v04DedupeKey(definition, map[string]any{"return_id": "SDR-b", "target": "pi4"})
	if c == a {
		t.Fatal("changed identity reused dedupe key")
	}
}

func TestV04ParameterizedJobLifecycleAndDedupe(t *testing.T) {
	dir := t.TempDir()
	control := filepath.Join(dir, "control")
	script := "#!/bin/sh\nif [ \"$1\" = job-run-file ]; then test -f \"$3\" || exit 7; sleep 1; printf 'typed:%s\\n' \"$2\"; exit 0; fi\nexit 2\n"
	if err := os.WriteFile(control, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	app := &application{control: control, moduleDir: dir, stateDir: dir, runtimeDir: dir, jobTimeout: time.Minute, maxJobs: 2, jobs: make(map[string]*jobState)}
	definition := v04JobDefinition{Name: "collect", DedupeKeys: []string{"return_id"}}
	params := map[string]any{"return_id": "SDR-abc"}
	first, reused, digest, err := app.startV04Job(definition, params)
	if err != nil || reused || digest == "" {
		t.Fatalf("first job failed reused=%v digest=%q err=%v", reused, digest, err)
	}
	second, reused, digest2, err := app.startV04Job(definition, params)
	if err != nil || !reused || first != second || digest2 != digest {
		t.Fatalf("active duplicate not reused reused=%v same=%v digest=%q/%q err=%v", reused, first == second, digest, digest2, err)
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		record := first.snapshot()
		if record.Status == "success" {
			if got := string(first.stdout.Bytes()); got != "typed:collect\n" {
				t.Fatalf("unexpected output: %q", got)
			}
			return
		}
		if record.Status == "failed" {
			t.Fatalf("job failed: %s", record.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("typed job did not finish")
}

func TestV04ResolveInventoryItem(t *testing.T) {
	dir := t.TempDir()
	control := filepath.Join(dir, "control")
	script := "#!/bin/sh\nif [ \"$1\" = inventory ] && [ \"$2\" = returns ]; then printf '%s\\n' '{\"ok\":true,\"items\":[{\"return_id\":\"SDR-1\",\"state\":\"available\"}]}'; exit 0; fi\nexit 2\n"
	if err := os.WriteFile(control, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	app := &application{control: control, moduleDir: dir, stateDir: dir, runtimeDir: dir}
	item, err := app.resolveInventoryItem(context.Background(), "returns", "return_id", "SDR-1")
	if err != nil || item["state"] != "available" {
		t.Fatalf("inventory resolution failed item=%#v err=%v", item, err)
	}
	if _, err := app.resolveInventoryItem(context.Background(), "returns", "return_id", "SDR-missing"); err == nil {
		t.Fatal("stale inventory identity accepted")
	}
}

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	v03CapabilitySchema = "root-module-webui.extensions.v1"
	maxV03JSONBytes     = 256 << 10
	maxV03UploadBytes   = 1 << 20
	maxV03OutputBytes   = 1 << 20
	v03PreviewTTL       = 10 * time.Minute
)

type v03FieldDefinition struct {
	Key          string             `json:"key"`
	Label        string             `json:"label"`
	Description  string             `json:"description,omitempty"`
	Type         string             `json:"type"`
	Required     bool               `json:"required,omitempty"`
	Secret       bool               `json:"secret,omitempty"`
	ExportPolicy string             `json:"export_policy,omitempty"`
	Min          *int64             `json:"min,omitempty"`
	Max          *int64             `json:"max,omitempty"`
	MaxLength    int                `json:"max_length,omitempty"`
	Pattern      string             `json:"pattern,omitempty"`
	Options      []optionDefinition `json:"options,omitempty"`
}

type v03CollectionDefinition struct {
	Name                 string               `json:"name"`
	Label                string               `json:"label"`
	Description          string               `json:"description,omitempty"`
	Risk                 string               `json:"risk"`
	IdentityKey          string               `json:"identity_key"`
	MaxRecords           int                  `json:"max_records"`
	Fields               []v03FieldDefinition `json:"fields"`
	RequiresConfirmation bool                 `json:"requires_confirmation,omitempty"`
	ConfirmationText     string               `json:"confirmation_text,omitempty"`
}

type v03ImportDefinition struct {
	Name                 string `json:"name"`
	Label                string `json:"label"`
	Description          string `json:"description,omitempty"`
	Format               string `json:"format"`
	Risk                 string `json:"risk"`
	MaxBytes             int64  `json:"max_bytes"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"`
	ConfirmationText     string `json:"confirmation_text,omitempty"`
}

type v03ExportDefinition struct {
	Name                 string `json:"name"`
	Label                string `json:"label"`
	Description          string `json:"description,omitempty"`
	Format               string `json:"format"`
	Risk                 string `json:"risk"`
	MaxBytes             int    `json:"max_bytes"`
	Filename             string `json:"filename,omitempty"`
	SecretPolicy         string `json:"secret_policy"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"`
	ConfirmationText     string `json:"confirmation_text,omitempty"`
}

type v03CapabilityDocument struct {
	Schema      string                    `json:"schema"`
	Module      moduleDefinition          `json:"module"`
	Features    map[string]bool           `json:"features"`
	Collections []v03CollectionDefinition `json:"collections,omitempty"`
	Imports     []v03ImportDefinition     `json:"imports,omitempty"`
	Exports     []v03ExportDefinition     `json:"exports,omitempty"`
}

type v03CollectionRequest struct {
	Name         string           `json:"name"`
	Mode         string           `json:"mode"`
	Records      []map[string]any `json:"records"`
	PreviewToken string           `json:"preview_token,omitempty"`
	Confirmation string           `json:"confirmation,omitempty"`
}

type v03ImportApplyRequest struct {
	Name         string `json:"name"`
	PreviewToken string `json:"preview_token"`
	Confirmation string `json:"confirmation,omitempty"`
}

type v03ExportRequest struct {
	Name         string `json:"name"`
	Confirmation string `json:"confirmation,omitempty"`
}

type v03PreviewRecord struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Digest     string `json:"digest"`
	UploadPath string `json:"upload_path,omitempty"`
	ExpiresAt  string `json:"expires_at"`
}

func registerV03Handlers(mux *http.ServeMux, app *application) {
	mux.HandleFunc("/v03.js", app.requireSession(app.v03Asset))
	mux.HandleFunc("/api/v1/v03/capabilities", app.requireSession(app.v03CapabilitiesHandler))
	mux.HandleFunc("/api/v1/v03/collection", app.requireSession(app.v03CollectionHandler))
	mux.HandleFunc("/api/v1/v03/import", app.requireSession(app.v03ImportPreviewHandler))
	mux.HandleFunc("/api/v1/v03/import/apply", app.requireSession(app.v03ImportApplyHandler))
	mux.HandleFunc("/api/v1/v03/export", app.requireSession(app.v03ExportHandler))
}

func (a *application) v03Asset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, http.MethodGet, http.MethodHead)
		return
	}
	path := filepath.Join(a.webroot, "v03.js")
	data, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
}

func (a *application) v03CapabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	document, err := a.loadV03Capabilities(ctx)
	if err != nil {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "v0.3 extension capability disabled"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "extensions": document})
}

func (a *application) loadV03Capabilities(ctx context.Context) (v03CapabilityDocument, error) {
	var document v03CapabilityDocument
	output, err := a.runControl(ctx, maxControlOutput, "capabilities-v03")
	if err != nil {
		return document, err
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return document, fmt.Errorf("invalid v0.3 capability JSON: %w", err)
	}
	if document.Schema != v03CapabilitySchema {
		return document, errors.New("unsupported v0.3 capability schema")
	}
	if document.Module.ID != a.capabilities.Module.ID {
		return document, errors.New("v0.3 module id does not match base capabilities")
	}
	collectionNames := map[string]bool{}
	for _, definition := range document.Collections {
		if err := validateV03CollectionDefinition(definition); err != nil {
			return document, err
		}
		if collectionNames[definition.Name] {
			return document, fmt.Errorf("duplicate collection: %s", definition.Name)
		}
		collectionNames[definition.Name] = true
	}
	importNames := map[string]bool{}
	for _, definition := range document.Imports {
		if err := validateV03ImportDefinition(definition); err != nil {
			return document, err
		}
		if importNames[definition.Name] {
			return document, fmt.Errorf("duplicate import: %s", definition.Name)
		}
		importNames[definition.Name] = true
	}
	exportNames := map[string]bool{}
	for _, definition := range document.Exports {
		if err := validateV03ExportDefinition(definition); err != nil {
			return document, err
		}
		if exportNames[definition.Name] {
			return document, fmt.Errorf("duplicate export: %s", definition.Name)
		}
		exportNames[definition.Name] = true
	}
	return document, nil
}

func validateV03CollectionDefinition(definition v03CollectionDefinition) error {
	if !safeNamePattern.MatchString(definition.Name) {
		return fmt.Errorf("invalid collection name: %s", definition.Name)
	}
	if err := validateRisk(definition.Risk); err != nil {
		return fmt.Errorf("collection %s: %w", definition.Name, err)
	}
	if definition.MaxRecords < 1 || definition.MaxRecords > 128 {
		return fmt.Errorf("collection %s max_records must be 1..128", definition.Name)
	}
	if !safeNamePattern.MatchString(definition.IdentityKey) {
		return fmt.Errorf("collection %s has invalid identity_key", definition.Name)
	}
	if len(definition.Fields) == 0 || len(definition.Fields) > 32 {
		return fmt.Errorf("collection %s fields must be 1..32", definition.Name)
	}
	seen := map[string]bool{}
	identityFound := false
	for _, field := range definition.Fields {
		if err := validateV03FieldDefinition(field); err != nil {
			return fmt.Errorf("collection %s: %w", definition.Name, err)
		}
		if seen[field.Key] {
			return fmt.Errorf("collection %s has duplicate field %s", definition.Name, field.Key)
		}
		seen[field.Key] = true
		if field.Key == definition.IdentityKey {
			identityFound = true
			if field.Type != "string" || !field.Required || field.Secret {
				return fmt.Errorf("collection %s identity field must be required non-secret string", definition.Name)
			}
		}
	}
	if !identityFound {
		return fmt.Errorf("collection %s identity field is not declared", definition.Name)
	}
	return validateConfirmation(definition.RequiresConfirmation, definition.ConfirmationText)
}

func validateV03FieldDefinition(field v03FieldDefinition) error {
	if !safeNamePattern.MatchString(field.Key) {
		return fmt.Errorf("invalid field key: %s", field.Key)
	}
	switch field.Type {
	case "boolean", "integer", "string", "enum":
	default:
		return fmt.Errorf("unsupported field type: %s", field.Type)
	}
	if field.MaxLength < 0 || field.MaxLength > 4096 {
		return fmt.Errorf("invalid max_length for %s", field.Key)
	}
	if field.Pattern != "" {
		if _, err := regexp.Compile(field.Pattern); err != nil {
			return fmt.Errorf("invalid pattern for %s: %w", field.Key, err)
		}
	}
	switch field.ExportPolicy {
	case "", "public", "reference", "secret", "credential_material":
	default:
		return fmt.Errorf("invalid export_policy for %s", field.Key)
	}
	if field.Type == "enum" && len(field.Options) == 0 {
		return fmt.Errorf("enum field %s requires options", field.Key)
	}
	return nil
}

func validateV03ImportDefinition(definition v03ImportDefinition) error {
	if !safeNamePattern.MatchString(definition.Name) {
		return fmt.Errorf("invalid import name: %s", definition.Name)
	}
	if err := validateRisk(definition.Risk); err != nil {
		return fmt.Errorf("import %s: %w", definition.Name, err)
	}
	switch definition.Format {
	case "json", "zip":
	default:
		return fmt.Errorf("import %s has unsupported format", definition.Name)
	}
	if definition.MaxBytes < 1 || definition.MaxBytes > maxV03UploadBytes {
		return fmt.Errorf("import %s max_bytes out of range", definition.Name)
	}
	return validateConfirmation(definition.RequiresConfirmation, definition.ConfirmationText)
}

func validateV03ExportDefinition(definition v03ExportDefinition) error {
	if !safeNamePattern.MatchString(definition.Name) {
		return fmt.Errorf("invalid export name: %s", definition.Name)
	}
	if err := validateRisk(definition.Risk); err != nil {
		return fmt.Errorf("export %s: %w", definition.Name, err)
	}
	switch definition.Format {
	case "json", "zip":
	default:
		return fmt.Errorf("export %s has unsupported format", definition.Name)
	}
	if definition.MaxBytes < 1 || definition.MaxBytes > maxV03OutputBytes {
		return fmt.Errorf("export %s max_bytes out of range", definition.Name)
	}
	if definition.SecretPolicy == "" {
		definition.SecretPolicy = "redacted"
	}
	if definition.SecretPolicy != "redacted" && definition.SecretPolicy != "reference" {
		return fmt.Errorf("export %s secret_policy must be redacted or reference", definition.Name)
	}
	if definition.Filename != "" {
		if filepath.Base(definition.Filename) != definition.Filename || strings.ContainsAny(definition.Filename, "\r\n\"\\/") {
			return fmt.Errorf("export %s has unsafe filename", definition.Name)
		}
	}
	return validateConfirmation(definition.RequiresConfirmation, definition.ConfirmationText)
}

func validateConfirmation(required bool, text string) error {
	if !required {
		return nil
	}
	if text == "" || len(text) > 128 {
		return errors.New("invalid confirmation text")
	}
	return nil
}

func findV03Collection(document v03CapabilityDocument, name string) (v03CollectionDefinition, bool) {
	for _, definition := range document.Collections {
		if definition.Name == name {
			return definition, true
		}
	}
	return v03CollectionDefinition{}, false
}

func findV03Import(document v03CapabilityDocument, name string) (v03ImportDefinition, bool) {
	for _, definition := range document.Imports {
		if definition.Name == name {
			return definition, true
		}
	}
	return v03ImportDefinition{}, false
}

func findV03Export(document v03CapabilityDocument, name string) (v03ExportDefinition, bool) {
	for _, definition := range document.Exports {
		if definition.Name == name {
			return definition, true
		}
	}
	return v03ExportDefinition{}, false
}

func (a *application) v03CollectionHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	document, err := a.loadV03Capabilities(ctx)
	if err != nil || !document.Features["collections"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "collection capability disabled"})
		return
	}
	if r.Method == http.MethodGet {
		name := r.URL.Query().Get("name")
		if _, ok := findV03Collection(document, name); !ok {
			writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported collection"})
			return
		}
		output, err := a.runControl(ctx, maxInventoryOutput, "collection-get", name)
		if err != nil {
			a.controlError(w, err)
			return
		}
		writeValidatedJSON(w, output)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
		return
	}
	if !a.requireV03JSONMutation(w, r) {
		return
	}
	var request v03CollectionRequest
	if err := decodeV03JSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	definition, ok := findV03Collection(document, request.Name)
	if !ok {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported collection"})
		return
	}
	normalized, err := validateV03Records(definition, request.Records)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	request.Records = normalized
	requestPath, digest, err := a.writeV03NormalizedRequest("collection", request)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, responseEnvelope{OK: false, Error: "could not stage collection request"})
		return
	}
	defer os.Remove(requestPath)

	switch request.Mode {
	case "preview":
		output, err := a.runControl(ctx, maxControlOutput, "collection-preview", request.Name, requestPath)
		if err != nil {
			a.controlError(w, err)
			return
		}
		result, err := decodeControlJSON(output)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, responseEnvelope{OK: false, Error: "module backend returned invalid JSON"})
			return
		}
		token, err := a.writeV03Preview(v03PreviewRecord{Kind: "collection", Name: request.Name, Digest: digest, ExpiresAt: time.Now().Add(v03PreviewTTL).UTC().Format(time.RFC3339Nano)})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, responseEnvelope{OK: false, Error: "could not persist preview state"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "preview_token": token, "result": result})
	case "apply":
		if definition.RequiresConfirmation && request.Confirmation != definition.ConfirmationText {
			writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "confirmation text does not match"})
			return
		}
		preview, path, err := a.readV03Preview(request.PreviewToken)
		if err != nil || preview.Kind != "collection" || preview.Name != request.Name || preview.Digest != digest {
			writeJSON(w, http.StatusConflict, responseEnvelope{OK: false, Error: "matching unexpired preview required"})
			return
		}
		output, err := a.runControl(ctx, maxControlOutput, "collection-apply", request.Name, requestPath)
		if err != nil {
			a.controlError(w, err)
			return
		}
		_ = os.Remove(path)
		writeValidatedJSON(w, output)
	default:
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "mode must be preview or apply"})
	}
}

func validateV03Records(definition v03CollectionDefinition, records []map[string]any) ([]map[string]any, error) {
	if len(records) > definition.MaxRecords {
		return nil, errors.New("record count exceeds declared maximum")
	}
	fieldIndex := map[string]v03FieldDefinition{}
	for _, field := range definition.Fields {
		fieldIndex[field.Key] = field
	}
	identities := map[string]bool{}
	normalized := make([]map[string]any, 0, len(records))
	for index, record := range records {
		if len(record) > len(fieldIndex) {
			return nil, fmt.Errorf("record %d contains unknown fields", index+1)
		}
		item := map[string]any{}
		for key := range record {
			if _, ok := fieldIndex[key]; !ok {
				return nil, fmt.Errorf("record %d contains unknown field %s", index+1, key)
			}
		}
		for _, field := range definition.Fields {
			value, present := record[field.Key]
			if !present {
				if field.Required {
					return nil, fmt.Errorf("record %d missing required field %s", index+1, field.Key)
				}
				continue
			}
			normalizedValue, err := validateV03FieldValue(field, value)
			if err != nil {
				return nil, fmt.Errorf("record %d: %w", index+1, err)
			}
			item[field.Key] = normalizedValue
		}
		identity, _ := item[definition.IdentityKey].(string)
		if identity == "" {
			return nil, fmt.Errorf("record %d has empty identity", index+1)
		}
		if identities[identity] {
			return nil, fmt.Errorf("duplicate record identity: %s", identity)
		}
		identities[identity] = true
		normalized = append(normalized, item)
	}
	return normalized, nil
}

func validateV03FieldValue(field v03FieldDefinition, value any) (any, error) {
	switch field.Type {
	case "boolean":
		boolean, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("%s must be boolean", field.Key)
		}
		return boolean, nil
	case "integer":
		number, ok := value.(json.Number)
		if !ok {
			return nil, fmt.Errorf("%s must be integer", field.Key)
		}
		integer, err := number.Int64()
		if err != nil {
			return nil, fmt.Errorf("%s must be integer", field.Key)
		}
		if field.Min != nil && integer < *field.Min {
			return nil, fmt.Errorf("%s is below minimum", field.Key)
		}
		if field.Max != nil && integer > *field.Max {
			return nil, fmt.Errorf("%s is above maximum", field.Key)
		}
		return integer, nil
	case "string":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be string", field.Key)
		}
		if field.Required && text == "" {
			return nil, fmt.Errorf("%s is required", field.Key)
		}
		maxLength := field.MaxLength
		if maxLength == 0 {
			maxLength = 1024
		}
		if len(text) > maxLength {
			return nil, fmt.Errorf("%s exceeds maximum length", field.Key)
		}
		if field.Pattern != "" && !regexp.MustCompile(field.Pattern).MatchString(text) {
			return nil, fmt.Errorf("%s does not match required pattern", field.Key)
		}
		return text, nil
	case "enum":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be string", field.Key)
		}
		for _, option := range field.Options {
			if text == option.Value {
				return text, nil
			}
		}
		return nil, fmt.Errorf("%s has unsupported value", field.Key)
	default:
		return nil, fmt.Errorf("unsupported field type for %s", field.Key)
	}
}

func (a *application) v03ImportPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if !a.requireV03UploadMutation(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	document, err := a.loadV03Capabilities(ctx)
	if err != nil || !document.Features["transfer"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "transfer capability disabled"})
		return
	}
	name := r.URL.Query().Get("name")
	definition, ok := findV03Import(document, name)
	if !ok {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported import"})
		return
	}
	uploadPath, digest, size, err := a.stageV03Upload(w, r, definition.MaxBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	keepUpload := false
	defer func() {
		if !keepUpload {
			_ = os.Remove(uploadPath)
		}
	}()
	output, err := a.runControl(ctx, maxControlOutput, "import-preview", name, uploadPath)
	if err != nil {
		a.controlError(w, err)
		return
	}
	result, err := decodeControlJSON(output)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, responseEnvelope{OK: false, Error: "module backend returned invalid JSON"})
		return
	}
	token, err := a.writeV03Preview(v03PreviewRecord{Kind: "import", Name: name, Digest: digest, UploadPath: uploadPath, ExpiresAt: time.Now().Add(v03PreviewTTL).UTC().Format(time.RFC3339Nano)})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, responseEnvelope{OK: false, Error: "could not persist import preview"})
		return
	}
	keepUpload = true
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "preview_token": token, "sha256": digest, "bytes": size, "result": result})
}

func (a *application) v03ImportApplyHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireV03JSONMutation(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	document, err := a.loadV03Capabilities(ctx)
	if err != nil || !document.Features["transfer"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "transfer capability disabled"})
		return
	}
	var request v03ImportApplyRequest
	if err := decodeV03JSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	definition, ok := findV03Import(document, request.Name)
	if !ok {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported import"})
		return
	}
	if definition.RequiresConfirmation && request.Confirmation != definition.ConfirmationText {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "confirmation text does not match"})
		return
	}
	preview, previewPath, err := a.readV03Preview(request.PreviewToken)
	if err != nil || preview.Kind != "import" || preview.Name != request.Name || preview.UploadPath == "" {
		writeJSON(w, http.StatusConflict, responseEnvelope{OK: false, Error: "matching unexpired import preview required"})
		return
	}
	digest, size, err := digestRegularFileWithin(preview.UploadPath, filepath.Join(a.runtimeDir, "uploads"), definition.MaxBytes)
	if err != nil || digest != preview.Digest {
		writeJSON(w, http.StatusConflict, responseEnvelope{OK: false, Error: "staged import changed or expired"})
		return
	}
	requestPath, _, err := a.writeV03NormalizedRequest("import-apply", map[string]any{"name": request.Name, "preview_token": request.PreviewToken, "sha256": digest, "bytes": size, "confirmation": request.Confirmation})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, responseEnvelope{OK: false, Error: "could not stage import apply request"})
		return
	}
	defer os.Remove(requestPath)
	output, err := a.runControl(ctx, maxControlOutput, "import-apply", request.Name, preview.UploadPath, requestPath)
	if err != nil {
		a.controlError(w, err)
		return
	}
	_ = os.Remove(preview.UploadPath)
	_ = os.Remove(previewPath)
	writeValidatedJSON(w, output)
}

func (a *application) v03ExportHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireV03JSONMutation(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	document, err := a.loadV03Capabilities(ctx)
	if err != nil || !document.Features["transfer"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "transfer capability disabled"})
		return
	}
	var request v03ExportRequest
	if err := decodeV03JSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	definition, ok := findV03Export(document, request.Name)
	if !ok {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported export"})
		return
	}
	if definition.RequiresConfirmation && request.Confirmation != definition.ConfirmationText {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "confirmation text does not match"})
		return
	}
	output, err := a.runControl(ctx, definition.MaxBytes, "export", request.Name)
	if err != nil {
		a.controlError(w, err)
		return
	}
	filename := definition.Filename
	if filename == "" {
		extension := definition.Format
		filename = a.capabilities.Module.ID + "-" + definition.Name + "." + extension
	}
	if definition.Format == "zip" {
		w.Header().Set("Content-Type", "application/zip")
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("X-WebUI-Export-Policy", definition.SecretPolicy)
	_, _ = w.Write(output)
}

func (a *application) requireV03JSONMutation(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return false
	}
	if r.Header.Get("Origin") != a.origin || r.Header.Get(requestGuardHeader) != "1" {
		writeJSON(w, http.StatusForbidden, responseEnvelope{OK: false, Error: "mutation request rejected"})
		return false
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, responseEnvelope{OK: false, Error: "application/json required"})
		return false
	}
	return true
}

func (a *application) requireV03UploadMutation(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Origin") != a.origin || r.Header.Get(requestGuardHeader) != "1" {
		writeJSON(w, http.StatusForbidden, responseEnvelope{OK: false, Error: "mutation request rejected"})
		return false
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		writeJSON(w, http.StatusUnsupportedMediaType, responseEnvelope{OK: false, Error: "valid content type required"})
		return false
	}
	switch contentType {
	case "application/json", "application/zip", "application/octet-stream":
		return true
	default:
		writeJSON(w, http.StatusUnsupportedMediaType, responseEnvelope{OK: false, Error: "unsupported import content type"})
		return false
	}
}

func decodeV03JSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxV03JSONBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errors.New("invalid JSON request")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func (a *application) writeV03NormalizedRequest(prefix string, value any) (string, string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(data)
	requestPath, err := a.writeRequestFile(prefix, value)
	if err != nil {
		return "", "", err
	}
	return requestPath, hex.EncodeToString(digest[:]), nil
}

func decodeControlJSON(output []byte) (any, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func (a *application) v03PreviewDirectory() string {
	return filepath.Join(a.runtimeDir, "previews")
}

func (a *application) writeV03Preview(record v03PreviewRecord) (string, error) {
	if err := os.MkdirAll(a.v03PreviewDirectory(), 0o700); err != nil {
		return "", err
	}
	token, err := randomHex(16)
	if err != nil {
		return "", err
	}
	path := filepath.Join(a.v03PreviewDirectory(), token+".json")
	if err := writeRuntimeJSON(path, record); err != nil {
		return "", err
	}
	return token, nil
}

func (a *application) readV03Preview(token string) (v03PreviewRecord, string, error) {
	var record v03PreviewRecord
	if !jobIDPattern.MatchString(token) {
		return record, "", errors.New("invalid preview token")
	}
	path := filepath.Join(a.v03PreviewDirectory(), token+".json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return record, "", errors.New("preview not found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return record, "", err
	}
	if err := json.Unmarshal(data, &record); err != nil {
		return record, "", err
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, record.ExpiresAt)
	if err != nil || !time.Now().Before(expiresAt) {
		_ = os.Remove(path)
		if record.UploadPath != "" {
			_ = os.Remove(record.UploadPath)
		}
		return record, "", errors.New("preview expired")
	}
	return record, path, nil
}

func (a *application) stageV03Upload(w http.ResponseWriter, r *http.Request, maximum int64) (string, string, int64, error) {
	if maximum < 1 || maximum > maxV03UploadBytes {
		return "", "", 0, errors.New("invalid upload limit")
	}
	dir := filepath.Join(a.runtimeDir, "uploads")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", 0, err
	}
	file, err := os.CreateTemp(dir, "import-*")
	if err != nil {
		return "", "", 0, err
	}
	path := file.Name()
	_ = os.Chmod(path, 0o600)
	body := http.MaxBytesReader(w, r.Body, maximum)
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return "", "", 0, errors.New("import exceeds limit or could not be staged")
	}
	if closeErr != nil || written < 1 || written > maximum {
		_ = os.Remove(path)
		return "", "", 0, errors.New("invalid staged import")
	}
	return path, hex.EncodeToString(hash.Sum(nil)), written, nil
}

func digestRegularFileWithin(path, root string, maximum int64) (string, int64, error) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", 0, err
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return "", 0, err
	}
	if cleanPath == cleanRoot || !strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator)) {
		return "", 0, errors.New("file outside private upload directory")
	}
	info, err := os.Lstat(cleanPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", 0, errors.New("staged import is not a regular file")
	}
	if info.Size() < 1 || info.Size() > maximum {
		return "", 0, errors.New("staged import size out of range")
	}
	file, err := os.Open(cleanPath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(file, maximum+1)); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), info.Size(), nil
}

func parseV03MaxBytes(raw string, fallback int64) int64 {
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

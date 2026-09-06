package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	v04CapabilitySchema = "root-module-webui.extensions.v2"
	v04RequestMaxBytes  = 64 << 10
)

type v04ParameterDefinition struct {
	Key         string             `json:"key"`
	Label       string             `json:"label"`
	Description string             `json:"description,omitempty"`
	Type        string             `json:"type"`
	Required    bool               `json:"required,omitempty"`
	Min         *int64             `json:"min,omitempty"`
	Max         *int64             `json:"max,omitempty"`
	MaxLength   int                `json:"max_length,omitempty"`
	Pattern     string             `json:"pattern,omitempty"`
	Options     []optionDefinition `json:"options,omitempty"`
	Reference   string             `json:"reference,omitempty"`
}

type v04JobDefinition struct {
	Name        string                   `json:"name"`
	Label       string                   `json:"label"`
	Description string                   `json:"description,omitempty"`
	Risk        string                   `json:"risk"`
	Parameters  []v04ParameterDefinition `json:"parameters,omitempty"`
	DedupeKeys  []string                 `json:"dedupe_keys,omitempty"`
	Phases      []string                 `json:"phases,omitempty"`
}

type v04ReferenceDefinition struct {
	Name              string `json:"name"`
	Label             string `json:"label"`
	SourceCollection  string `json:"source_collection"`
	SourceIdentityKey string `json:"source_identity_key"`
}

type v04InventoryOperationDefinition struct {
	Name          string `json:"name"`
	Label         string `json:"label"`
	Description   string `json:"description,omitempty"`
	Inventory     string `json:"inventory"`
	IdentityField string `json:"identity_field"`
	Job           string `json:"job"`
	JobParameter  string `json:"job_parameter"`
}

type v04CapabilityDocument struct {
	Schema              string                            `json:"schema"`
	Module              moduleDefinition                  `json:"module"`
	Features            map[string]bool                   `json:"features"`
	References          []v04ReferenceDefinition          `json:"references,omitempty"`
	Jobs                []v04JobDefinition                `json:"jobs,omitempty"`
	InventoryOperations []v04InventoryOperationDefinition `json:"inventory_operations,omitempty"`
}

type v04JobRequest struct {
	Name       string         `json:"name"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

type v04InventoryOperationRequest struct {
	Name   string `json:"name"`
	ItemID string `json:"item_id"`
}

type v04JobMeta struct {
	Digest    string
	DedupeKey string
	CreatedAt time.Time
}

var (
	v04MetaMu sync.Mutex
	v04Meta   = map[string]v04JobMeta{}
)

func registerV04Handlers(mux *http.ServeMux, app *application) {
	mux.HandleFunc("/v04.js", app.requireSession(app.v04Asset))
	mux.HandleFunc("/api/v1/v04/capabilities", app.requireSession(app.v04CapabilitiesHandler))
	mux.HandleFunc("/api/v1/v04/jobs", app.requireSession(app.v04JobsHandler))
	mux.HandleFunc("/api/v1/v04/inventory-operation", app.requireSession(app.v04InventoryOperationHandler))
	mux.HandleFunc("/api/v1/v04/reference", app.requireSession(app.v04ReferenceHandler))
}

func (a *application) v04Asset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, http.MethodGet, http.MethodHead)
		return
	}
	path := filepath.Join(a.webroot, "v04.js")
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

func (a *application) loadV04Capabilities(ctx context.Context) (v04CapabilityDocument, error) {
	var document v04CapabilityDocument
	output, err := a.runControl(ctx, maxControlOutput, "capabilities-v04")
	if err != nil {
		return document, err
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return document, fmt.Errorf("invalid v0.4 capability JSON: %w", err)
	}
	if document.Schema != v04CapabilitySchema {
		return document, errors.New("unsupported v0.4 capability schema")
	}
	if document.Module.ID != a.capabilities.Module.ID {
		return document, errors.New("v0.4 module id does not match base capabilities")
	}
	return document, validateV04Capabilities(document, a)
}

func validateV04Capabilities(document v04CapabilityDocument, a *application) error {
	referenceNames := map[string]v04ReferenceDefinition{}
	for _, reference := range document.References {
		if !safeNamePattern.MatchString(reference.Name) || !safeNamePattern.MatchString(reference.SourceCollection) || !safeNamePattern.MatchString(reference.SourceIdentityKey) {
			return fmt.Errorf("invalid reference declaration: %s", reference.Name)
		}
		if _, exists := referenceNames[reference.Name]; exists {
			return fmt.Errorf("duplicate reference: %s", reference.Name)
		}
		referenceNames[reference.Name] = reference
	}

	jobs := map[string]v04JobDefinition{}
	for _, job := range document.Jobs {
		if !safeNamePattern.MatchString(job.Name) {
			return fmt.Errorf("invalid v0.4 job name: %s", job.Name)
		}
		if err := validateRisk(job.Risk); err != nil {
			return fmt.Errorf("v0.4 job %s: %w", job.Name, err)
		}
		if _, exists := jobs[job.Name]; exists {
			return fmt.Errorf("duplicate v0.4 job: %s", job.Name)
		}
		if len(job.Parameters) > 16 || len(job.DedupeKeys) > 8 || len(job.Phases) > 16 {
			return fmt.Errorf("v0.4 job %s exceeds declaration bounds", job.Name)
		}
		params := map[string]bool{}
		for _, parameter := range job.Parameters {
			if err := validateV04Parameter(parameter, referenceNames); err != nil {
				return fmt.Errorf("v0.4 job %s: %w", job.Name, err)
			}
			if params[parameter.Key] {
				return fmt.Errorf("v0.4 job %s has duplicate parameter %s", job.Name, parameter.Key)
			}
			params[parameter.Key] = true
		}
		for _, key := range job.DedupeKeys {
			if !params[key] {
				return fmt.Errorf("v0.4 job %s dedupe key %s is not a declared parameter", job.Name, key)
			}
		}
		for _, phase := range job.Phases {
			if !safeNamePattern.MatchString(phase) {
				return fmt.Errorf("v0.4 job %s has invalid phase %s", job.Name, phase)
			}
		}
		jobs[job.Name] = job
	}

	operations := map[string]bool{}
	for _, operation := range document.InventoryOperations {
		if !safeNamePattern.MatchString(operation.Name) || !safeNamePattern.MatchString(operation.Inventory) || !safeNamePattern.MatchString(operation.IdentityField) || !safeNamePattern.MatchString(operation.Job) || !safeNamePattern.MatchString(operation.JobParameter) {
			return fmt.Errorf("invalid inventory operation: %s", operation.Name)
		}
		if operations[operation.Name] {
			return fmt.Errorf("duplicate inventory operation: %s", operation.Name)
		}
		if _, ok := a.inventoryIndex[operation.Inventory]; !ok {
			return fmt.Errorf("inventory operation %s references undeclared inventory %s", operation.Name, operation.Inventory)
		}
		job, ok := jobs[operation.Job]
		if !ok {
			return fmt.Errorf("inventory operation %s references undeclared v0.4 job %s", operation.Name, operation.Job)
		}
		found := false
		for _, parameter := range job.Parameters {
			if parameter.Key == operation.JobParameter {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("inventory operation %s references undeclared job parameter %s", operation.Name, operation.JobParameter)
		}
		operations[operation.Name] = true
	}
	return nil
}

func validateV04Parameter(parameter v04ParameterDefinition, references map[string]v04ReferenceDefinition) error {
	if !safeNamePattern.MatchString(parameter.Key) {
		return fmt.Errorf("invalid parameter key: %s", parameter.Key)
	}
	switch parameter.Type {
	case "boolean", "integer", "string", "enum", "reference":
	default:
		return fmt.Errorf("unsupported parameter type: %s", parameter.Type)
	}
	if parameter.MaxLength < 0 || parameter.MaxLength > 4096 {
		return fmt.Errorf("invalid max_length for %s", parameter.Key)
	}
	if parameter.Pattern != "" {
		if _, err := regexp.Compile(parameter.Pattern); err != nil {
			return fmt.Errorf("invalid pattern for %s: %w", parameter.Key, err)
		}
	}
	if parameter.Type == "enum" && len(parameter.Options) == 0 {
		return fmt.Errorf("enum parameter %s requires options", parameter.Key)
	}
	if parameter.Type == "reference" {
		if _, ok := references[parameter.Reference]; !ok {
			return fmt.Errorf("reference parameter %s uses undeclared reference %s", parameter.Key, parameter.Reference)
		}
	}
	return nil
}

func (a *application) v04CapabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), statusControlTimeout)
	defer cancel()
	document, err := a.loadV04Capabilities(ctx)
	if err != nil {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "v0.4 extension capability disabled"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "extensions": document})
}

func (a *application) v04ReferenceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), statusControlTimeout)
	defer cancel()
	document, err := a.loadV04Capabilities(ctx)
	if err != nil || !document.Features["references"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "reference capability disabled"})
		return
	}
	name := r.URL.Query().Get("name")
	reference, ok := findV04Reference(document, name)
	if !ok {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported reference"})
		return
	}
	values, err := a.loadV04ReferenceValues(ctx, reference)
	if err != nil {
		a.controlError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": name, "items": values})
}

func (a *application) v04JobsHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireMutation(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), statusControlTimeout)
	defer cancel()
	document, err := a.loadV04Capabilities(ctx)
	if err != nil || !document.Features["typed_jobs"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "typed job capability disabled"})
		return
	}
	var request v04JobRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	definition, ok := findV04Job(document, request.Name)
	if !ok {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported typed job"})
		return
	}
	normalized, err := a.validateV04Parameters(ctx, document, definition, request.Parameters)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	job, reused, digest, err := a.startV04Job(definition, normalized)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errV04Capacity) {
			status = http.StatusConflict
		}
		writeJSON(w, status, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "reused": reused, "request_digest": digest, "job": job.snapshot()})
}

func (a *application) v04InventoryOperationHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireMutation(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	document, err := a.loadV04Capabilities(ctx)
	if err != nil || !document.Features["inventory_operations"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "inventory operation capability disabled"})
		return
	}
	var request v04InventoryOperationRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	operation, ok := findV04InventoryOperation(document, request.Name)
	if !ok {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported inventory operation"})
		return
	}
	if request.ItemID == "" || len(request.ItemID) > 256 {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "invalid inventory item id"})
		return
	}
	item, err := a.resolveInventoryItem(ctx, operation.Inventory, operation.IdentityField, request.ItemID)
	if err != nil {
		writeJSON(w, http.StatusConflict, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	_ = item // Resolution is the authorization boundary; the adapter receives only the declared identity.
	job, ok := findV04Job(document, operation.Job)
	if !ok {
		writeJSON(w, http.StatusBadGateway, responseEnvelope{OK: false, Error: "inventory operation job declaration unavailable"})
		return
	}
	normalized, err := a.validateV04Parameters(ctx, document, job, map[string]any{operation.JobParameter: request.ItemID})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	started, reused, digest, err := a.startV04Job(job, normalized)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errV04Capacity) {
			status = http.StatusConflict
		}
		writeJSON(w, status, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "reused": reused, "request_digest": digest, "job": started.snapshot()})
}

func findV04Reference(document v04CapabilityDocument, name string) (v04ReferenceDefinition, bool) {
	for _, item := range document.References {
		if item.Name == name {
			return item, true
		}
	}
	return v04ReferenceDefinition{}, false
}

func findV04Job(document v04CapabilityDocument, name string) (v04JobDefinition, bool) {
	for _, item := range document.Jobs {
		if item.Name == name {
			return item, true
		}
	}
	return v04JobDefinition{}, false
}

func findV04InventoryOperation(document v04CapabilityDocument, name string) (v04InventoryOperationDefinition, bool) {
	for _, item := range document.InventoryOperations {
		if item.Name == name {
			return item, true
		}
	}
	return v04InventoryOperationDefinition{}, false
}

func (a *application) validateV04Parameters(ctx context.Context, document v04CapabilityDocument, definition v04JobDefinition, values map[string]any) (map[string]any, error) {
	if values == nil {
		values = map[string]any{}
	}
	if len(values) > len(definition.Parameters) {
		return nil, errors.New("typed job contains unknown parameters")
	}
	index := map[string]v04ParameterDefinition{}
	for _, parameter := range definition.Parameters {
		index[parameter.Key] = parameter
	}
	for key := range values {
		if _, ok := index[key]; !ok {
			return nil, fmt.Errorf("unknown typed job parameter: %s", key)
		}
	}
	normalized := map[string]any{}
	for _, parameter := range definition.Parameters {
		value, present := values[parameter.Key]
		if !present {
			if parameter.Required {
				return nil, fmt.Errorf("missing required typed job parameter: %s", parameter.Key)
			}
			continue
		}
		normalizedValue, err := validateV04ParameterValue(parameter, value)
		if err != nil {
			return nil, err
		}
		if parameter.Type == "reference" {
			reference, ok := findV04Reference(document, parameter.Reference)
			if !ok {
				return nil, fmt.Errorf("reference declaration unavailable for %s", parameter.Key)
			}
			items, err := a.loadV04ReferenceValues(ctx, reference)
			if err != nil {
				return nil, fmt.Errorf("reference lookup failed for %s: %w", parameter.Key, err)
			}
			text := normalizedValue.(string)
			found := false
			for _, item := range items {
				if item == text {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("%s references an unknown item", parameter.Key)
			}
		}
		normalized[parameter.Key] = normalizedValue
	}
	return normalized, nil
}

func validateV04ParameterValue(parameter v04ParameterDefinition, value any) (any, error) {
	switch parameter.Type {
	case "boolean":
		boolean, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("%s must be boolean", parameter.Key)
		}
		return boolean, nil
	case "integer":
		number, ok := value.(json.Number)
		if !ok {
			return nil, fmt.Errorf("%s must be integer", parameter.Key)
		}
		integer, err := number.Int64()
		if err != nil {
			return nil, fmt.Errorf("%s must be integer", parameter.Key)
		}
		if parameter.Min != nil && integer < *parameter.Min {
			return nil, fmt.Errorf("%s is below minimum", parameter.Key)
		}
		if parameter.Max != nil && integer > *parameter.Max {
			return nil, fmt.Errorf("%s is above maximum", parameter.Key)
		}
		return integer, nil
	case "string", "reference":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be string", parameter.Key)
		}
		if parameter.Required && text == "" {
			return nil, fmt.Errorf("%s is required", parameter.Key)
		}
		maximum := parameter.MaxLength
		if maximum == 0 {
			maximum = 1024
		}
		if len(text) > maximum {
			return nil, fmt.Errorf("%s exceeds maximum length", parameter.Key)
		}
		if parameter.Pattern != "" && !regexp.MustCompile(parameter.Pattern).MatchString(text) {
			return nil, fmt.Errorf("%s does not match required pattern", parameter.Key)
		}
		return text, nil
	case "enum":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be string", parameter.Key)
		}
		for _, option := range parameter.Options {
			if option.Value == text {
				return text, nil
			}
		}
		return nil, fmt.Errorf("%s has unsupported value", parameter.Key)
	default:
		return nil, fmt.Errorf("unsupported parameter type for %s", parameter.Key)
	}
}

func (a *application) loadV04ReferenceValues(ctx context.Context, reference v04ReferenceDefinition) ([]string, error) {
	v03, err := a.loadV03Capabilities(ctx)
	if err != nil {
		return nil, errors.New("typed collection extension unavailable")
	}
	collection, ok := findV03Collection(v03, reference.SourceCollection)
	if !ok || collection.IdentityKey != reference.SourceIdentityKey {
		return nil, errors.New("reference source collection mismatch")
	}
	output, err := a.runControl(ctx, maxInventoryOutput, "collection-get", reference.SourceCollection)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Records []map[string]any `json:"records"`
		Items   []map[string]any `json:"items"`
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		return nil, errors.New("reference source returned invalid JSON")
	}
	rows := envelope.Records
	if rows == nil {
		rows = envelope.Items
	}
	values := make([]string, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		value, ok := row[reference.SourceIdentityKey].(string)
		if !ok || value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	sort.Strings(values)
	return values, nil
}

var errV04Capacity = errors.New("job capacity reached")

func (a *application) startV04Job(definition v04JobDefinition, parameters map[string]any) (*jobState, bool, string, error) {
	payload := map[string]any{"name": definition.Name, "parameters": parameters}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return nil, false, "", err
	}
	digestRaw := sha256.Sum256(canonical)
	digest := hex.EncodeToString(digestRaw[:])
	dedupe := v04DedupeKey(definition, parameters)
	if dedupe != "" {
		if existing := a.findActiveV04Job(dedupe); existing != nil {
			return existing, true, digest, nil
		}
	}
	if a.activeJobCount() >= int64(a.maxJobs) {
		return nil, false, digest, errV04Capacity
	}
	requestPath, err := a.writeRequestFile("job-v04", payload)
	if err != nil {
		return nil, false, digest, err
	}
	id, err := randomHex(16)
	if err != nil {
		_ = os.Remove(requestPath)
		return nil, false, digest, err
	}
	job := newJobState(id, definition.Name)
	a.jobsMu.Lock()
	a.pruneJobsLocked()
	a.jobs[id] = job
	a.jobsMu.Unlock()
	v04MetaMu.Lock()
	v04Meta[id] = v04JobMeta{Digest: digest, DedupeKey: dedupe, CreatedAt: time.Now()}
	pruneV04MetaLocked(a)
	v04MetaMu.Unlock()
	a.activeJobs.Add(1)
	go a.runV04Job(job, requestPath)
	return job, false, digest, nil
}

func v04DedupeKey(definition v04JobDefinition, parameters map[string]any) string {
	if len(definition.DedupeKeys) == 0 {
		return ""
	}
	selected := map[string]any{"name": definition.Name}
	for _, key := range definition.DedupeKeys {
		selected[key] = parameters[key]
	}
	data, err := json.Marshal(selected)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (a *application) findActiveV04Job(dedupe string) *jobState {
	v04MetaMu.Lock()
	defer v04MetaMu.Unlock()
	for id, meta := range v04Meta {
		if meta.DedupeKey != dedupe {
			continue
		}
		a.jobsMu.RLock()
		job := a.jobs[id]
		a.jobsMu.RUnlock()
		if job == nil {
			continue
		}
		status := job.snapshot().Status
		if status == "queued" || status == "running" {
			return job
		}
	}
	return nil
}

func pruneV04MetaLocked(a *application) {
	if len(v04Meta) <= 32 {
		return
	}
	type item struct {
		id      string
		created time.Time
	}
	items := make([]item, 0, len(v04Meta))
	for id, meta := range v04Meta {
		items = append(items, item{id: id, created: meta.CreatedAt})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].created.Before(items[j].created) })
	for _, candidate := range items {
		if len(v04Meta) <= 32 {
			break
		}
		a.jobsMu.RLock()
		job := a.jobs[candidate.id]
		a.jobsMu.RUnlock()
		if job != nil {
			status := job.snapshot().Status
			if status == "queued" || status == "running" {
				continue
			}
		}
		delete(v04Meta, candidate.id)
	}
}

func (a *application) runV04Job(job *jobState, requestPath string) {
	defer os.Remove(requestPath)
	job.mu.Lock()
	job.record.Status = "running"
	job.record.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	job.started = time.Now()
	job.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), a.jobTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, a.control, "job-run-file", job.record.Name, requestPath)
	cmd.Env = a.controlEnvironment()
	cmd.Stdout = &job.stdout
	cmd.Stderr = &job.stderr
	err := cmd.Run()

	finished := time.Now()
	exitCode := 0
	status := "success"
	errorMessage := ""
	if err != nil {
		status = "failed"
		exitCode = -1
		errorMessage = err.Error()
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			errorMessage = "job timeout exceeded"
		}
	}
	job.mu.Lock()
	job.finished = finished
	job.record.Status = status
	job.record.FinishedAt = finished.UTC().Format(time.RFC3339Nano)
	job.record.ExitCode = &exitCode
	job.record.Error = errorMessage
	job.mu.Unlock()
	a.activeJobs.Add(-1)
	a.touch()
}

func (a *application) resolveInventoryItem(ctx context.Context, inventory, identityField, itemID string) (map[string]any, error) {
	output, err := a.runControl(ctx, maxInventoryOutput, "inventory", inventory)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Items []map[string]any `json:"items"`
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		return nil, errors.New("inventory returned invalid JSON")
	}
	for _, item := range envelope.Items {
		if value, ok := item[identityField].(string); ok && value == itemID {
			return item, nil
		}
	}
	return nil, errors.New("inventory item is stale or unavailable")
}

func digestV04Request(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func decodeV04Strict(data []byte, destination any) error {
	if len(data) > v04RequestMaxBytes {
		return errors.New("v0.4 document exceeds size limit")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, os.ErrClosed) && err == nil {
		return errors.New("v0.4 document must contain one JSON value")
	}
	return nil
}

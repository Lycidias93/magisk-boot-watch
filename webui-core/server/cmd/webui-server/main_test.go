package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func int64Ptr(value int64) *int64 { return &value }

func TestValidateListenAddress(t *testing.T) {
	if err := validateListenAddress("127.0.0.1:0"); err != nil {
		t.Fatalf("loopback rejected: %v", err)
	}
	for _, address := range []string{"0.0.0.0:0", "[::1]:0", "localhost:0", "example.com:8080"} {
		if err := validateListenAddress(address); err == nil {
			t.Fatalf("unsafe address accepted: %s", address)
		}
	}
}

func TestReadBootstrapToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	token := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readBootstrapToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != token {
		t.Fatalf("unexpected token: %q", got)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readBootstrapToken(path); err == nil {
		t.Fatal("world-readable token accepted")
	}
}

func TestBootstrapOneTimeCookie(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenPath, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := &application{
		hostPort:            "127.0.0.1:12345",
		origin:              "http://127.0.0.1:12345",
		bootstrapToken:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		session:             "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		tokenFile:           tokenPath,
		sessionCookieMaxAge: 900,
	}
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:12345/bootstrap?token="+app.bootstrapToken, nil)
	recorder := httptest.NewRecorder()
	app.bootstrap(recorder, request)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("bootstrap code=%d body=%s", recorder.Code, recorder.Body.String())
	}
	response := recorder.Result()
	cookies := response.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count=%d", len(cookies))
	}
	if !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unsafe cookie attributes: %#v", cookies[0])
	}
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Fatal("bootstrap token file was not removed")
	}

	second := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:12345/bootstrap?token=anything", nil)
	secondRecorder := httptest.NewRecorder()
	app.bootstrap(secondRecorder, second)
	if secondRecorder.Code != http.StatusForbidden {
		t.Fatalf("second bootstrap code=%d", secondRecorder.Code)
	}
}

func TestCoreHTMLAssetsAreServed(t *testing.T) {
	dir := t.TempDir()
	assets := []string{
		"index.html",
		"mobile-input-viewport.js",
		"app.js",
		"app.css",
		"race-guard.js",
		"race-guard.css",
		"observability.js",
		"observability.css",
		"v03.js",
		"v04.js",
	}
	for _, name := range assets {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("asset:"+name), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	app := &application{
		webroot:        dir,
		session:        "0123456789abcdef",
		sessionExpires: time.Now().Add(time.Minute),
	}
	mux := http.NewServeMux()
	registerV03Handlers(mux, app)
	registerV04Handlers(mux, app)
	mux.HandleFunc("/", app.pageOrAsset)

	for _, path := range []string{
		"/",
		"/mobile-input-viewport.js",
		"/app.js",
		"/app.css",
		"/race-guard.js",
		"/race-guard.css",
		"/observability.js",
		"/observability.css",
		"/v03.js",
		"/v04.js",
	} {
		request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1"+path, nil)
		request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: app.session})
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("asset %s code=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
		if recorder.Body.Len() == 0 {
			t.Fatalf("asset %s returned empty body", path)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/not-allowlisted.js", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: app.session})
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected arbitrary asset status=%d", recorder.Code)
	}
}

func TestValidateConfig(t *testing.T) {
	app := &application{configIndex: map[string]configField{
		"enabled": {Key: "enabled", Type: "boolean"},
		"mode": {
			Key:  "mode",
			Type: "enum",
			Options: []optionDefinition{
				{Value: "balanced", Label: "Balanced"},
				{Value: "battery", Label: "Battery"},
			},
		},
		"interval_seconds": {Key: "interval_seconds", Type: "integer", Min: int64Ptr(15), Max: int64Ptr(3600)},
	}}
	request := map[string]any{
		"enabled":          true,
		"mode":             "balanced",
		"interval_seconds": json.Number("60"),
	}
	normalized, err := app.validateConfig(request)
	if err != nil {
		t.Fatal(err)
	}
	if normalized["interval_seconds"] != int64(60) {
		t.Fatalf("unexpected normalized config: %#v", normalized)
	}
	request["mode"] = "invalid"
	if _, err := app.validateConfig(request); err == nil {
		t.Fatal("invalid enum accepted")
	}
}

func TestValidateActionJobBindings(t *testing.T) {
	valid := &application{
		capabilities: capabilityDocument{Features: map[string]bool{"jobs": true}},
		actionIndex: map[string]actionDefinition{
			"sort-now": {Name: "sort-now", Risk: "caution", SupportsDryRun: true, ApplyJob: "sort-now"},
		},
		jobIndex: map[string]jobDefinition{
			"sort-now": {Name: "sort-now", Risk: "caution"},
		},
	}
	if err := valid.validateActionJobBindings(); err != nil {
		t.Fatalf("valid action/job binding rejected: %v", err)
	}

	cases := []struct {
		name string
		app  *application
	}{
		{
			name: "jobs feature disabled",
			app: &application{
				capabilities: capabilityDocument{Features: map[string]bool{"jobs": false}},
				actionIndex:  map[string]actionDefinition{"a": {Name: "a", Risk: "safe", SupportsDryRun: true, ApplyJob: "j"}},
				jobIndex:     map[string]jobDefinition{"j": {Name: "j", Risk: "safe"}},
			},
		},
		{
			name: "dry run missing",
			app: &application{
				capabilities: capabilityDocument{Features: map[string]bool{"jobs": true}},
				actionIndex:  map[string]actionDefinition{"a": {Name: "a", Risk: "safe", ApplyJob: "j"}},
				jobIndex:     map[string]jobDefinition{"j": {Name: "j", Risk: "safe"}},
			},
		},
		{
			name: "confirmation unsupported",
			app: &application{
				capabilities: capabilityDocument{Features: map[string]bool{"jobs": true}},
				actionIndex:  map[string]actionDefinition{"a": {Name: "a", Risk: "safe", SupportsDryRun: true, ApplyJob: "j", RequiresConfirmation: true}},
				jobIndex:     map[string]jobDefinition{"j": {Name: "j", Risk: "safe"}},
			},
		},
		{
			name: "job missing",
			app: &application{
				capabilities: capabilityDocument{Features: map[string]bool{"jobs": true}},
				actionIndex:  map[string]actionDefinition{"a": {Name: "a", Risk: "safe", SupportsDryRun: true, ApplyJob: "j"}},
				jobIndex:     map[string]jobDefinition{},
			},
		},
		{
			name: "risk mismatch",
			app: &application{
				capabilities: capabilityDocument{Features: map[string]bool{"jobs": true}},
				actionIndex:  map[string]actionDefinition{"a": {Name: "a", Risk: "safe", SupportsDryRun: true, ApplyJob: "j"}},
				jobIndex:     map[string]jobDefinition{"j": {Name: "j", Risk: "caution"}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.app.validateActionJobBindings(); err == nil {
				t.Fatal("invalid action/job binding accepted")
			}
		})
	}
}

func TestLimitedBuffer(t *testing.T) {
	buffer := &limitedBuffer{limit: 4}
	if _, err := buffer.Write([]byte("abcdef")); err != nil {
		t.Fatal(err)
	}
	if !buffer.Exceeded() {
		t.Fatal("limit exceedance not recorded")
	}
	if got := string(buffer.Bytes()); got != "abcd" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestBackgroundJobLifecycle(t *testing.T) {
	dir := t.TempDir()
	control := filepath.Join(dir, "control")
	script := "#!/bin/sh\nif [ \"$1\" = job-run ]; then printf 'job:%s\\n' \"$2\"; exit 0; fi\nexit 2\n"
	if err := os.WriteFile(control, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	app := &application{
		control:    control,
		moduleDir:  dir,
		stateDir:   dir,
		runtimeDir: dir,
		jobTimeout: time.Minute,
		maxJobs:    1,
		jobs:       make(map[string]*jobState),
	}
	job, err := app.startJob("diagnostics")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		record := job.snapshot()
		if record.Status == "success" {
			if record.ExitCode == nil || *record.ExitCode != 0 {
				t.Fatalf("unexpected exit code: %#v", record.ExitCode)
			}
			if string(job.stdout.Bytes()) != "job:diagnostics\n" {
				t.Fatalf("unexpected output: %q", job.stdout.Bytes())
			}
			return
		}
		if record.Status == "failed" {
			t.Fatalf("job failed: %s", record.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("job did not finish")
}

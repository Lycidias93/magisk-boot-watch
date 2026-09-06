package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var version = "dev"

const (
	maxBodyBytes         = 32 << 10
	maxControlOutput     = 256 << 10
	maxJobOutput         = 256 << 10
	maxInventoryOutput   = 512 << 10
	sessionCookieName    = "root_module_webui_session"
	requestGuardHeader   = "X-WebUI-Request"
	statusControlTimeout = 15 * time.Second
)

var (
	safeNamePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)
	jobIDPattern    = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

type optionDefinition struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type configField struct {
	Key         string             `json:"key"`
	Label       string             `json:"label"`
	Description string             `json:"description,omitempty"`
	Type        string             `json:"type"`
	Required    bool               `json:"required,omitempty"`
	Secret      bool               `json:"secret,omitempty"`
	Min         *int64             `json:"min,omitempty"`
	Max         *int64             `json:"max,omitempty"`
	MaxLength   int                `json:"max_length,omitempty"`
	Pattern     string             `json:"pattern,omitempty"`
	Options     []optionDefinition `json:"options,omitempty"`
}

type actionDefinition struct {
	Name                 string `json:"name"`
	Label                string `json:"label"`
	Description          string `json:"description,omitempty"`
	Risk                 string `json:"risk"`
	SupportsDryRun       bool   `json:"supports_dry_run,omitempty"`
	ApplyJob             string `json:"apply_job,omitempty"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"`
	ConfirmationText     string `json:"confirmation_text,omitempty"`
}

type jobDefinition struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Risk        string `json:"risk"`
}

type inventoryDefinition struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type moduleDefinition struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type capabilityDocument struct {
	Schema      string                `json:"schema"`
	Module      moduleDefinition      `json:"module"`
	Features    map[string]bool       `json:"features"`
	Config      []configField         `json:"config_fields,omitempty"`
	Actions     []actionDefinition    `json:"actions,omitempty"`
	Jobs        []jobDefinition       `json:"jobs,omitempty"`
	Inventories []inventoryDefinition `json:"inventories,omitempty"`
}

type actionRequest struct {
	Name         string `json:"name"`
	DryRun       bool   `json:"dry_run,omitempty"`
	Confirmation string `json:"confirmation,omitempty"`
}

type jobRequest struct {
	Name string `json:"name"`
}

type stateFile struct {
	Port      int    `json:"port"`
	PID       int    `json:"pid"`
	Version   string `json:"version"`
	StartedAt string `json:"started_at"`
}

type responseEnvelope struct {
	OK           bool                `json:"ok"`
	Error        string              `json:"error,omitempty"`
	Service      string              `json:"service,omitempty"`
	Capabilities *capabilityDocument `json:"capabilities,omitempty"`
	Data         any                 `json:"data,omitempty"`
}

type limitedBuffer struct {
	mu       sync.RWMutex
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - b.buffer.Len()
	if remaining > 0 {
		write := p
		if len(write) > remaining {
			write = write[:remaining]
		}
		_, _ = b.buffer.Write(write)
	}
	if len(p) > remaining {
		b.exceeded = true
	}
	return len(p), nil
}

func (b *limitedBuffer) Bytes() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]byte(nil), b.buffer.Bytes()...)
}

func (b *limitedBuffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buffer.Len()
}

func (b *limitedBuffer) Exceeded() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.exceeded
}

type jobRecord struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
	StartedAt    string  `json:"started_at,omitempty"`
	FinishedAt   string  `json:"finished_at,omitempty"`
	ExitCode     *int    `json:"exit_code,omitempty"`
	Error        string  `json:"error,omitempty"`
	StdoutBytes  int     `json:"stdout_bytes"`
	StderrBytes  int     `json:"stderr_bytes"`
	Truncated    bool    `json:"truncated"`
	DurationSecs float64 `json:"duration_seconds,omitempty"`
}

type jobState struct {
	mu       sync.RWMutex
	record   jobRecord
	stdout   limitedBuffer
	stderr   limitedBuffer
	started  time.Time
	finished time.Time
}

func newJobState(id, name string) *jobState {
	return &jobState{
		record: jobRecord{
			ID:        id,
			Name:      name,
			Status:    "queued",
			CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		},
		stdout: limitedBuffer{limit: maxJobOutput},
		stderr: limitedBuffer{limit: maxJobOutput},
	}
}

func (j *jobState) snapshot() jobRecord {
	j.mu.RLock()
	record := j.record
	started := j.started
	finished := j.finished
	j.mu.RUnlock()
	record.StdoutBytes = j.stdout.Len()
	record.StderrBytes = j.stderr.Len()
	record.Truncated = j.stdout.Exceeded() || j.stderr.Exceeded()
	if !started.IsZero() {
		end := time.Now()
		if !finished.IsZero() {
			end = finished
		}
		record.DurationSecs = end.Sub(started).Seconds()
	}
	return record
}

type application struct {
	webroot             string
	control             string
	moduleDir           string
	stateDir            string
	runtimeDir          string
	tokenFile           string
	hostPort            string
	origin              string
	session             string
	bootstrapToken      string
	bootstrapMu         sync.Mutex
	bootstrapUsed       bool
	started             time.Time
	lastRequest         atomic.Int64
	logger              *log.Logger
	capabilities        capabilityDocument
	configIndex         map[string]configField
	actionIndex         map[string]actionDefinition
	jobIndex            map[string]jobDefinition
	inventoryIndex      map[string]inventoryDefinition
	jobTimeout          time.Duration
	maxJobs             int
	jobsMu              sync.RWMutex
	jobs                map[string]*jobState
	activeJobs          atomic.Int64
	sessionCookieMaxAge int
	sessionExpires      time.Time
}

func main() {
	var listenAddress string
	var webroot string
	var control string
	var moduleDir string
	var stateDir string
	var runtimeDir string
	var tokenFile string
	var statePath string
	var pidPath string
	var idleTimeout time.Duration
	var sessionTTL time.Duration
	var jobTimeout time.Duration
	var maxJobs int
	var selfTest bool

	flag.StringVar(&listenAddress, "listen", "127.0.0.1:0", "loopback listen address")
	flag.StringVar(&webroot, "webroot", "", "static WebUI directory")
	flag.StringVar(&control, "control", "", "module-control executable")
	flag.StringVar(&moduleDir, "module-dir", "", "installed module directory")
	flag.StringVar(&stateDir, "state-dir", "", "persistent module state directory")
	flag.StringVar(&runtimeDir, "runtime-dir", "", "private temporary WebUI runtime directory")
	flag.StringVar(&tokenFile, "token-file", "", "0600 bootstrap-token file")
	flag.StringVar(&statePath, "state-file", "", "atomic ready/state file")
	flag.StringVar(&pidPath, "pid-file", "", "PID file")
	flag.DurationVar(&idleTimeout, "idle-timeout", 15*time.Minute, "idle shutdown duration")
	flag.DurationVar(&sessionTTL, "session-ttl", 15*time.Minute, "browser session lifetime")
	flag.DurationVar(&jobTimeout, "job-timeout", 30*time.Minute, "maximum background job duration")
	flag.IntVar(&maxJobs, "max-jobs", 2, "maximum concurrent jobs")
	flag.BoolVar(&selfTest, "self-test", false, "validate the module adapter and exit")
	flag.Parse()

	logger := log.New(os.Stdout, "webui-server: ", log.LstdFlags|log.LUTC)

	if err := validatePaths(webroot, control, moduleDir, stateDir, runtimeDir); err != nil {
		logger.Fatal(err)
	}
	if idleTimeout < time.Minute || idleTimeout > time.Hour {
		logger.Fatal("idle timeout must be between 1m and 1h")
	}
	if sessionTTL < time.Minute || sessionTTL > time.Hour {
		logger.Fatal("session TTL must be between 1m and 1h")
	}
	if jobTimeout < time.Minute || jobTimeout > 24*time.Hour {
		logger.Fatal("job timeout must be between 1m and 24h")
	}
	if maxJobs < 1 || maxJobs > 4 {
		logger.Fatal("max-jobs must be between 1 and 4")
	}

	app := &application{
		webroot:        webroot,
		control:        control,
		moduleDir:      moduleDir,
		stateDir:       stateDir,
		runtimeDir:     runtimeDir,
		tokenFile:      tokenFile,
		started:        time.Now().UTC(),
		logger:         logger,
		configIndex:    make(map[string]configField),
		actionIndex:    make(map[string]actionDefinition),
		jobIndex:       make(map[string]jobDefinition),
		inventoryIndex: make(map[string]inventoryDefinition),
		jobTimeout:     jobTimeout,
		maxJobs:        maxJobs,
		jobs:           make(map[string]*jobState),
	}
	app.touch()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := app.loadCapabilities(ctx); err != nil {
		cancel()
		logger.Fatalf("capabilities: %v", err)
	}
	cancel()

	if selfTest {
		ctx, cancel = context.WithTimeout(context.Background(), statusControlTimeout)
		defer cancel()
		if _, err := app.runControl(ctx, maxControlOutput, "status"); err != nil {
			logger.Fatalf("status self-test: %v", err)
		}
		if _, err := app.runControl(ctx, maxControlOutput, "config-get"); err != nil && app.capabilities.Features["config"] {
			logger.Fatalf("config self-test: %v", err)
		}
		fmt.Printf("service=webui-server\nversion=%s\ncapability_schema=%s\nmodule_id=%s\nRESULT: WEBUI_SERVER_SELF_TEST_PASS\n",
			version, app.capabilities.Schema, app.capabilities.Module.ID)
		return
	}

	if err := validateListenAddress(listenAddress); err != nil {
		logger.Fatal(err)
	}
	token, err := readBootstrapToken(tokenFile)
	if err != nil {
		logger.Fatalf("bootstrap token: %v", err)
	}
	session, err := randomHex(32)
	if err != nil {
		logger.Fatalf("session: %v", err)
	}
	app.bootstrapToken = token
	app.session = session

	listener, err := net.Listen("tcp4", listenAddress)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	tcpAddress, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !tcpAddress.IP.IsLoopback() {
		logger.Fatal("listener is not loopback")
	}
	app.hostPort = net.JoinHostPort("127.0.0.1", strconv.Itoa(tcpAddress.Port))
	app.origin = "http://" + app.hostPort

	if err := writeRuntimeJSON(statePath, stateFile{
		Port:      tcpAddress.Port,
		PID:       os.Getpid(),
		Version:   version,
		StartedAt: app.started.Format(time.RFC3339Nano),
	}); err != nil {
		logger.Fatalf("state file: %v", err)
	}
	if err := writePIDFile(pidPath, os.Getpid()); err != nil {
		logger.Fatalf("pid file: %v", err)
	}
	defer os.Remove(statePath)
	defer os.Remove(pidPath)
	defer os.Remove(tokenFile)

	mux := http.NewServeMux()
	mux.HandleFunc("/bootstrap", app.bootstrap)
	mux.HandleFunc("/api/v1/health", app.health)
	mux.HandleFunc("/api/v1/capabilities", app.requireSession(app.capabilitiesHandler))
	mux.HandleFunc("/api/v1/status", app.requireSession(app.status))
	mux.HandleFunc("/api/v1/config", app.requireSession(app.config))
	mux.HandleFunc("/api/v1/log", app.requireSession(app.moduleLog))
	mux.HandleFunc("/api/v1/action", app.requireSession(app.action))
	mux.HandleFunc("/api/v1/jobs", app.requireSession(app.jobsHandler))
	mux.HandleFunc("/api/v1/jobs/", app.requireSession(app.jobHandler))
	mux.HandleFunc("/api/v1/inventory", app.requireSession(app.inventory))
	registerV03Handlers(mux, app)
	registerV04Handlers(mux, app)
	mux.HandleFunc("/", app.pageOrAsset)

	server := &http.Server{
		Handler:           app.securityHeaders(app.localOnly(mux)),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}

	shutdown := make(chan struct{})
	go app.idleMonitor(server, idleTimeout, shutdown)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-signals
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = server.Shutdown(ctx)
		cancel()
	}()

	logger.Printf("version=%s listen=%s idle_timeout=%s session_ttl=%s max_jobs=%d job_timeout=%s",
		version, listener.Addr(), idleTimeout, sessionTTL, maxJobs, jobTimeout)
	app.sessionCookieMaxAge = int(sessionTTL.Seconds())
	app.sessionExpires = time.Now().Add(sessionTTL)
	err = server.Serve(listener)
	close(shutdown)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("serve: %v", err)
	}
	logger.Printf("stopped")
}

func validatePaths(webroot, control, moduleDir, stateDir, runtimeDir string) error {
	for _, item := range []struct {
		name string
		path string
		dir  bool
	}{
		{"webroot", webroot, true},
		{"control", control, false},
		{"module-dir", moduleDir, true},
		{"state-dir", stateDir, true},
		{"runtime-dir", runtimeDir, true},
	} {
		if item.path == "" {
			return fmt.Errorf("%s is required", item.name)
		}
		info, err := os.Stat(item.path)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		if item.dir != info.IsDir() {
			return fmt.Errorf("%s has wrong file type", item.name)
		}
	}
	return nil
}

func validateListenAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid listen address: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() || host != "127.0.0.1" {
		return errors.New("listen address must use 127.0.0.1")
	}
	return nil
}

func readBootstrapToken(path string) (string, error) {
	if path == "" {
		return "", errors.New("token-file is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("token file must be a regular file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", errors.New("token file must not be group/world accessible")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if len(token) < 64 || len(token) > 128 {
		return "", errors.New("token length must be 64..128")
	}
	if _, err := hex.DecodeString(token); err != nil {
		return "", errors.New("token must be lowercase hexadecimal")
	}
	return token, nil
}

func randomHex(bytesCount int) (string, error) {
	data := make([]byte, bytesCount)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func (a *application) touch() {
	a.lastRequest.Store(time.Now().UnixNano())
}

func (a *application) activeJobCount() int64 {
	return a.activeJobs.Load()
}

func (a *application) idleMonitor(server *http.Server, idleTimeout time.Duration, shutdown <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if a.activeJobCount() > 0 {
				continue
			}
			last := time.Unix(0, a.lastRequest.Load())
			if time.Since(last) >= idleTimeout {
				a.logger.Printf("idle timeout reached")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = server.Shutdown(ctx)
				cancel()
				return
			}
		case <-shutdown:
			return
		}
	}
}

func (a *application) localOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != a.hostPort {
			http.Error(w, "host rejected", http.StatusForbidden)
			return
		}
		remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "invalid peer", http.StatusForbidden)
			return
		}
		remoteIP := net.ParseIP(remoteHost)
		if remoteIP == nil || !remoteIP.IsLoopback() {
			http.Error(w, "peer rejected", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *application) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), bluetooth=(), serial=()")
		next.ServeHTTP(w, r)
	})
}

var sessionCookieMaxAgeDefault = 900

func (a *application) bootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	a.bootstrapMu.Lock()
	defer a.bootstrapMu.Unlock()
	provided := r.URL.Query().Get("token")
	if a.bootstrapUsed || len(provided) != len(a.bootstrapToken) ||
		subtle.ConstantTimeCompare([]byte(provided), []byte(a.bootstrapToken)) != 1 {
		http.Error(w, "invalid or expired bootstrap token", http.StatusForbidden)
		return
	}
	a.bootstrapUsed = true
	a.bootstrapToken = ""
	a.touch()
	_ = os.Remove(a.tokenFile)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    a.session,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   a.sessionCookieMaxAgeValue(),
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// sessionCookieMaxAge is set after flag parsing. Kept on the application through
// this helper to make cookie policy testable without exposing the session value.
func (a *application) sessionCookieMaxAgeValue() int {
	if a.sessionCookieMaxAge > 0 {
		return a.sessionCookieMaxAge
	}
	return sessionCookieMaxAgeDefault
}

func (a *application) authenticated(r *http.Request) bool {
	if a.sessionExpires.IsZero() || !time.Now().Before(a.sessionExpires) {
		return false
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || len(cookie.Value) != len(a.session) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(a.session)) == 1
}

func (a *application) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authenticated(r) {
			writeJSON(w, http.StatusUnauthorized, responseEnvelope{OK: false, Error: "invalid or expired session"})
			return
		}
		a.touch()
		next(w, r)
	}
}

func (a *application) requireMutation(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return false
	}
	if r.Header.Get("Origin") != a.origin {
		writeJSON(w, http.StatusForbidden, responseEnvelope{OK: false, Error: "origin rejected"})
		return false
	}
	if r.Header.Get(requestGuardHeader) != "1" {
		writeJSON(w, http.StatusForbidden, responseEnvelope{OK: false, Error: "request guard missing"})
		return false
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, responseEnvelope{OK: false, Error: "application/json required"})
		return false
	}
	return true
}

func (a *application) pageOrAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, http.MethodGet, http.MethodHead)
		return
	}
	if !a.authenticated(r) {
		writeJSON(w, http.StatusUnauthorized, responseEnvelope{OK: false, Error: "invalid or expired session"})
		return
	}
	a.touch()
	cleaned := filepath.Clean("/" + r.URL.Path)
	if strings.Contains(cleaned, "/.") {
		http.NotFound(w, r)
		return
	}
	relative := strings.TrimPrefix(cleaned, "/")
	if relative == "" {
		relative = "index.html"
	}
	switch relative {
	case "index.html", "embedded-host-bootstrap.js", "mobile-input-viewport.js", "app.js", "app.css", "race-guard.js", "race-guard.css", "observability.js", "observability.css":
	default:
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(a.webroot, relative)
	data, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch filepath.Ext(relative) {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
}

func (a *application) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, responseEnvelope{OK: true, Service: "root-module-webui"})
}

func (a *application) capabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	document := a.capabilities
	writeJSON(w, http.StatusOK, responseEnvelope{OK: true, Capabilities: &document})
}

func (a *application) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), statusControlTimeout)
	defer cancel()
	output, err := a.runControl(ctx, maxControlOutput, "status")
	if err != nil {
		a.controlError(w, err)
		return
	}
	writeValidatedJSON(w, output)
}

func (a *application) config(w http.ResponseWriter, r *http.Request) {
	if !a.capabilities.Features["config"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "config capability disabled"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		output, err := a.runControl(ctx, maxControlOutput, "config-get")
		if err != nil {
			a.controlError(w, err)
			return
		}
		writeValidatedJSON(w, output)
	case http.MethodPost:
		if !a.requireMutation(w, r) {
			return
		}
		var request map[string]any
		if err := decodeJSON(w, r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
			return
		}
		normalized, err := a.validateConfig(request)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
			return
		}
		requestPath, err := a.writeRequestFile("config", normalized)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, responseEnvelope{OK: false, Error: "could not stage config request"})
			return
		}
		defer os.Remove(requestPath)
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		output, err := a.runControl(ctx, maxControlOutput, "config-apply", requestPath)
		if err != nil {
			a.controlError(w, err)
			return
		}
		writeValidatedJSON(w, output)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *application) moduleLog(w http.ResponseWriter, r *http.Request) {
	if !a.capabilities.Features["logs"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "log capability disabled"})
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	lines := 200
	if raw := r.URL.Query().Get("lines"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 1000 {
			writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "lines must be 1..1000"})
			return
		}
		lines = value
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	output, err := a.runControl(ctx, maxControlOutput, "log", strconv.Itoa(lines))
	if err != nil {
		a.controlError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(output)
}

func (a *application) action(w http.ResponseWriter, r *http.Request) {
	if !a.capabilities.Features["actions"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "action capability disabled"})
		return
	}
	if !a.requireMutation(w, r) {
		return
	}
	var request actionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
		return
	}
	definition, ok := a.actionIndex[request.Name]
	if !ok {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported action"})
		return
	}
	if request.DryRun && !definition.SupportsDryRun {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "action does not support dry-run"})
		return
	}
	if definition.RequiresConfirmation && request.Confirmation != definition.ConfirmationText {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "confirmation text does not match"})
		return
	}
	if definition.ApplyJob != "" && !request.DryRun {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "action apply must be started as declared job"})
		return
	}
	requestPath, err := a.writeRequestFile("action", request)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, responseEnvelope{OK: false, Error: "could not stage action request"})
		return
	}
	defer os.Remove(requestPath)
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	output, err := a.runControl(ctx, maxControlOutput, "action-file", request.Name, requestPath)
	if err != nil {
		a.controlError(w, err)
		return
	}
	writeValidatedJSON(w, output)
}

func (a *application) jobsHandler(w http.ResponseWriter, r *http.Request) {
	if !a.capabilities.Features["jobs"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "job capability disabled"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, responseEnvelope{OK: true, Data: a.listJobs()})
	case http.MethodPost:
		if !a.requireMutation(w, r) {
			return
		}
		var request jobRequest
		if err := decodeJSON(w, r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
			return
		}
		if _, ok := a.jobIndex[request.Name]; !ok {
			writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported job"})
			return
		}
		if a.activeJobCount() >= int64(a.maxJobs) {
			writeJSON(w, http.StatusConflict, responseEnvelope{OK: false, Error: "job capacity reached"})
			return
		}
		job, err := a.startJob(request.Name)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, responseEnvelope{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, responseEnvelope{OK: true, Data: job.snapshot()})
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *application) jobHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	relative := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
	parts := strings.Split(strings.Trim(relative, "/"), "/")
	if len(parts) < 1 || !jobIDPattern.MatchString(parts[0]) {
		http.NotFound(w, r)
		return
	}
	a.jobsMu.RLock()
	job := a.jobs[parts[0]]
	a.jobsMu.RUnlock()
	if job == nil {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 1 {
		writeJSON(w, http.StatusOK, responseEnvelope{OK: true, Data: job.snapshot()})
		return
	}
	if len(parts) == 2 && parts[1] == "output" {
		stream := r.URL.Query().Get("stream")
		if stream == "" {
			stream = "stdout"
		}
		if stream != "stdout" && stream != "stderr" {
			writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "stream must be stdout or stderr"})
			return
		}
		offset, limit, err := parseWindow(r, maxJobOutput)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: err.Error()})
			return
		}
		data := job.stdout.Bytes()
		if stream == "stderr" {
			data = job.stderr.Bytes()
		}
		if offset > len(data) {
			offset = len(data)
		}
		end := offset + limit
		if end > len(data) {
			end = len(data)
		}
		writeJSON(w, http.StatusOK, responseEnvelope{OK: true, Data: map[string]any{
			"stream":    stream,
			"offset":    offset,
			"next":      end,
			"total":     len(data),
			"truncated": job.stdout.Exceeded() || job.stderr.Exceeded(),
			"text":      string(data[offset:end]),
		}})
		return
	}
	http.NotFound(w, r)
}

func parseWindow(r *http.Request, maximum int) (int, int, error) {
	offset := 0
	limit := 16 << 10
	if raw := r.URL.Query().Get("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 || value > maximum {
			return 0, 0, errors.New("offset out of range")
		}
		offset = value
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 64<<10 {
			return 0, 0, errors.New("limit must be 1..65536")
		}
		limit = value
	}
	return offset, limit, nil
}

func (a *application) inventory(w http.ResponseWriter, r *http.Request) {
	if !a.capabilities.Features["inventory"] {
		writeJSON(w, http.StatusNotFound, responseEnvelope{OK: false, Error: "inventory capability disabled"})
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	name := r.URL.Query().Get("name")
	if _, ok := a.inventoryIndex[name]; !ok {
		writeJSON(w, http.StatusBadRequest, responseEnvelope{OK: false, Error: "unsupported inventory"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	output, err := a.runControl(ctx, maxInventoryOutput, "inventory", name)
	if err != nil {
		a.controlError(w, err)
		return
	}
	writeValidatedJSON(w, output)
}

func (a *application) loadCapabilities(ctx context.Context) error {
	output, err := a.runControl(ctx, maxControlOutput, "capabilities")
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&a.capabilities); err != nil {
		return fmt.Errorf("invalid capability JSON: %w", err)
	}
	if a.capabilities.Schema != "root-module-webui.capabilities.v1" {
		return errors.New("unsupported capability schema")
	}
	if !safeNamePattern.MatchString(a.capabilities.Module.ID) {
		return errors.New("invalid module id in capabilities")
	}
	for _, field := range a.capabilities.Config {
		if !safeNamePattern.MatchString(field.Key) {
			return fmt.Errorf("invalid config key: %s", field.Key)
		}
		if _, exists := a.configIndex[field.Key]; exists {
			return fmt.Errorf("duplicate config key: %s", field.Key)
		}
		switch field.Type {
		case "boolean", "integer", "string", "enum":
		default:
			return fmt.Errorf("unsupported config type: %s", field.Type)
		}
		if field.MaxLength < 0 || field.MaxLength > 4096 {
			return fmt.Errorf("invalid max_length for %s", field.Key)
		}
		if field.Pattern != "" {
			if _, err := regexp.Compile(field.Pattern); err != nil {
				return fmt.Errorf("invalid pattern for %s: %w", field.Key, err)
			}
		}
		a.configIndex[field.Key] = field
	}
	for _, action := range a.capabilities.Actions {
		if !safeNamePattern.MatchString(action.Name) {
			return fmt.Errorf("invalid action name: %s", action.Name)
		}
		if err := validateRisk(action.Risk); err != nil {
			return fmt.Errorf("action %s: %w", action.Name, err)
		}
		if action.RequiresConfirmation && (action.ConfirmationText == "" || len(action.ConfirmationText) > 128) {
			return fmt.Errorf("action %s has invalid confirmation text", action.Name)
		}
		if _, exists := a.actionIndex[action.Name]; exists {
			return fmt.Errorf("duplicate action: %s", action.Name)
		}
		a.actionIndex[action.Name] = action
	}
	for _, job := range a.capabilities.Jobs {
		if !safeNamePattern.MatchString(job.Name) {
			return fmt.Errorf("invalid job name: %s", job.Name)
		}
		if err := validateRisk(job.Risk); err != nil {
			return fmt.Errorf("job %s: %w", job.Name, err)
		}
		if _, exists := a.jobIndex[job.Name]; exists {
			return fmt.Errorf("duplicate job: %s", job.Name)
		}
		a.jobIndex[job.Name] = job
	}
	for _, inventory := range a.capabilities.Inventories {
		if !safeNamePattern.MatchString(inventory.Name) {
			return fmt.Errorf("invalid inventory name: %s", inventory.Name)
		}
		if _, exists := a.inventoryIndex[inventory.Name]; exists {
			return fmt.Errorf("duplicate inventory: %s", inventory.Name)
		}
		a.inventoryIndex[inventory.Name] = inventory
	}
	if err := a.validateActionJobBindings(); err != nil {
		return err
	}
	return nil
}

func (a *application) validateActionJobBindings() error {
	for _, action := range a.actionIndex {
		if action.ApplyJob == "" {
			continue
		}
		if !a.capabilities.Features["jobs"] {
			return fmt.Errorf("action %s apply_job requires jobs feature", action.Name)
		}
		if !action.SupportsDryRun {
			return fmt.Errorf("action %s apply_job requires supports_dry_run", action.Name)
		}
		if action.RequiresConfirmation {
			return fmt.Errorf("action %s apply_job cannot use base action confirmation", action.Name)
		}
		job, ok := a.jobIndex[action.ApplyJob]
		if !ok {
			return fmt.Errorf("action %s apply_job references undeclared job: %s", action.Name, action.ApplyJob)
		}
		if job.Risk != action.Risk {
			return fmt.Errorf("action %s apply_job risk must match job %s", action.Name, action.ApplyJob)
		}
	}
	return nil
}

func validateRisk(value string) error {
	switch value {
	case "safe", "caution", "danger":
		return nil
	default:
		return errors.New("risk must be safe, caution, or danger")
	}
}

func (a *application) validateConfig(request map[string]any) (map[string]any, error) {
	if len(request) != len(a.configIndex) {
		return nil, errors.New("config must contain exactly the declared fields")
	}
	normalized := make(map[string]any, len(request))
	for key, value := range request {
		field, ok := a.configIndex[key]
		if !ok {
			return nil, fmt.Errorf("unknown config field: %s", key)
		}
		switch field.Type {
		case "boolean":
			boolean, ok := value.(bool)
			if !ok {
				return nil, fmt.Errorf("%s must be boolean", key)
			}
			normalized[key] = boolean
		case "integer":
			number, ok := value.(json.Number)
			if !ok {
				return nil, fmt.Errorf("%s must be integer", key)
			}
			integer, err := number.Int64()
			if err != nil {
				return nil, fmt.Errorf("%s must be integer", key)
			}
			if field.Min != nil && integer < *field.Min {
				return nil, fmt.Errorf("%s is below minimum", key)
			}
			if field.Max != nil && integer > *field.Max {
				return nil, fmt.Errorf("%s is above maximum", key)
			}
			normalized[key] = integer
		case "string":
			text, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("%s must be string", key)
			}
			if field.Required && text == "" {
				return nil, fmt.Errorf("%s is required", key)
			}
			maxLength := field.MaxLength
			if maxLength == 0 {
				maxLength = 1024
			}
			if len(text) > maxLength {
				return nil, fmt.Errorf("%s exceeds maximum length", key)
			}
			if field.Pattern != "" {
				pattern := regexp.MustCompile(field.Pattern)
				if !pattern.MatchString(text) {
					return nil, fmt.Errorf("%s does not match required pattern", key)
				}
			}
			normalized[key] = text
		case "enum":
			text, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("%s must be string", key)
			}
			allowed := false
			for _, option := range field.Options {
				if text == option.Value {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, fmt.Errorf("%s has unsupported value", key)
			}
			normalized[key] = text
		}
	}
	return normalized, nil
}

func (a *application) writeRequestFile(prefix string, value any) (string, error) {
	requestDir := filepath.Join(a.runtimeDir, "requests")
	if err := os.MkdirAll(requestDir, 0o700); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(requestDir, prefix+"-*.json")
	if err != nil {
		return "", err
	}
	path := file.Name()
	_ = os.Chmod(path, 0o600)
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(true)
	writeErr := encoder.Encode(value)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = os.Remove(path)
		return "", writeErr
	}
	return path, nil
}

func (a *application) startJob(name string) (*jobState, error) {
	id, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	job := newJobState(id, name)
	a.jobsMu.Lock()
	a.pruneJobsLocked()
	a.jobs[id] = job
	a.jobsMu.Unlock()
	a.activeJobs.Add(1)
	go a.runJob(job)
	return job, nil
}

func (a *application) runJob(job *jobState) {
	job.mu.Lock()
	job.record.Status = "running"
	job.record.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	job.started = time.Now()
	job.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), a.jobTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, a.control, "job-run", job.record.Name)
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

func (a *application) listJobs() []jobRecord {
	a.jobsMu.RLock()
	records := make([]jobRecord, 0, len(a.jobs))
	for _, job := range a.jobs {
		records = append(records, job.snapshot())
	}
	a.jobsMu.RUnlock()
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt
	})
	return records
}

func (a *application) pruneJobsLocked() {
	if len(a.jobs) < 20 {
		return
	}
	type candidate struct {
		id        string
		createdAt string
		running   bool
	}
	candidates := make([]candidate, 0, len(a.jobs))
	for id, job := range a.jobs {
		snapshot := job.snapshot()
		candidates = append(candidates, candidate{
			id:        id,
			createdAt: snapshot.CreatedAt,
			running:   snapshot.Status == "queued" || snapshot.Status == "running",
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].createdAt < candidates[j].createdAt
	})
	for _, item := range candidates {
		if len(a.jobs) < 20 {
			return
		}
		if !item.running {
			delete(a.jobs, item.id)
		}
	}
}

func (a *application) controlEnvironment() []string {
	return append(os.Environ(),
		"MODULE_DIR="+a.moduleDir,
		"MODULE_STATE_DIR="+a.stateDir,
		"WEBUI_RUNTIME_DIR="+a.runtimeDir,
	)
}

func (a *application) runControl(ctx context.Context, limit int, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, a.control, args...)
	cmd.Env = a.controlEnvironment()
	var stdout limitedBuffer
	var stderr limitedBuffer
	stdout.limit = limit
	stderr.limit = 64 << 10
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if stdout.Exceeded() || stderr.Exceeded() {
		return nil, errors.New("control output exceeded limit")
	}
	if err != nil {
		message := strings.TrimSpace(string(stderr.Bytes()))
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("module-control failed: %s", message)
	}
	return stdout.Bytes(), nil
}

func (a *application) controlError(w http.ResponseWriter, err error) {
	a.logger.Printf("control error: %v", err)
	writeJSON(w, http.StatusBadGateway, responseEnvelope{OK: false, Error: "module backend failed"})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
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

func writeValidatedJSON(w http.ResponseWriter, output []byte) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		writeJSON(w, http.StatusBadGateway, responseEnvelope{OK: false, Error: "module backend returned invalid JSON"})
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeJSON(w, http.StatusMethodNotAllowed, responseEnvelope{OK: false, Error: "method not allowed"})
}

func writeRuntimeJSON(path string, value any) error {
	if path == "" {
		return errors.New("state-file is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp := path + ".tmp"
	file, err := os.OpenFile(temp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	writeErr := json.NewEncoder(file).Encode(value)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = os.Remove(temp)
		return writeErr
	}
	return os.Rename(temp, path)
}

func writePIDFile(path string, pid int) error {
	if path == "" {
		return errors.New("pid-file is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, []byte(strconv.Itoa(pid)+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

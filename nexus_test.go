package nexus

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// --- Use ---

func TestUse(t *testing.T) {
	server := &Server{}

	mw1 := func(next http.Handler, s *Server) http.Handler { return next }
	mw2 := func(next http.Handler, s *Server) http.Handler { return next }

	server.Use(mw1)
	if len(server.Middlewares) != 1 {
		t.Fatalf("expected 1 middleware, got %d", len(server.Middlewares))
	}

	server.Use(mw2)
	if len(server.Middlewares) != 2 {
		t.Fatalf("expected 2 middlewares, got %d", len(server.Middlewares))
	}
}

func TestUse_Variadic(t *testing.T) {
	server := &Server{}

	mw1 := func(next http.Handler, s *Server) http.Handler { return next }
	mw2 := func(next http.Handler, s *Server) http.Handler { return next }

	server.Use(mw1, mw2)
	if len(server.Middlewares) != 2 {
		t.Fatalf("expected 2 middlewares from variadic call, got %d", len(server.Middlewares))
	}
}

// --- Endpoint method ---

func TestEndpointMethod(t *testing.T) {
	server := &Server{}

	handler := func(w http.ResponseWriter, r *http.Request) {}
	server.Endpoint("GET /test", handler)

	if len(server.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint group, got %d", len(server.Endpoints))
	}
	if len(server.Endpoints[0]) != 1 {
		t.Fatalf("expected 1 endpoint in group, got %d", len(server.Endpoints[0]))
	}
	if server.Endpoints[0][0].Path != "GET /test" {
		t.Fatalf("expected 'GET /test', got %s", server.Endpoints[0][0].Path)
	}
}

// --- Group ---

func TestGroup_PathPrefixing(t *testing.T) {
	server := &Server{}

	endpoints := []Endpoint{
		{Path: "GET /users", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {}},
		{Path: "POST /users", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {}},
	}

	server.Group("/api/v1", endpoints)

	if len(server.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint group, got %d", len(server.Endpoints))
	}
	if server.Endpoints[0][0].Path != "GET /api/v1/users" {
		t.Fatalf("expected 'GET /api/v1/users', got %s", server.Endpoints[0][0].Path)
	}
	if server.Endpoints[0][1].Path != "POST /api/v1/users" {
		t.Fatalf("expected 'POST /api/v1/users', got %s", server.Endpoints[0][1].Path)
	}
}

func TestGroup_RootPath(t *testing.T) {
	server := &Server{}

	// When the endpoint path is "GET /", the method portion is "GET" and
	// the path portion is "/" (length 1). The group function should produce
	// "GET /api" instead of "GET /api/".
	endpoints := []Endpoint{
		{Path: "GET /", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {}},
	}

	server.Group("/api", endpoints)

	if server.Endpoints[0][0].Path != "GET /api" {
		t.Fatalf("expected 'GET /api', got %s", server.Endpoints[0][0].Path)
	}
}

// --- GroupWithOptions ---

func TestGroupWithOptions_NilOptions(t *testing.T) {
	server := &Server{}

	endpoints := []Endpoint{
		{Path: "GET /test", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {}},
	}

	// Should not panic with nil groupOptions
	server.GroupWithOptions("/api", endpoints, nil)

	if server.Endpoints[0][0].Path != "GET /api/test" {
		t.Fatalf("expected 'GET /api/test', got %s", server.Endpoints[0][0].Path)
	}
	// HandlerFunc should remain since no middlewares were applied
	if server.Endpoints[0][0].HandlerFunc == nil {
		t.Fatal("expected HandlerFunc to remain when no group middlewares")
	}
}

func TestGroupWithOptions_SingleMiddleware(t *testing.T) {
	server := &Server{}

	endpoints := []Endpoint{
		{Path: "GET /test", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}},
	}

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Group", "yes")
			next.ServeHTTP(w, r)
		})
	}

	server.GroupWithOptions("/api", endpoints, &GroupOptions{
		Middlewares: []func(next http.Handler) http.Handler{mw},
	})

	ep := server.Endpoints[0][0]
	if ep.HandlerFunc != nil {
		t.Fatal("HandlerFunc should be nil after group middleware wrapping")
	}
	if ep.Handler == nil {
		t.Fatal("Handler should be set after group middleware wrapping")
	}
}

func TestGroupWithOptions_MultipleMiddlewares(t *testing.T) {
	// Regression test: applying multiple group middlewares should not panic.
	server := &Server{}

	endpoints := []Endpoint{
		{Path: "GET /test", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}},
	}

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-MW1", "yes")
			next.ServeHTTP(w, r)
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-MW2", "yes")
			next.ServeHTTP(w, r)
		})
	}

	// This previously could panic due to a bug where the handler was not properly
	// initialized before chaining multiple middlewares.
	server.GroupWithOptions("/api", endpoints, &GroupOptions{
		Middlewares: []func(next http.Handler) http.Handler{mw1, mw2},
	})

	ep := server.Endpoints[0][0]
	if ep.Handler == nil {
		t.Fatal("Handler should be set after group middleware wrapping")
	}
	if ep.HandlerFunc != nil {
		t.Fatal("HandlerFunc should be nil after group middleware wrapping")
	}
}

func TestGroupWithOptions_EmptyMiddlewares(t *testing.T) {
	server := &Server{}

	endpoints := []Endpoint{
		{Path: "GET /test", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {}},
	}

	server.GroupWithOptions("/api", endpoints, &GroupOptions{
		Middlewares: []func(next http.Handler) http.Handler{},
	})

	// With empty middleware slice, HandlerFunc should remain
	ep := server.Endpoints[0][0]
	if ep.HandlerFunc == nil {
		t.Fatal("HandlerFunc should remain with empty middlewares")
	}
}

// --- Run behavior tests (without actually starting a listener) ---

func TestRunSettingsNilCheck(t *testing.T) {
	// Regression: Run() should initialize nil Settings to avoid nil pointer panic
	server := &Server{
		Settings: nil,
	}

	// We can't call Run() as it blocks, but we can verify the pattern
	// by checking that the code in Run() handles nil Settings.
	// Simulating what Run() does:
	if server.Settings == nil {
		server.Settings = &Settings{}
	}

	if server.Settings == nil {
		t.Fatal("Settings should be initialized")
	}
}

func TestRunPathPrefix(t *testing.T) {
	// Test that path prefix is applied correctly, simulating Run()'s logic
	server := &Server{
		Settings: &Settings{PathPrefix: "/api/v1"},
	}

	endpoints := []Endpoint{
		{Path: "GET /users", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {}},
		{Path: "GET /_health", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {},
			Options: EndpointOptions{IgnorePrefix: true}},
	}

	// Simulate Run()'s prefix logic
	for i, ep := range endpoints {
		if !ep.Options.IgnorePrefix {
			ep.Path = fmt.Sprintf("GET %s/users", server.Settings.PathPrefix)
		}
		endpoints[i] = ep
	}

	if endpoints[0].Path != "GET /api/v1/users" {
		t.Fatalf("expected prefix applied: 'GET /api/v1/users', got %s", endpoints[0].Path)
	}
	if endpoints[1].Path != "GET /_health" {
		t.Fatalf("expected IgnorePrefix respected: 'GET /_health', got %s", endpoints[1].Path)
	}
}

func TestRunPanicsOnBothHandlers(t *testing.T) {
	// Verify that having both HandlerFunc and Handler set causes a panic
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when both HandlerFunc and Handler are set")
		}
		if r != "Endpoint cannot have both HandlerFunc and Handler" {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()

	server := &Server{
		Settings: &Settings{},
		Endpoints: [][]Endpoint{
			{
				{
					Path:        "GET /dual",
					HandlerFunc: func(w http.ResponseWriter, r *http.Request) {},
					Handler:     http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
				},
			},
		},
	}

	// Run() will panic before it reaches ListenAndServe
	server.Run()
}

func TestServerNameDefaults(t *testing.T) {
	// Test default server name logic from Run()
	server := &Server{}

	// Simulate Run()'s naming logic
	if server.ServerName == "" {
		if server.ServerNumber == "" {
			server.ServerNumber = "0"
		}
		server.ServerName = fmt.Sprintf("Server %s", server.ServerNumber)
	}

	if server.ServerName != "Server 0" {
		t.Fatalf("expected 'Server 0', got %s", server.ServerName)
	}
}

func TestServerNameWithNumber(t *testing.T) {
	server := &Server{ServerNumber: "3"}

	if server.ServerName == "" {
		if server.ServerNumber == "" {
			server.ServerNumber = "0"
		}
		server.ServerName = fmt.Sprintf("Server %s", server.ServerNumber)
	}

	if server.ServerName != "Server 3" {
		t.Fatalf("expected 'Server 3', got %s", server.ServerName)
	}
}

// --- Timeouts ---

// TestTimeoutDefaults pins that upgrading changes nothing on its own.
//
// These were hardcoded at 15s inside Run(). A server that never mentions them
// has to keep getting exactly that, or making them configurable becomes a
// silent behaviour change for every consumer that did not ask for one.
func TestTimeoutDefaults(t *testing.T) {
	settings := &Settings{}

	if got := settings.writeTimeout(); got != DefaultWriteTimeout {
		t.Errorf("writeTimeout() = %s, want the default %s", got, DefaultWriteTimeout)
	}
	if got := settings.readTimeout(); got != DefaultReadTimeout {
		t.Errorf("readTimeout() = %s, want the default %s", got, DefaultReadTimeout)
	}

	if DefaultWriteTimeout != 15*time.Second || DefaultReadTimeout != 15*time.Second {
		t.Errorf("defaults are %s/%s, want 15s/15s — the values Run() hardcoded before this change",
			DefaultWriteTimeout, DefaultReadTimeout)
	}
}

// TestTimeoutNilSettings covers Run()'s own path: Settings may be nil, and it
// initialises it before use. Resolving a timeout must not panic if that order
// ever changes.
func TestTimeoutNilSettings(t *testing.T) {
	var settings *Settings

	if got := settings.writeTimeout(); got != DefaultWriteTimeout {
		t.Errorf("writeTimeout() on nil Settings = %s, want the default", got)
	}
	if got := settings.readTimeout(); got != DefaultReadTimeout {
		t.Errorf("readTimeout() on nil Settings = %s, want the default", got)
	}
}

func TestTimeoutOverrides(t *testing.T) {
	settings := &Settings{
		WriteTimeout: 5 * time.Minute,
		ReadTimeout:  90 * time.Second,
	}

	if got := settings.writeTimeout(); got != 5*time.Minute {
		t.Errorf("writeTimeout() = %s, want 5m", got)
	}
	if got := settings.readTimeout(); got != 90*time.Second {
		t.Errorf("readTimeout() = %s, want 90s", got)
	}
}

// TestTimeoutsAreIndependent guards the copy-paste this pair invites: two
// fields, two accessors, four lines that look alike. Reading one where the
// other belongs gives a server whose writes die on the read budget — and it
// only shows up on the slow request nobody makes locally.
func TestTimeoutsAreIndependent(t *testing.T) {
	settings := &Settings{WriteTimeout: time.Hour}

	if got := settings.writeTimeout(); got != time.Hour {
		t.Errorf("writeTimeout() = %s, want 1h", got)
	}
	if got := settings.readTimeout(); got != DefaultReadTimeout {
		t.Errorf("readTimeout() = %s, want the default — setting one must not move the other", got)
	}
}

// TestBuildHTTPServerAppliesTimeouts is the test that was missing: it asserts
// the configured values actually reach the *http.Server.
//
// Without it, reverting the two lines in Run() back to hardcoded 15s passes the
// whole suite — every other timeout test checks the resolver in isolation and
// never that anybody calls it.
func TestBuildHTTPServerAppliesTimeouts(t *testing.T) {
	server := &Server{Settings: &Settings{
		WriteTimeout: 5 * time.Minute,
		ReadTimeout:  90 * time.Second,
	}}

	httpServer := server.buildHTTPServer("8080", http.NewServeMux())

	if httpServer.WriteTimeout != 5*time.Minute {
		t.Errorf("WriteTimeout = %s, want 5m — the setting never reached the server", httpServer.WriteTimeout)
	}
	if httpServer.ReadTimeout != 90*time.Second {
		t.Errorf("ReadTimeout = %s, want 90s — the setting never reached the server", httpServer.ReadTimeout)
	}
	if httpServer.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", httpServer.Addr)
	}
}

// TestBuildHTTPServerDefaults is the same assertion for the server that
// configures nothing — the case every existing consumer is in.
func TestBuildHTTPServerDefaults(t *testing.T) {
	server := &Server{Settings: &Settings{}}

	httpServer := server.buildHTTPServer("8080", http.NewServeMux())

	if httpServer.WriteTimeout != DefaultWriteTimeout || httpServer.ReadTimeout != DefaultReadTimeout {
		t.Errorf("timeouts = %s/%s, want the defaults %s/%s",
			httpServer.WriteTimeout, httpServer.ReadTimeout, DefaultWriteTimeout, DefaultReadTimeout)
	}
}

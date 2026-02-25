package nexus

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- ApplyMiddlewares ---

func TestApplyMiddlewares_NoMiddlewares(t *testing.T) {
	server := &Server{}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.ApplyMiddlewares(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestApplyMiddlewares_DebugMode(t *testing.T) {
	server := &Server{
		Debug:          true,
		ServerName:     "DebugServer",
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /test"},
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.ApplyMiddlewares(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	// In debug mode, LogRequest is applied. Known endpoint should pass through.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 in debug mode for known endpoint, got %d", w.Code)
	}
}

func TestApplyMiddlewares_CustomMiddleware(t *testing.T) {
	headerMW := func(next http.Handler, server *Server) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "applied")
			next.ServeHTTP(w, r)
		})
	}

	server := &Server{
		Middlewares: []func(next http.Handler, server *Server) http.Handler{headerMW},
	}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.ApplyMiddlewares(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	if w.Header().Get("X-Custom") != "applied" {
		t.Fatal("custom middleware header not applied")
	}
}

func TestApplyMiddlewares_ReverseOrder(t *testing.T) {
	// Middlewares are applied in reverse order, so the first middleware
	// in the slice is the outermost (runs first on request).
	order := []string{}

	mw1 := func(next http.Handler, server *Server) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1")
			next.ServeHTTP(w, r)
		})
	}
	mw2 := func(next http.Handler, server *Server) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2")
			next.ServeHTTP(w, r)
		})
	}

	server := &Server{
		Middlewares: []func(next http.Handler, server *Server) http.Handler{mw1, mw2},
	}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	handler := server.ApplyMiddlewares(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	// mw1 is outermost (index 0 applied last in reverse loop), so runs first
	if len(order) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(order), order)
	}
	if order[0] != "mw1" || order[1] != "mw2" || order[2] != "handler" {
		t.Fatalf("expected [mw1, mw2, handler], got %v", order)
	}
}

// --- LogRequest ---

func TestLogRequest_KnownEndpoint(t *testing.T) {
	server := &Server{
		Debug:          true,
		ServerName:     "Test",
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /api/data"},
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := server.LogRequest(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/data", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("expected 'ok', got %s", w.Body.String())
	}
}

func TestLogRequest_UnknownEndpoint404(t *testing.T) {
	server := &Server{
		Debug:          true,
		ServerName:     "Test",
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /known"},
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.LogRequest(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/unknown", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown endpoint, got %d", w.Code)
	}
}

func TestLogRequest_HealthSkipsLogging(t *testing.T) {
	// /_health path should still pass through (just no logging).
	// We verify it still calls the inner handler by checking status.
	server := &Server{
		Debug:          true,
		ServerName:     "Test",
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /_health"},
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	})

	handler := server.LogRequest(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_health", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// --- ValidateSecret ---

func TestValidateSecret_PublicEndpointBypasses(t *testing.T) {
	server := &Server{
		Secret:         "mysecret",
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /public", Options: EndpointOptions{IsPublic: true}},
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("public"))
	})

	handler := server.ValidateSecret(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/public", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for public endpoint, got %d", w.Code)
	}
}

func TestValidateSecret_ValidSecret(t *testing.T) {
	server := &Server{
		Secret:         "mysecret",
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /private"},
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.ValidateSecret(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/private", nil)
	r.Header.Set("x-secret", "mysecret")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid secret, got %d", w.Code)
	}
}

func TestValidateSecret_InvalidSecret(t *testing.T) {
	server := &Server{
		Secret:         "mysecret",
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /private"},
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.ValidateSecret(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/private", nil)
	r.Header.Set("x-secret", "wrong")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid secret, got %d", w.Code)
	}
}

func TestValidateSecret_MissingSecret(t *testing.T) {
	server := &Server{
		Secret:         "mysecret",
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /private"},
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.ValidateSecret(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/private", nil)
	// No x-secret header
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with missing secret, got %d", w.Code)
	}
}

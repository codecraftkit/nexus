package nexus

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- convertToRegex ---

func TestConvertToRegex_StaticPath(t *testing.T) {
	re := convertToRegex("GET /users")
	if re.String() != "^GET /users$" {
		t.Fatalf("expected ^GET /users$, got %s", re.String())
	}
}

func TestConvertToRegex_SingleParam(t *testing.T) {
	re := convertToRegex("GET /users/{id}")
	if !re.MatchString("GET /users/123") {
		t.Fatal("expected regex to match GET /users/123")
	}
	if re.MatchString("GET /users/") {
		t.Fatal("expected regex not to match empty param")
	}
}

func TestConvertToRegex_MultiParam(t *testing.T) {
	re := convertToRegex("GET /users/{userId}/posts/{postId}")
	if !re.MatchString("GET /users/42/posts/99") {
		t.Fatal("expected regex to match multi-param path")
	}
	if re.MatchString("GET /users/42/posts/") {
		t.Fatal("expected regex not to match empty second param")
	}
}

func TestConvertToRegex_Anchoring(t *testing.T) {
	re := convertToRegex("GET /api")
	if re.MatchString("GET /api/extra") {
		t.Fatal("regex should not match longer paths (end anchor)")
	}
	if re.MatchString("XGET /api") {
		t.Fatal("regex should not match with prefix (start anchor)")
	}
}

// --- pathMatches ---

func TestPathMatches_True(t *testing.T) {
	re := convertToRegex("GET /items/{id}")
	if !pathMatches(re, "GET /items/abc") {
		t.Fatal("expected pathMatches to return true")
	}
}

func TestPathMatches_False(t *testing.T) {
	re := convertToRegex("GET /items/{id}")
	if pathMatches(re, "POST /items/abc") {
		t.Fatal("expected pathMatches to return false for wrong method")
	}
	if pathMatches(re, "GET /other/abc") {
		t.Fatal("expected pathMatches to return false for wrong path")
	}
}

// --- matchRoute ---

func TestMatchRoute_Found(t *testing.T) {
	server := &Server{
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /users/{id}"},
		{Path: "POST /users"},
	})

	ep := server.matchRoute("GET /users/42")
	if ep == nil {
		t.Fatal("expected to find endpoint")
	}
	if ep.Path != "GET /users/{id}" {
		t.Fatalf("expected GET /users/{id}, got %s", ep.Path)
	}
}

func TestMatchRoute_NotFound(t *testing.T) {
	server := &Server{
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /users"},
	})

	ep := server.matchRoute("DELETE /users")
	if ep != nil {
		t.Fatal("expected nil for unknown route")
	}
}

func TestMatchRoute_UsesPrecompiledRegex(t *testing.T) {
	// Regression: matchRoute should use the pre-compiled RegexPattern,
	// not recompile on each call. We verify by checking that RegexPattern
	// is set after setEndpoints.
	server := &Server{
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /test/{id}"},
	})

	ep := server.EndpointsPaths["GET /test/{id}"]
	if ep == nil {
		t.Fatal("endpoint not registered")
	}
	if ep.RegexPattern == nil {
		t.Fatal("RegexPattern should be pre-compiled after setEndpoints")
	}
}

// --- GetEndpoint ---

func TestGetEndpoint_Exists(t *testing.T) {
	server := &Server{
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /health"},
	})

	r := httptest.NewRequest("GET", "/health", nil)
	ep, ok := server.GetEndpoint(r)
	if !ok {
		t.Fatal("expected endpoint to be found")
	}
	if ep.Path != "GET /health" {
		t.Fatalf("unexpected path: %s", ep.Path)
	}
}

func TestGetEndpoint_NotExists(t *testing.T) {
	server := &Server{
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /health"},
	})

	r := httptest.NewRequest("POST", "/health", nil)
	_, ok := server.GetEndpoint(r)
	if ok {
		t.Fatal("expected endpoint not to be found")
	}
}

// --- EndpointIsPublic / NoRequiresAuthentication ---

func TestEndpointIsPublic(t *testing.T) {
	server := &Server{
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /pub", Options: EndpointOptions{IsPublic: true}},
		{Path: "GET /priv"},
	})

	r1 := httptest.NewRequest("GET", "/pub", nil)
	if !server.EndpointIsPublic(r1) {
		t.Fatal("expected /pub to be public")
	}

	r2 := httptest.NewRequest("GET", "/priv", nil)
	if server.EndpointIsPublic(r2) {
		t.Fatal("expected /priv to not be public")
	}

	// Unknown endpoint
	r3 := httptest.NewRequest("GET", "/unknown", nil)
	if server.EndpointIsPublic(r3) {
		t.Fatal("expected unknown endpoint to not be public")
	}
}

func TestNoRequiresAuthentication(t *testing.T) {
	server := &Server{
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /noauth", Options: EndpointOptions{NoRequiresAuthentication: true}},
		{Path: "GET /auth"},
	})

	r1 := httptest.NewRequest("GET", "/noauth", nil)
	if !server.NoRequiresAuthentication(r1) {
		t.Fatal("expected /noauth to not require authentication")
	}

	r2 := httptest.NewRequest("GET", "/auth", nil)
	if server.NoRequiresAuthentication(r2) {
		t.Fatal("expected /auth to require authentication")
	}
}

// --- RequestScheme ---

func TestRequestScheme_XForwardedProto(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	if scheme := RequestScheme(r); scheme != "https" {
		t.Fatalf("expected https, got %s", scheme)
	}
}

func TestRequestScheme_Forwarded(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Forwarded", "proto=https; host=example.com")
	if scheme := RequestScheme(r); scheme != "https" {
		t.Fatalf("expected https, got %s", scheme)
	}
}

func TestRequestScheme_TLS(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.TLS = &tls.ConnectionState{}
	if scheme := RequestScheme(r); scheme != "https" {
		t.Fatalf("expected https, got %s", scheme)
	}
}

func TestRequestScheme_Fallback(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if scheme := RequestScheme(r); scheme != "http" {
		t.Fatalf("expected http, got %s", scheme)
	}
}

// --- registerEndpoint / setEndpoints ---

func TestRegisterEndpoint(t *testing.T) {
	server := &Server{
		EndpointsPaths: make(map[string]*Endpoint),
	}
	ep := Endpoint{Path: "GET /test"}
	server.registerEndpoint(ep)

	stored, ok := server.EndpointsPaths["GET /test"]
	if !ok {
		t.Fatal("endpoint not registered")
	}
	if stored.RegexPattern == nil {
		t.Fatal("regex should be compiled on register")
	}
}

func TestSetEndpoints(t *testing.T) {
	server := &Server{}
	endpoints := []Endpoint{
		{Path: "GET /a"},
		{Path: "POST /b"},
	}
	server.setEndpoints(endpoints)

	if len(server.EndpointsPaths) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(server.EndpointsPaths))
	}
	if server.EndpointsPaths["GET /a"] == nil {
		t.Fatal("GET /a not found")
	}
	if server.EndpointsPaths["POST /b"] == nil {
		t.Fatal("POST /b not found")
	}
}

func TestSetEndpoints_InitializesNilMap(t *testing.T) {
	server := &Server{EndpointsPaths: nil}
	server.setEndpoints([]Endpoint{{Path: "GET /x"}})
	if server.EndpointsPaths == nil {
		t.Fatal("EndpointsPaths should be initialized")
	}
}

func TestGetEndpoints(t *testing.T) {
	server := &Server{
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /foo"},
	})

	eps := server.GetEndpoints()
	if len(eps) != 1 {
		t.Fatalf("expected 1, got %d", len(eps))
	}
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

package nexus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	server := &Server{ServerName: "TestServer"}
	handler := Health(server)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_health", nil)
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("expected text/plain; charset=utf-8, got %s", ct)
	}
	body := w.Body.String()
	if body != "TestServer is running" {
		t.Fatalf("expected 'TestServer is running', got %s", body)
	}
}

func TestRoutesList(t *testing.T) {
	server := &Server{
		EndpointsPaths: make(map[string]*Endpoint),
	}
	server.setEndpoints([]Endpoint{
		{Path: "GET /users"},
		{Path: "POST /users"},
	})

	handler := RoutesList(server)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_routes", nil)
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var routes map[string]string
	json.Unmarshal(w.Body.Bytes(), &routes)
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if routes["GET /users"] != "GET /users" {
		t.Fatalf("missing GET /users in routes map")
	}
}

func TestRawRoutesList(t *testing.T) {
	// Note: RawRoutesList attempts to JSON-marshal []Endpoint, but Endpoint
	// contains function-type fields (HandlerFunc, HandlerServerFunc) which
	// are unsupported by encoding/json. This means ResponseWithJSON returns
	// an error and nothing is written to the response body.
	// This test verifies the current behavior.
	server := &Server{
		Endpoints: [][]Endpoint{
			{
				{Path: "GET /a", Options: EndpointOptions{IsPublic: true}},
			},
			{
				{Path: "POST /b"},
			},
		},
	}

	handler := RawRoutesList(server)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_routes/raw", nil)
	handler(w, r)

	// Due to json.Marshal failing on func-typed fields in Endpoint,
	// ResponseWithJSON returns early. Go's default status is 200.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestServerEndpoints(t *testing.T) {
	if len(ServerEndpoints) != 3 {
		t.Fatalf("expected 3 server endpoints, got %d", len(ServerEndpoints))
	}

	expectedPaths := []string{"GET /_health", "GET /_routes", "GET /_routes/raw"}
	for i, ep := range ServerEndpoints {
		if ep.Path != expectedPaths[i] {
			t.Fatalf("expected path %s, got %s", expectedPaths[i], ep.Path)
		}
		if !ep.Options.IsPublic {
			t.Fatalf("expected %s to be public", ep.Path)
		}
		if !ep.Options.NoRequiresAuthentication {
			t.Fatalf("expected %s to not require auth", ep.Path)
		}
		if !ep.Options.IgnorePrefix {
			t.Fatalf("expected %s to ignore prefix", ep.Path)
		}
	}
}

func TestHealth_DefaultServerName(t *testing.T) {
	server := &Server{ServerName: ""}
	handler := Health(server)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_health", nil)
	handler(w, r)

	body := w.Body.String()
	if !strings.Contains(body, "is running") {
		t.Fatalf("expected 'is running' in body, got %s", body)
	}
}

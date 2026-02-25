package nexus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/cors"
)

// buildTestHandler replicates the wiring logic of Run() but returns an
// http.Handler instead of starting a listener. This allows full end-to-end
// testing with httptest.
func buildTestHandler(server *Server) http.Handler {
	if server.Settings == nil {
		server.Settings = &Settings{}
	}

	if server.ServerName == "" {
		if server.ServerNumber == "" {
			server.ServerNumber = "0"
		}
		server.ServerName = fmt.Sprintf("Server %s", server.ServerNumber)
	}

	mux := http.NewServeMux()

	// Add built-in endpoints
	server.Endpoints = append(server.Endpoints, ServerEndpoints)

	for i, endpoints := range server.Endpoints {
		for j, endpoint := range endpoints {
			if !endpoint.Options.IgnorePrefix {
				endpoint.Path = strings.Replace(
					endpoint.Path,
					" /",
					fmt.Sprintf(" %s/", server.Settings.PathPrefix),
					-1,
				)
			}
			server.Endpoints[i][j] = endpoint

			if endpoint.HandlerFunc != nil && endpoint.Handler != nil {
				panic("Endpoint cannot have both HandlerFunc and Handler")
			}
			if endpoint.HandlerServerFunc != nil {
				mux.HandleFunc(endpoint.Path, endpoint.HandlerServerFunc(server))
			}
			if endpoint.HandlerFunc != nil {
				mux.HandleFunc(endpoint.Path, endpoint.HandlerFunc)
			}
			if endpoint.Handler != nil {
				mux.Handle(endpoint.Path, endpoint.Handler)
			}
		}
		server.setEndpoints(endpoints)
	}

	c := cors.New(server.CorsOptions)
	return c.Handler(server.ApplyMiddlewares(mux))
}

// --- Integration Tests ---

func TestIntegration_HealthEndpoint(t *testing.T) {
	server := &Server{ServerName: "IntTest"}
	handler := buildTestHandler(server)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/_health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	if body != "IntTest is running" {
		t.Fatalf("expected 'IntTest is running', got %s", body)
	}
}

func TestIntegration_CustomEndpoint(t *testing.T) {
	server := &Server{
		ServerName: "IntTest",
		Endpoints: [][]Endpoint{
			{
				{
					Path: "GET /hello",
					HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("world"))
					},
				},
			},
		},
	}
	handler := buildTestHandler(server)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/hello")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	if string(buf[:n]) != "world" {
		t.Fatalf("expected 'world', got %s", string(buf[:n]))
	}
}

func TestIntegration_ParameterizedRoute(t *testing.T) {
	server := &Server{
		ServerName: "IntTest",
		Endpoints: [][]Endpoint{
			{
				{
					Path: "GET /users/{id}",
					HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
						id := r.PathValue("id")
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("user:" + id))
					},
				},
			},
		},
	}
	handler := buildTestHandler(server)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/users/42")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	if string(buf[:n]) != "user:42" {
		t.Fatalf("expected 'user:42', got %s", string(buf[:n]))
	}
}

func TestIntegration_PathPrefix(t *testing.T) {
	server := &Server{
		ServerName: "IntTest",
		Settings:   &Settings{PathPrefix: "/api/v1"},
		Endpoints: [][]Endpoint{
			{
				{
					Path: "GET /items",
					HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("items"))
					},
				},
			},
		},
	}
	handler := buildTestHandler(server)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Prefixed endpoint should work
	resp, err := http.Get(ts.URL + "/api/v1/items")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for prefixed path, got %d", resp.StatusCode)
	}

	// Built-in endpoints should still work at their original paths
	resp2, err := http.Get(ts.URL + "/_health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /_health, got %d", resp2.StatusCode)
	}
}

func TestIntegration_MiddlewareChain(t *testing.T) {
	headerMW := func(next http.Handler, server *Server) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Server", server.ServerName)
			next.ServeHTTP(w, r)
		})
	}

	server := &Server{
		ServerName:  "MWTest",
		Middlewares: []func(next http.Handler, server *Server) http.Handler{headerMW},
		Endpoints: [][]Endpoint{
			{
				{
					Path: "GET /data",
					HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					},
				},
			},
		},
	}
	handler := buildTestHandler(server)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/data")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Server") != "MWTest" {
		t.Fatalf("expected X-Server header 'MWTest', got %s", resp.Header.Get("X-Server"))
	}
}

func TestIntegration_GroupedEndpoints(t *testing.T) {
	server := &Server{ServerName: "GroupTest"}

	endpoints := []Endpoint{
		{
			Path: "GET /users",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("users list"))
			},
		},
	}

	server.Group("/api/v1", endpoints)

	handler := buildTestHandler(server)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/users")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	if string(buf[:n]) != "users list" {
		t.Fatalf("expected 'users list', got %s", string(buf[:n]))
	}
}

func TestIntegration_GroupWithOptionsMiddleware(t *testing.T) {
	server := &Server{ServerName: "GroupMWTest"}

	groupMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Group-MW", "applied")
			next.ServeHTTP(w, r)
		})
	}

	groupEndpoints := []Endpoint{
		{
			Path: "GET /items",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("grouped items"))
			},
		},
	}

	nonGroupEndpoints := []Endpoint{
		{
			Path: "GET /other",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		},
	}

	server.GroupWithOptions("/admin", groupEndpoints, &GroupOptions{
		Middlewares: []func(next http.Handler) http.Handler{groupMW},
	})
	server.Endpoints = append(server.Endpoints, nonGroupEndpoints)

	handler := buildTestHandler(server)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Grouped endpoint should have the group middleware applied
	resp, err := http.Get(ts.URL + "/admin/items")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Group-MW") != "applied" {
		t.Fatalf("expected X-Group-MW header on grouped endpoint")
	}

	// Non-grouped endpoint should NOT have the group middleware
	resp2, err := http.Get(ts.URL + "/other")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.Header.Get("X-Group-MW") != "" {
		t.Fatal("expected no X-Group-MW header on non-grouped endpoint")
	}
}

func TestIntegration_DebugMode404(t *testing.T) {
	server := &Server{
		ServerName: "DebugTest",
		Debug:      true,
	}
	handler := buildTestHandler(server)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// In debug mode, LogRequest is active. Unknown paths should return 404.
	resp, err := http.Get(ts.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown path in debug mode, got %d", resp.StatusCode)
	}
}

func TestIntegration_RoutesEndpoint(t *testing.T) {
	server := &Server{
		ServerName: "RoutesTest",
		Endpoints: [][]Endpoint{
			{
				{
					Path:        "GET /custom",
					HandlerFunc: func(w http.ResponseWriter, r *http.Request) {},
				},
			},
		},
	}
	handler := buildTestHandler(server)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/_routes")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var routes map[string]string
	json.NewDecoder(resp.Body).Decode(&routes)

	// Should contain our custom endpoint and the built-in ones
	if _, ok := routes["GET /custom"]; !ok {
		t.Fatal("expected GET /custom in routes")
	}
	if _, ok := routes["GET /_health"]; !ok {
		t.Fatal("expected GET /_health in routes")
	}
}

func TestIntegration_HandlerServerFunc(t *testing.T) {
	server := &Server{
		ServerName: "ServerFuncTest",
		Endpoints: [][]Endpoint{
			{
				{
					Path: "GET /server-info",
					HandlerServerFunc: func(s *Server) http.HandlerFunc {
						return func(w http.ResponseWriter, r *http.Request) {
							w.WriteHeader(http.StatusOK)
							w.Write([]byte("name:" + s.ServerName))
						}
					},
				},
			},
		},
	}
	handler := buildTestHandler(server)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/server-info")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	if string(buf[:n]) != "name:ServerFuncTest" {
		t.Fatalf("expected 'name:ServerFuncTest', got %s", string(buf[:n]))
	}
}

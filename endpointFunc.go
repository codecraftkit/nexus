package nexus

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// EndpointFunc contains the endpoint's functions

// EndpointIsPublic evalue if the endpoint is public
func (server *Server) EndpointIsPublic(r *http.Request) bool {
	endpoint, ok := server.GetEndpoint(r)
	return ok && endpoint.Options.IsPublic
}

func (server *Server) NoRequiresAuthentication(r *http.Request) bool {
	endpoint, ok := server.GetEndpoint(r)
	return ok && endpoint.Options.NoRequiresAuthentication
}

// GetEndpoint evaluate if a path exists in the endpoints and return the endpoint and a bool if exists
func (server *Server) GetEndpoint(r *http.Request) (*Endpoint, bool) {
	route := r.URL.Path
	method := r.Method
	path := fmt.Sprintf("%s %s", method, route)
	endpoint := server.matchRoute(path)
	if endpoint != nil {
		return endpoint, true
	}
	return nil, false
}

// GetEndpoints return all endpoints
func (server *Server) GetEndpoints() map[string]*Endpoint {
	return server.EndpointsPaths
}

// registerEndpoint add a endpoint to the endpoint's map
func (server *Server) registerEndpoint(endpoint Endpoint) {
	// Convertir a regex
	compiledRegex := convertToRegex(endpoint.Path)
	endpoint.RegexPattern = compiledRegex
	server.EndpointsPaths[endpoint.Path] = &endpoint
}

// setEndpoints add a list of endpoints to the endpoint's map
func (server *Server) setEndpoints(endpoints []Endpoint) {

	if server.EndpointsPaths == nil {
		server.EndpointsPaths = make(map[string]*Endpoint)
	}

	for _, endpoint := range endpoints {
		if server.Debug {
			fmt.Println(endpoint.Path)
		}
		server.registerEndpoint(endpoint)
	}
}

func convertToRegex(pattern string) *regexp.Regexp {

	// Buscar parámetros en la forma {param}
	re := regexp.MustCompile(`\{([^}]+)\}`)

	// Reemplazar {param} por regex y almacenar nombres de parámetros
	regexPattern := re.ReplaceAllStringFunc(pattern, func(match string) string {
		return `([^/]+)` // Grupo de captura para valores dinámicos
	})

	// Agregar inicio ^ y fin $ para coincidencia exacta
	fullRegex := "^" + regexPattern + "$"

	// Compilar regex final
	compiledRegex := regexp.MustCompile(fullRegex)

	return compiledRegex
}

func pathMatches(compiledRegex *regexp.Regexp, url string) bool {
	matches := compiledRegex.FindStringSubmatch(url)
	if matches == nil {
		return false
	}
	return true
}

func (server *Server) matchRoute(url string) *Endpoint {
	for _, ep := range server.EndpointsPaths {
		if ep.RegexPattern != nil && ep.RegexPattern.MatchString(url) {
			return ep
		}
	}
	return nil
}

func RequestScheme(r *http.Request) string {
	// 1) Encabezados estándar de proxies/CDN
	if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
		return xf
	}
	// 2) RFC 7239: Forwarded: proto=https; host=...
	if fwd := r.Header.Get("Forwarded"); fwd != "" {
		// búsqueda simple
		parts := strings.Split(fwd, ";")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(strings.ToLower(p), "proto=") {
				return strings.Trim(strings.SplitN(p, "=", 2)[1], `"`)
			}
		}
	}
	// 3) Conexión TLS local
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

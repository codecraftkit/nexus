package nexus

import (
	"fmt"
	"net/http"
	"regexp"
)

// EndpointFunc contains the endpoint's functions

// EndpointIsPublic evalue if the endpoint is public
func (server *Server) EndpointIsPublic(r *http.Request) bool {
	/**
	Evaluo si el path el publico para no evaluar secret
	*/
	endpoint, ok := server.GetEndpoint(r)

	if ok && endpoint.Options.IsPublic {
		return true
	}

	return false
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
	for endpoint := range server.EndpointsPaths {
		compiledRegex := convertToRegex(endpoint)
		if pathMatches(compiledRegex, url) {
			return server.EndpointsPaths[endpoint]
		}
	}
	return nil
}

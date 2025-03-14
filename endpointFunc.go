package nexus

import (
	"fmt"
	"net/http"
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
func (server *Server) GetEndpoint(r *http.Request) (Endpoint, bool) {
	route := r.URL.Path
	method := r.Method
	path := fmt.Sprintf("%s %s", method, route)
	endpoint, ok := server.EndpointsPaths[path]
	return endpoint, ok
}

// GetEndpoints return all endpoints
func (server *Server) GetEndpoints() map[string]Endpoint {
	return server.EndpointsPaths
}

// registerEndpoint add a endpoint to the endpoint's map
func (server *Server) registerEndpoint(endpoint Endpoint) {
	server.EndpointsPaths[endpoint.Path] = endpoint
}

// setEndpoints add a list of endpoints to the endpoint's map
func (server *Server) setEndpoints(endpoints []Endpoint) {

	if server.EndpointsPaths == nil {
		server.EndpointsPaths = make(map[string]Endpoint)
	}

	for _, endpoint := range endpoints {
		if server.Debug {
			fmt.Println(endpoint.Path)
		}
		server.registerEndpoint(endpoint)
	}
}

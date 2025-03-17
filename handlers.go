package nexus

import (
	"fmt"
	"net/http"
)

// Health check if the server is running
func Health(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%s is running", server.ServerName)))
	}
}

// RoutesList return all routes
func RoutesList(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var routes map[string]string = make(map[string]string)
		endpoints := server.GetEndpoints()

		for _, endpoint := range endpoints {
			//paths := strings.Split(endpoint.Path, " ")
			//pathName := strings.Replace(paths[1][1:], "/", "_", -1)
			//if pathName == "" {
			//	pathName = "index"
			//}
			routes[endpoint.Path] = endpoint.Path
		}

		ResponseWithJSON(w, http.StatusOK, routes)
	}
}

func RawRoutesList(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var routes []Endpoint

		for _, endpoints := range server.Endpoints {
			for _, endpoint := range endpoints {
				routes = append(routes, endpoint)
			}
		}

		fmt.Println(routes)

		ResponseWithJSON(w, http.StatusOK, routes)
	}
}

// ServerEndpoints is the list of endpoints for the server
var ServerEndpoints = []Endpoint{
	{Path: "GET /_health", HandlerServerFunc: Health, Options: EndpointOptions{IsPublic: true, NoRequiresAuthentication: true}},
	{Path: "GET /_routes", HandlerServerFunc: RoutesList, Options: EndpointOptions{IsPublic: true, NoRequiresAuthentication: true}},
	{Path: "GET /_routes/raw", HandlerServerFunc: RawRoutesList, Options: EndpointOptions{IsPublic: true, NoRequiresAuthentication: true}},
}

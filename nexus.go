package nexus

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/rs/cors"
)

// Run a new Server
func (server *Server) Run() {

	if server.Settings == nil {
		server.Settings = &Settings{}
	}

	// Server Name
	if server.ServerName == "" {
		if server.ServerNumber == "" {
			server.ServerNumber = "0"
		}
		server.ServerName = fmt.Sprintf("Server %s", server.ServerNumber)
	}

	mux := server.PrepareEndpoints()

	port := server.Port
	if port == "" {
		port = "8080"
	}

	c := cors.New(server.CorsOptions)

	httpServer := server.buildHTTPServer(port, c.Handler(
		server.ApplyMiddlewares(
			mux,
		),
	))

	if server.RunningServerMessage == "" {
		server.RunningServerMessage = fmt.Sprintf("[%s] Server running on port %s\n", server.ServerName, httpServer.Addr)
	}

	fmt.Print(server.RunningServerMessage)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}

// Serve set and run several Severs
func Serve(servers []*Server) {

	wg := sync.WaitGroup{}
	wg.Add(len(servers))

	for index, srv := range servers {
		go func(s *Server) {
			defer wg.Done()
			s.ServerNumber = fmt.Sprintf("%d", index)
			s.Run()
		}(srv)
	}

	wg.Wait()
}

// SetDebug set debug mode
func (server *Server) setDebug(debug bool) {
	server.Debug = debug
}

func (server *Server) Use(middlewares ...func(next http.Handler, server *Server) http.Handler) {
	server.Middlewares = append(server.Middlewares, middlewares...)
}

func (server *Server) Endpoint(path string, handler http.HandlerFunc) {
	endpoint := Endpoint{
		Path:        path,
		HandlerFunc: handler,
	}
	server.Endpoints = append(server.Endpoints, []Endpoint{endpoint})
}

func (server *Server) Group(group string, apiEndpoints []Endpoint) {

	for i, endpoint := range apiEndpoints {
		paths := strings.Split(endpoint.Path, " ")
		if len(paths[1]) == 1 {
			endpoint.Path = fmt.Sprintf("%s %s", paths[0], group)
		} else {
			endpoint.Path = strings.Replace(
				endpoint.Path,
				" /",
				fmt.Sprintf(" %s/", group),
				-1,
			)
		}
		apiEndpoints[i] = endpoint
	}

	server.Endpoints = append(server.Endpoints, apiEndpoints)

}

func (server *Server) GroupWithOptions(group string, apiEndpoints []Endpoint, groupOptions *GroupOptions) {

	for i, endpoint := range apiEndpoints {
		paths := strings.Split(endpoint.Path, " ")
		if len(paths[1]) == 1 {
			endpoint.Path = fmt.Sprintf("%s %s", paths[0], group)
		} else {
			endpoint.Path = strings.Replace(
				endpoint.Path,
				" /",
				fmt.Sprintf(" %s/", group),
				-1,
			)
		}

		if groupOptions != nil {
			middlewares := groupOptions.Middlewares

			if len(middlewares) > 0 {
				var handler http.Handler = endpoint.HandlerFunc
				for i := len(middlewares) - 1; i >= 0; i-- {
					handler = middlewares[i](handler)
				}
				endpoint.Handler = handler
				endpoint.HandlerFunc = nil
			}
		}

		apiEndpoints[i] = endpoint
	}

	server.Endpoints = append(server.Endpoints, apiEndpoints)

}

// buildHTTPServer assembles the *http.Server that Run() listens with.
//
// Split out so the wiring can be asserted. Run() ends in ListenAndServe and
// log.Fatal, so anything decided inline there is only reachable by starting a
// real server on a real port — which is how the timeouts stayed hardcoded
// without a test noticing.
func (server *Server) buildHTTPServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      handler,
		WriteTimeout: server.Settings.writeTimeout(),
		ReadTimeout:  server.Settings.readTimeout(),
	}
}

package nexus

import (
	"fmt"
	"github.com/rs/cors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Run a new Server
func (server *Server) Run() {

	// Server Name
	if server.ServerName == "" {
		if server.ServerNumber == "" {
			server.ServerNumber = "0"
		}
		server.ServerName = fmt.Sprintf("Server %s", server.ServerNumber)
	}

	mux := http.NewServeMux()

	// Add the basic endpoints from the library
	server.Endpoints = append(server.Endpoints, ServerEndpoints)

	// Add the endpoints from the user setup
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

			// If an endpoint has both Handler and HandlerFunc the server going to crash
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

	port := server.Port
	if port == "" {
		port = "8080"
	}

	c := cors.New(server.CorsOptions)

	httpServer := &http.Server{
		Addr: fmt.Sprintf(":%s", port),
		Handler: c.Handler(
			server.ApplyMiddlewares(
				mux,
			),
		),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if server.RunningServerMessage == "" {
		server.RunningServerMessage = fmt.Sprintf("[%s] Server running on port %s\n", server.ServerName, httpServer.Addr)
	}

	fmt.Printf(server.RunningServerMessage)
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
				for _, middleware := range middlewares {
					endpoint.Handler = middleware(http.Handler(endpoint.HandlerFunc))
					endpoint.HandlerFunc = nil
				}
			}
		}

		apiEndpoints[i] = endpoint
	}

	server.Endpoints = append(server.Endpoints, apiEndpoints)

}

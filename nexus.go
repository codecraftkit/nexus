package nexus

import (
	"fmt"
	"github.com/rs/cors"
	"log"
	"net/http"
	"sync"
	"time"
)

// Create and run new Server
func (server *Server) Create() {

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
	for _, endpoints := range server.Endpoints {

		server.setEndpoints(endpoints)

		for _, endpoint := range endpoints {

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
			s.Create()
		}(srv)
	}

	wg.Wait()
}

// SetDebug set debug mode
func (server *Server) setDebug(debug bool) {
	server.Debug = debug
}

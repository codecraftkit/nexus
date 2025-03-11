package nexus

import (
	"fmt"
	"github.com/rs/cors"
	"log"
	"net/http"
	"sync"
	"time"
)

type ServerStruct struct {
	ServerName           string
	ServerNumber         string
	RunningServerMessage string
	Secret               string
	Debug                bool
	Port                 string
	Middlewares          []func(next http.Handler, server *ServerStruct) http.Handler
	Endpoints            [][]EndpointPath
	EndpointsPaths       map[string]EndpointPath
	CorsOptions          cors.Options
}

type EndpointPath struct {
	Path                     string
	HandlerFunc              http.HandlerFunc
	Handler                  http.Handler
	HandlerServerFunc        func(server *ServerStruct) http.HandlerFunc
	IsPublic                 bool
	NoRequiresAuthentication bool
}

func (server *ServerStruct) Create() {

	server.EndpointsPaths = make(map[string]EndpointPath)

	if server.ServerName == "" {
		if server.ServerNumber == "" {
			server.ServerNumber = "0"
		}
		server.ServerName = fmt.Sprintf("Server %s", server.ServerNumber)
	}

	mux := http.NewServeMux()

	server.Endpoints = append(server.Endpoints, ServerEndpoints)

	for _, endpoints := range server.Endpoints {

		server.setEndpoints(endpoints)

		for _, endpoint := range endpoints {

			if endpoint.HandlerFunc != nil && endpoint.Handler != nil {
				panic("Endpoint cannot have both HandlerFunc and Handler")
				//log.Fatal("Endpoint cannot have both HandlerFunc and Handler")
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

func Serve(servers []*ServerStruct) {

	wg := sync.WaitGroup{}
	wg.Add(len(servers))

	for index, srv := range servers {
		go func(s *ServerStruct) {
			defer wg.Done()
			s.ServerNumber = fmt.Sprintf("%d", index)
			s.Create()
		}(srv)
	}

	wg.Wait()
}

func (server *ServerStruct) setDebug(debug bool) {
	server.Debug = debug
}

func (server *ServerStruct) setEndpoint(endpoint EndpointPath) {
	server.EndpointsPaths[endpoint.Path] = endpoint
	//server.Endpoints = append(server.Endpoints, []EndpointPath{endpoint})
}

func (server *ServerStruct) setEndpoints(endpoints []EndpointPath) {
	for _, endpoint := range endpoints {
		if server.Debug {
			fmt.Println(endpoint.Path)
		}
		server.setEndpoint(endpoint)
	}
}

func (server *ServerStruct) GetEndpoints() map[string]EndpointPath {
	return server.EndpointsPaths
}

func (server *ServerStruct) EndpointIsPublic(r *http.Request) bool {
	/**
	Evaluo si el path el publico para no evaluar secret
	*/
	route := r.URL.Path
	method := r.Method
	path := fmt.Sprintf("%s %s", method, route)
	endpoint, ok := server.EndpointsPaths[path]

	if ok && endpoint.IsPublic {
		return true
	}

	return false
}

func Health(server *ServerStruct) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%s is running", server.ServerName)))
	}
}

func RoutesList(server *ServerStruct) http.HandlerFunc {
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

var ServerEndpoints = []EndpointPath{
	{Path: "GET /_health", HandlerServerFunc: Health, IsPublic: true},
	{Path: "GET /_routes", HandlerServerFunc: RoutesList, IsPublic: true},
}

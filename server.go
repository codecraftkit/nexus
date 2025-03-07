package nexus

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

type ServerStruct struct {
	RunningServerMessage string
	Secret               string
	Debug                bool
	Port                 string
	Middlewares          []func(next http.Handler) http.Handler
	Endpoints            [][]EndpointPath
	EndpointsPaths       map[string]EndpointPath
}

type EndpointPath struct {
	Path                     string
	HandlerFunc              http.HandlerFunc
	Handler                  http.Handler
	IsPublic                 bool
	NoRequiresAuthentication bool
}

var Server ServerStruct = ServerStruct{}

func (server *ServerStruct) Create(serverSettings *ServerStruct) {

	server.RunningServerMessage = serverSettings.RunningServerMessage
	server.Secret = serverSettings.Secret
	server.Port = serverSettings.Port
	server.Debug = serverSettings.Debug
	server.Endpoints = serverSettings.Endpoints
	server.EndpointsPaths = make(map[string]EndpointPath)
	server.Middlewares = serverSettings.Middlewares

	mux := http.NewServeMux()

	server.Endpoints = append(server.Endpoints, ServerEndpoints)
	//server.setEndpoints(ServerEndpoints)

	for _, endpoints := range server.Endpoints {

		server.setEndpoints(endpoints)

		for _, endpoint := range endpoints {

			if endpoint.HandlerFunc != nil && endpoint.Handler != nil {
				panic("Endpoint cannot have both HandlerFunc and Handler")
				//log.Fatal("Endpoint cannot have both HandlerFunc and Handler")
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

	httpServer := &http.Server{
		Addr: fmt.Sprintf(":%s", port),
		Handler: ApplyMiddlewares(
			mux,
			server.Middlewares,
		),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if server.RunningServerMessage == "" {
		server.RunningServerMessage = fmt.Sprintf("Server running on port %s\n", httpServer.Addr)
	}

	fmt.Printf(server.RunningServerMessage)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
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

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func RoutesList(w http.ResponseWriter, r *http.Request) {
	var routes map[string]string = make(map[string]string)
	endpoints := Server.GetEndpoints()

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

var ServerEndpoints = []EndpointPath{
	{Path: "GET /_health", HandlerFunc: Health, IsPublic: true},
	{Path: "GET /_routes", HandlerFunc: RoutesList, IsPublic: true},
}

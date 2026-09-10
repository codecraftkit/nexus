package nexus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func preparedServer() *Server {
	return &Server{
		Settings: &Settings{PathPrefix: "/api/v1"},
		Endpoints: [][]Endpoint{{
			{Path: "GET /private/thing", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {}},
			{Path: "GET /open/thing", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {},
				Options: EndpointOptions{IsPublic: true, NoRequiresAuthentication: true}},
		}},
	}
}

// El motivo entero por el que este método existe: sin él, la única forma de
// llenar el índice de rutas era levantar el servidor, así que quien armaba su
// propio mux veía TODO como no público — y una afirmación del tipo "este grupo
// exige el secreto" pasaba por un motivo que no tenía nada que ver con el grupo.
func TestPrepareEndpointsFillsTheRouteIndexWithoutListening(t *testing.T) {
	server := preparedServer()

	// Antes de preparar, el índice está vacío y todo parece privado. Esta mitad
	// documenta el estado que producía el falso verde.
	before := httptest.NewRequest(http.MethodGet, "/api/v1/open/thing", nil)
	if server.EndpointIsPublic(before) || server.NoRequiresAuthentication(before) {
		t.Fatal("an unprepared server should know nothing about its routes")
	}

	mux := server.PrepareEndpoints()
	if mux == nil {
		t.Fatal("no mux came back")
	}

	open := httptest.NewRequest(http.MethodGet, "/api/v1/open/thing", nil)
	if !server.EndpointIsPublic(open) || !server.NoRequiresAuthentication(open) {
		t.Fatal("the public endpoint did not come back public")
	}
	private := httptest.NewRequest(http.MethodGet, "/api/v1/private/thing", nil)
	if server.EndpointIsPublic(private) || server.NoRequiresAuthentication(private) {
		t.Fatal("a private endpoint came back public")
	}
	// Y las opciones tienen que resolverse contra el path YA PREFIJADO: si el
	// índice guardara los paths sin prefijo, nunca matchearía una URL real y
	// todo volvería a parecer privado, en silencio.
	unprefixed := httptest.NewRequest(http.MethodGet, "/open/thing", nil)
	if server.EndpointIsPublic(unprefixed) {
		t.Fatal("the index is matching unprefixed paths")
	}
}

// Preparar dos veces no puede prefijar dos veces ni duplicar las rutas de la
// librería. Run llama a este método, así que un segundo llamado es alcanzable.
func TestPrepareEndpointsIsIdempotent(t *testing.T) {
	server := preparedServer()
	server.PrepareEndpoints()
	groups, routes := len(server.Endpoints), len(server.EndpointsPaths)

	server.PrepareEndpoints()

	if len(server.Endpoints) != groups {
		t.Fatalf("groups went from %d to %d", groups, len(server.Endpoints))
	}
	if len(server.EndpointsPaths) != routes {
		t.Fatalf("routes went from %d to %d", routes, len(server.EndpointsPaths))
	}
	for path := range server.EndpointsPaths {
		if strings.Contains(path, "/api/v1/api/v1") {
			t.Fatalf("the prefix was applied twice: %q", path)
		}
	}
	// Y el endpoint sigue resolviendo: una idempotencia que deja el índice
	// intacto pero roto no sirve de nada.
	if !server.EndpointIsPublic(httptest.NewRequest(http.MethodGet, "/api/v1/open/thing", nil)) {
		t.Fatal("preparing twice broke the index")
	}
}

// Las rutas de la librería se copian antes de prefijarse.
//
// Hoy las tres llevan IgnorePrefix, así que el prefijado no las toca y compartir
// el slice del paquete resulta inocuo. Esta prueba agrega una que NO lo lleva,
// que es el caso que la copia protege: sin ella, el prefijo del primer servidor
// aterriza en las rutas del segundo, y Serve corre varios a la vez.
//
// Restaurar ServerEndpoints al salir no es cortesía: es una variable del
// paquete, y dejarla sucia contaminaría a las demás pruebas del archivo.
func TestPreparingTwoServersDoesNotCrossContaminate(t *testing.T) {
	original := ServerEndpoints
	t.Cleanup(func() { ServerEndpoints = original })
	ServerEndpoints = append(append([]Endpoint(nil), original...),
		Endpoint{Path: "GET /_prefixed", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {},
			Options: EndpointOptions{IsPublic: true}})

	first := preparedServer()
	first.PrepareEndpoints()

	second := &Server{Settings: &Settings{PathPrefix: "/other"}}
	second.PrepareEndpoints()

	for path := range second.EndpointsPaths {
		if strings.Contains(path, "/api/v1") {
			t.Fatalf("the first server's prefix leaked into the second: %q", path)
		}
	}
	// Cada servidor tiene la ruta con SU prefijo, y las que ignoran el prefijo
	// siguen sin él en los dos.
	if !second.EndpointIsPublic(httptest.NewRequest(http.MethodGet, "/other/_prefixed", nil)) {
		t.Fatal("the second server did not get its own prefixed library route")
	}
	if !first.EndpointIsPublic(httptest.NewRequest(http.MethodGet, "/api/v1/_prefixed", nil)) {
		t.Fatal("the first server lost its prefixed library route")
	}
	for _, s := range []*Server{first, second} {
		if !s.EndpointIsPublic(httptest.NewRequest(http.MethodGet, "/_health", nil)) {
			t.Fatal("an IgnorePrefix route stopped resolving")
		}
	}
}

# nexus

**Nexus** is a Go library designed to simplify the creation of HTTP servers by leveraging Go's standard library. It provides developers with a streamlined interface to set up robust and efficient web servers without relying on external frameworks. By utilizing Go's native capabilities, Nexus ensures optimal performance and seamless integration within the Go ecosystem.

## Install

```bash
go get github.com/codecraftkit/nexus
```

## Quick Start

```go
package main

import (
	"net/http"
	"os"

	"github.com/codecraftkit/nexus"
)

func main() {
	server := &nexus.Server{
		ServerName: "MyApp",
		Port:       os.Getenv("PORT"),
		Debug:      true,
		Endpoints: [][]nexus.Endpoint{
			HomeEndpoints,
		},
	}

	server.Run()
}

var HomeEndpoints = []nexus.Endpoint{
	{Path: "GET /", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	}},
}
```

## Real-World Example

A common pattern is to organize handlers in separate packages with dependency injection. Each handler package registers its own routes via `server.Group()`:

### main.go

```go
package main

import "myapp/server"

func main() {
	server.InitServer().Run()
}
```

### server/server.go

```go
package server

import (
	"net/http"
	"os"

	"myapp/handlers/me"
	"myapp/handlers/users"
	"myapp/handlers/conversations"

	"github.com/codecraftkit/nexus"
	"github.com/rs/cors"
)

func InitServer() *nexus.Server {
	server := &nexus.Server{
		ServerName: "API",
		Port:       os.Getenv("PORT"),
		CorsOptions: cors.Options{
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
			AllowCredentials: true,
		},
		Settings: &nexus.Settings{
			PathPrefix: "/api/v1",
		},
	}

	// Global middlewares
	server.Use(LogMiddleware)
	server.Use(AuthMiddleware)

	// Register handler groups — each package registers its own routes
	userApp := &users.UserApplication{/* ... */}
	me.NewMeHandler(server, userApp)
	users.NewUsersHandler(server, userApp)
	conversations.NewConversationsHandler(server)

	return server
}

func LogMiddleware(next http.Handler, server *nexus.Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware(next http.Handler, server *nexus.Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

### handlers/users/handler.go

```go
package users

import "github.com/codecraftkit/nexus"

type UsersHandler struct {
	userApp UserApplicationPort
}

func NewUsersHandler(server *nexus.Server, userApp UserApplicationPort) {
	handler := &UsersHandler{userApp: userApp}

	server.Group("/users", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: handler.GetAll},
		{Path: "GET /{user_id}", HandlerFunc: handler.GetByID},
		{Path: "PUT /{user_id}/email", HandlerFunc: handler.UpdateEmail},
		{Path: "POST /ban", HandlerFunc: handler.ToBan},
		{Path: "DELETE /ban", HandlerFunc: handler.UnBan},
	})
}
```

### handlers/users/get_all.go

```go
package users

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

func (h *UsersHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	users, err := h.userApp.GetAll(r.Context())
	if err != nil {
		nexus.ResponseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	nexus.ResponseWithJSON(w, http.StatusOK, users)
}
```

### handlers/me/handler.go

```go
package me

import "github.com/codecraftkit/nexus"

type MeHandler struct {
	userApp UserApplicationPort
}

func NewMeHandler(server *nexus.Server, userApp UserApplicationPort) {
	handler := &MeHandler{userApp: userApp}

	server.Group("/me", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: handler.GetMe},
		{Path: "PUT /update-email", HandlerFunc: handler.UpdateEmail},
	})

	server.Group("/me/organizations", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: handler.GetOrganizations},
		{Path: "GET /owned", HandlerFunc: handler.GetOrganizationsOwned},
	})
}
```

### handlers/conversations/handler.go

```go
package conversations

import "github.com/codecraftkit/nexus"

type ConversationsHandler struct {
	// ...
}

func NewConversationsHandler(server *nexus.Server) {
	handler := &ConversationsHandler{}

	server.Group("/conversations", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: handler.GetAll},
		{Path: "DELETE /", HandlerFunc: handler.DeleteMany},
	})

	server.Group("/conversations/{conversation_id}", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: handler.GetByID},
		{Path: "DELETE /", HandlerFunc: handler.Delete},
	})

	server.Group("/conversations/{conversation_id}/security", []nexus.Endpoint{
		{Path: "POST /pin/validate", HandlerFunc: handler.ValidatePin},
	})
}
```

## Multiple Servers

Use `nexus.Serve()` to run multiple servers concurrently:

```go
func main() {
	apiServer := &nexus.Server{
		ServerName: "API",
		Port:       "8081",
		Settings:   &nexus.Settings{PathPrefix: "/api/v1"},
		Endpoints:  [][]nexus.Endpoint{ApiEndpoints},
	}

	adminServer := &nexus.Server{
		ServerName: "Admin",
		Port:       "8082",
		Settings:   &nexus.Settings{PathPrefix: "/admin"},
		Endpoints:  [][]nexus.Endpoint{AdminEndpoints},
	}

	nexus.Serve([]*nexus.Server{apiServer, adminServer})
}
```

## Server Configuration

| Field | Type | Description |
|---|---|---|
| `Port` | `string` | Port to listen on (defaults to `"8080"`) |
| `Debug` | `bool` | Enables request logging for all requests (except `/_health`) |
| `Secret` | `string` | Secret value for the `ValidateSecret` built-in middleware |
| `ServerName` | `string` | Name shown in logs (defaults to `"Server 0"`) |
| `RunningServerMessage` | `string` | Custom startup message |
| `Middlewares` | `[]func(next http.Handler, server *Server) http.Handler` | Global middleware chain |
| `Endpoints` | `[][]Endpoint` | Grouped endpoint slices |
| `CorsOptions` | `cors.Options` | CORS configuration (via `github.com/rs/cors`) |
| `Settings` | `*Settings` | Additional settings (see below) |

### Settings

```go
server := &nexus.Server{
	Settings: &nexus.Settings{
		PathPrefix:   "/api/v1", // Prepended to all endpoint paths
		IgnoreSecret: true,      // Disable secret validation

		// Server timeouts. Zero keeps the defaults below, so leaving them out
		// behaves exactly as before they were configurable.
		WriteTimeout: 30 * time.Second, // default: 15s
		ReadTimeout:  30 * time.Second, // default: 15s
	},
}
```

#### Streaming: do not disable the timeout server-wide

Zero means *"keep the default"*, not *"no timeout"* — on purpose. Turning the
write timeout off for the whole server so that one endpoint can stream lets a
single stuck client hold a connection forever, and removes the protection from
every other route to fix one.

For SSE, chunked downloads or any long-lived response, clear the deadline on
**that connection** instead. It is in the standard library:

```go
func streamHandler(w http.ResponseWriter, r *http.Request) {
	// Clears the write deadline for THIS connection only. The server default
	// keeps protecting every other route.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	// ... write events, Flush after each one
}
```

The `Settings` fields are for when the **whole service** needs a different
number — a gateway proxying slow upstreams, a service whose every route uploads.

## Endpoints

Endpoints use Go 1.22+ routing syntax: `"METHOD /path"`.

```go
var Endpoints = []nexus.Endpoint{
	{Path: "GET /users", HandlerFunc: GetUsers},
	{Path: "POST /users", HandlerFunc: CreateUser},
	{Path: "GET /users/{id}", HandlerFunc: GetUser},
	{Path: "PUT /users/{id}", HandlerFunc: UpdateUser},
	{Path: "DELETE /users/{id}", HandlerFunc: DeleteUser},
}
```

### Handler Types

Each endpoint supports three mutually exclusive handler types:

```go
// Standard http.HandlerFunc
{Path: "GET /home", HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("home"))
}}

// http.Handler (used internally by GroupWithOptions)
{Path: "GET /home", Handler: myHandler}

// Handler with access to the Server instance
{Path: "GET /status", HandlerServerFunc: func(server *nexus.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nexus.ResponseWithJSON(w, http.StatusOK, map[string]string{
			"server": server.ServerName,
		})
	}
}}
```

> Setting both `HandlerFunc` and `Handler` on the same endpoint will panic.

### Endpoint Options

```go
{
	Path:        "GET /public",
	HandlerFunc: PublicHandler,
	Options: nexus.EndpointOptions{
		IsPublic:                 true, // Skips secret validation
		NoRequiresAuthentication: true, // Marks as no-auth required
		IgnorePrefix:             true, // Ignores Settings.PathPrefix
	},
}
```

## Fluent API

Configure the server using methods instead of struct fields:

```go
server := &nexus.Server{Port: "8080"}

// Add middlewares
server.Use(AuthMiddleware, LogMiddleware)

// Add a single endpoint
server.Endpoint("GET /ping", func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
})

server.Run()
```

## Endpoint Groups

### Group

`Group()` prefixes all endpoint paths with the group string:

```go
server.Group("/users", []nexus.Endpoint{
	{Path: "GET /", HandlerFunc: handler.GetAll},             // → GET /users
	{Path: "GET /{user_id}", HandlerFunc: handler.GetByID},   // → GET /users/{user_id}
	{Path: "PUT /{user_id}/email", HandlerFunc: handler.UpdateEmail}, // → PUT /users/{user_id}/email
	{Path: "POST /ban", HandlerFunc: handler.ToBan},          // → POST /users/ban
})
```

You can use multiple groups with nested paths:

```go
server.Group("/conversations", []nexus.Endpoint{
	{Path: "GET /", HandlerFunc: handler.GetAll},
})

server.Group("/conversations/{conversation_id}", []nexus.Endpoint{
	{Path: "GET /", HandlerFunc: handler.GetByID},
	{Path: "DELETE /", HandlerFunc: handler.Delete},
})

server.Group("/conversations/{conversation_id}/security", []nexus.Endpoint{
	{Path: "POST /pin/validate", HandlerFunc: handler.ValidatePin},
})
```

### GroupWithOptions

`GroupWithOptions()` works like `Group()` but additionally wraps each endpoint with group-level middlewares:

```go
server.GroupWithOptions("/admin", AdminEndpoints, &nexus.GroupOptions{
	Middlewares: []func(next http.Handler) http.Handler{
		RequireAdmin,
	},
})
```

> Note: Group-level middlewares use the standard `func(next http.Handler) http.Handler` signature (without the server parameter).

## Middlewares

Nexus middlewares receive both the next handler and the server instance:

```go
func AuthMiddleware(next http.Handler, server *nexus.Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

Middlewares are applied in reverse order (last added wraps outermost). Register them via struct field or `Use()`:

```go
// Via struct
server := &nexus.Server{
	Middlewares: []func(next http.Handler, server *nexus.Server) http.Handler{
		AuthMiddleware,
	},
}

// Via Use() — can chain multiple at once
server.Use(ValidateSession)
server.Use(SetLanguage, GenerateCacheID, CachingMiddleware)
```

### Built-in Middlewares

- **LogRequest** — Automatically enabled when `Debug: true`. Logs `[ServerName] METHOD /path` for every request (except `/_health`).
- **ValidateSecret** — When `Secret` is set, checks the `x-secret` header on non-public endpoints.

## Response Helpers

```go
// JSON response
nexus.ResponseWithJSON(w, http.StatusOK, data)

// Error response (returns structured ErrorResponse)
nexus.ResponseWithError(w, http.StatusBadRequest, "invalid input")

// Error response with custom ErrorResponse
nexus.ResponseJsonWithError(w, http.StatusBadRequest, &nexus.ErrorResponse{
	Code:     400,
	Message:  "Validation failed",
	CodeName: "validation_error",
	Errors:   map[string]string{"name": "required"},
})

// Pagination helper — parses ?page=&limit= query params (defaults: page=1, limit=20)
skip, limit, page := nexus.GetOptions(r.URL.Query())

// Paginated response
nexus.ResponseWithPagination(w, http.StatusOK, users)
```

## Built-in Endpoints

Every server automatically includes these endpoints (they ignore `PathPrefix` and are always public):

| Endpoint | Description |
|---|---|
| `GET /_health` | Returns `"{ServerName} is running"` |
| `GET /_routes` | Returns a JSON map of all registered routes |
| `GET /_routes/raw` | Returns the full list of endpoints as JSON |
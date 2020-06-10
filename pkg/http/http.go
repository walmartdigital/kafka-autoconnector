package http

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server ...
type Server interface {
	ListenAndServe() error
}

// Router ...
type Router interface {
	HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) Route
}

// Route ...
type Route interface {
	Methods(methods ...string) Route
}

// RouterWrapper ...
type RouterWrapper struct {
	Router *mux.Router
}

// HandleFunc ...
func (r *RouterWrapper) HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) Route {
	return &routeWrapper{route: r.Router.HandleFunc(path, f)}
}

type routeWrapper struct {
	route *mux.Route
}

func (r *routeWrapper) Methods(methods ...string) Route {
	r.route.Methods(methods[0])
	return r
}

// WebServer ...
type WebServer struct {
	httpServer Server
	router     Router
}

// CreateServer ...
func CreateServer(server Server, router Router) *WebServer {
	instance := &WebServer{
		httpServer: server,
		router:     router,
	}
	return instance
}

// Run ...
func (s *WebServer) Run() {
	s.handleRequests()
}

func (s *WebServer) handleRequests() {
	s.router.HandleFunc("/metrics", promhttp.Handler().ServeHTTP).Methods("GET")
	err := s.httpServer.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

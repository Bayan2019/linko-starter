package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"boot.dev/linko/internal/store"
)

type server struct {
	httpServer *http.Server
	store      store.Store
	cancel     context.CancelFunc
}

func newServer(store store.Store, port int, cancel context.CancelFunc) *server {
	mux := http.NewServeMux()

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		// Ch 2. Logging Lv 3. Logging Requests
		// we can wrap the entire mux with the middleware, so that all requests are logged:
		Handler: requestLogger(logger)(mux),
	}

	s := &server{
		httpServer: srv,
		store:      store,
		cancel:     cancel,
	}

	mux.HandleFunc("GET /", s.handlerIndex)
	mux.Handle("POST /api/login", s.authMiddleware(http.HandlerFunc(s.handlerLogin)))
	mux.Handle("POST /api/shorten", s.authMiddleware(http.HandlerFunc(s.handlerShortenLink)))
	mux.Handle("GET /api/stats", s.authMiddleware(http.HandlerFunc(s.handlerStats)))
	mux.Handle("GET /api/urls", s.authMiddleware(http.HandlerFunc(s.handlerListURLs)))
	mux.HandleFunc("GET /{shortCode}", s.handlerRedirect)
	mux.HandleFunc("POST /admin/shutdown", s.handlerShutdown)

	return s
}

func (s *server) start() error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return err
	}

	// Ch 1. Observability Lv 3. What Is Observability?
	// When the server starts, print the following message to the console,
	// where %d is the port number:
	// ln.Addr() returns a net.Addr interface.
	if addr, ok := ln.Addr().(*net.TCPAddr); ok {
		httpPort := addr.Port
		logger.Printf("Linko is running on http://localhost:%d\n", httpPort)
	}

	if err := s.httpServer.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *server) shutdown(ctx context.Context) error {
	// Ch 1. Observability Lv 3. What Is Observability?
	// When the server shuts down (before it exits), print:
	logger.Println("Linko is shutting down")
	return s.httpServer.Shutdown(ctx)
}

func (s *server) handlerShutdown(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ENV") == "production" {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	go s.cancel()
}

////// accommodating functions
////// accommodating functions
////// accommodating functions
////// accommodating functions
////// accommodating functions
////// accommodating functions
////// accommodating functions
////// accommodating functions
////// accommodating functions
////// accommodating functions
////// accommodating functions

// Ch 2. Logging Lv 3. Logging Requests
// Implement the requestLogger middleware shown above,
// and update its log output to use this format:
func requestLogger(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			logger.Printf("Served request: %s %s", r.Method, r.URL.Path)
		})
	}
}

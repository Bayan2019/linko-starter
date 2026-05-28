package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"boot.dev/linko/internal/store"
)

// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
// Add a logger field to the server struct,
// and update server logging to use that injected logger.
type server struct {
	httpServer *http.Server
	store      store.Store
	cancel     context.CancelFunc
	logger     *slog.Logger
}

func newServer(
	store store.Store,
	port int,
	cancel context.CancelFunc,
	accessLogger *slog.Logger,
) *server {
	mux := http.NewServeMux()

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		// Ch 2. Logging Lv 3. Logging Requests
		// we can wrap the entire mux with the middleware, so that all requests are logged:
		// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
		// Use the access logger for server/request logs
		Handler: requestLogger(accessLogger)(mux),
	}

	s := &server{
		httpServer: srv,
		store:      store,
		cancel:     cancel,
		logger:     accessLogger,
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

type spyReadCloser struct {
	io.ReadCloser
	bytesRead int
}

func (r *spyReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	r.bytesRead += n
	return n, err
}

type spyResponseWriter struct {
	http.ResponseWriter
	bytesWritten int
	statusCode   int
}

func (w *spyResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytesWritten += n
	return n, err
}

func (w *spyResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
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
		s.logger.Info(fmt.Sprintf("Linko is running on http://localhost:%d\n", httpPort))
	}

	if err := s.httpServer.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *server) shutdown(ctx context.Context) error {
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
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			spyReader := &spyReadCloser{ReadCloser: r.Body}
			r.Body = spyReader
			spyWriter := &spyResponseWriter{ResponseWriter: w}
			next.ServeHTTP(spyWriter, r)

			// logger.Info(fmt.Sprintf("Served request: %s %s", r.Method, r.URL.Path))
			// Ch 3. Structured Logging Lv 5. Key-Value Pairs
			logger.Info(
				"Served request",
				"method", r.Method,
				"path", r.URL.Path,
				"client_ip", r.RemoteAddr,

				slog.Duration("duration", time.Since(start)),
				slog.Int("request_body_bytes", spyReader.bytesRead),

				slog.Int("response_status", spyWriter.statusCode),
				slog.Int("response_body_bytes", spyWriter.bytesWritten),
			)
		})
	}
}

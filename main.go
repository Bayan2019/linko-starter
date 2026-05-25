package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"boot.dev/linko/internal/store"
	"github.com/joho/godotenv"
	pkgerr "github.com/pkg/errors"
)

// Ch 2. Logging Lv 2. Use the Logger
// Create a global logger in main.go.
// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
// var mainLogger = log.New(
// 	// It should use os.Stderr,
// 	os.Stderr,
// 	// have a DEBUG: (with space) prefix,
// 	"DEBUG: ",
// 	// and use the standard log flags.
// 	log.LstdFlags,
// )

// Ch 2. Logging Lv 8. Logger Cleanup
// As you create your logger,
// also create a "close" function that cleans up any logger resources.
type closeFunc func() error

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("warning: assuming default configuration: .env unreadable: %v\n", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	httpPort := flag.Int("port", 8899, "port to listen on")
	dataDir := flag.String("data", "./data", "directory to store data")
	flag.Parse()

	status := run(ctx, cancel, *httpPort, *dataDir)
	cancel()
	os.Exit(status)
}

func run(ctx context.Context, cancel context.CancelFunc, httpPort int, dataDir string) int {

	// Ch 2. Logging Lv 5. Logger Configuration
	// Assume that in production,
	// Linko has a LINKO_LOG_FILE environment variable set.
	// In local development and staging, it is not set.
	var initializeLoggerFile = getEnv("LINKO_LOG_FILE", "")

	// Ch 2. Logging Lv 5. Logger Configuration
	// Add an initializeLogger helper.
	logger, close, err := initializeLogger(initializeLoggerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		return 1
	}
	// Ch 2. Logging Lv 8. Logger Cleanup
	// Call the close function before Linko exits.
	// defer a wrapper that calls it
	defer func() {
		if err := close(); err != nil {
			// and prints any cleanup error to STDERR.
			fmt.Fprintf(os.Stderr, "Failed to close logger: %v\n", err)
		}
	}()

	// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
	// Create two non-global loggers in run:
	// An "standard" logger
	// var standardLogger = log.New(
	// 	// writes to STDERR
	// 	os.Stderr,
	// 	// with an DEBUG: prefix
	// 	"DEBUG: ",
	// 	log.LstdFlags,
	// )
	// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
	// accessFile, err := os.OpenFile("linko.access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	// if err != nil {
	// 	standardLogger.Printf("Failed to open log file: %v", err)
	// 	return 1
	// }
	// defer accessFile.Close()

	// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
	// Create two non-global loggers in run:
	// An "access" logger
	// var accessLogger = log.New(
	// 	// writes to a file named linko.access.log
	// 	accessFile,
	// 	// with an INFO: prefix
	// 	"INFO: ",
	// 	log.LstdFlags,
	// )
	// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
	// use the standard logger for your Store and shutdown messages
	st, err := store.New(dataDir, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create store: %v\n", err))
		return 1
	}
	// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
	// Use the access logger for server/request logs
	s := newServer(*st, httpPort, cancel, logger)
	var serverErr error
	go func() {
		serverErr = s.start()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.shutdown(shutdownCtx); err != nil {
		logger.Error(fmt.Sprintf("failed to shutdown server: %v\n", err))
		return 1
	}

	// Ch 1. Observability Lv 3. What Is Observability?
	// When the server shuts down (before it exits), print:
	// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
	// use the standard logger for your Store and shutdown messages
	logger.Debug("Linko is shutting down")
	if serverErr != nil {
		logger.Error(fmt.Sprintf("server error: %v\n", serverErr))
		return 1
	}

	return 0
}

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	// Ch 3. Structured Logging Lv 3. Log Levels
	handlers := []slog.Handler{
		slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level:       slog.LevelDebug,
			ReplaceAttr: replaceAttr,
		}),
		slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level:       slog.LevelError,
			ReplaceAttr: replaceAttr,
		}),
	}
	// Ch 3. Structured Logging Lv 3. Log Levels
	closers := []closeFunc{}

	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file: %w", err)
		}
		// defer file.Close()
		// Ch 2. Logging Lv 7. Buffered Logging
		// wrap the file writer with bufio.NewWriterSize using an 8192 byte buffer.
		bufferedFile := bufio.NewWriterSize(file, 8192)
		// Ch 2. Logging Lv 8. Logger Cleanup
		// As you create your logger,
		// also create a "close" function
		// that cleans up any logger resources.
		close := func() error {
			// close function should .Flush the buffered writer
			if err := bufferedFile.Flush(); err != nil {
				return fmt.Errorf("failed to flush log file: %w", err)
			}
			// and .Close the file.
			if err := file.Close(); err != nil {
				return fmt.Errorf("failed to close log file: %w", err)
			}
			return nil
		}
		closers = append(closers, close)
		// multiWriter := io.MultiWriter(os.Stderr, bufferedFile)
		handlers = append(handlers, slog.NewJSONHandler(bufferedFile, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttr,
		}))

		// Ch 3. Structured Logging Lv 1. Slog Package
		// Update your logger type to *slog.Logger,
		// using slog.NewTextHandler.
		// You can use nil handler options for now.
		// logger := slog.New(slog.NewTextHandler(multiWriter, nil))

	}
	// Ch 2. Logging Lv 8. Logger Cleanup
	// For the STDERR logger, return a no-op close function that returns nil.
	// close = func() error {
	// 	return nil
	// }
	logger := slog.New(slog.NewMultiHandler(
		handlers...,
	))
	closer := func() error {
		var errs []error
		for _, close := range closers {
			if err := close(); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}

	// Ch 3. Structured Logging Lv 3. Log Levels
	// Use slog.Handlers
	// to configure your STDERR logs
	// to include DEBUG and above,
	// and your file logs to include INFO and above.
	// Use slog.NewMultiHandler to combine both handlers into one logger
	// used throughout the app.
	logger = slog.New(slog.NewMultiHandler(
		handlers...,
	))
	return logger, closer, nil
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

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		err, ok := a.Value.Any().(error)
		if !ok {
			return a
		}
		if stackErr, ok := errors.AsType[stackTracer](err); ok {
			return slog.GroupAttrs("error", slog.Attr{
				Key:   "message",
				Value: slog.StringValue(stackErr.Error()),
			}, slog.Attr{
				Key:   "stack_trace",
				Value: slog.StringValue(fmt.Sprintf("%+v", stackErr.StackTrace())),
			})
		}
	}
	return a
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

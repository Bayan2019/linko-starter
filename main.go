package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"boot.dev/linko/internal/store"
	"github.com/joho/godotenv"
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
	// Add an initializeLogger helper.
	var initializeLogger *log.Logger
	// Ch 2. Logging Lv 5. Logger Configuration
	// Assume that in production,
	// Linko has a LINKO_LOG_FILE environment variable set.
	// In local development and staging, it is not set.
	var initializeLoggerFile = getEnv("LINKO_LOG_FILE", "")
	// Ch 2. Logging Lv 5. Logger Configuration
	if initializeLoggerFile != "" {
		// If LINKO_LOG_FILE is set,
		// it should create a logger that writes to both the file and STDERR
		file, err := os.OpenFile(initializeLoggerFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}
		multiWriter := io.MultiWriter(os.Stderr, file)
		initializeLogger = log.New(multiWriter, "", log.LstdFlags)
	} else {
		initializeLogger = log.New(os.Stderr, "", log.LstdFlags)
	}
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
	st, err := store.New(dataDir, initializeLogger)
	if err != nil {
		initializeLogger.Printf("failed to create store: %v\n", err)
		return 1
	}
	// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
	// Use the access logger for server/request logs
	s := newServer(*st, httpPort, cancel, initializeLogger)
	var serverErr error
	go func() {
		serverErr = s.start()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.shutdown(shutdownCtx); err != nil {
		initializeLogger.Printf("failed to shutdown server: %v\n", err)
		return 1
	}

	// Ch 1. Observability Lv 3. What Is Observability?
	// When the server shuts down (before it exits), print:
	// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
	// use the standard logger for your Store and shutdown messages
	initializeLogger.Println("Linko is shutting down")
	if serverErr != nil {
		initializeLogger.Printf("server error: %v\n", serverErr)
		return 1
	}

	return 0
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

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

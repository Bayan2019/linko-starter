package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"boot.dev/linko/internal/build"
	"boot.dev/linko/internal/linkoerr"
	"boot.dev/linko/internal/store"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	isatty "github.com/mattn/go-isatty"

	// nethttppprof "net/http/pprof"
	pkgerr "github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"
)

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

	// fmt.Printf("status := run(ctx, cancel, *httpPort, *dataDir)\n")
	status := run(ctx, cancel, *httpPort, *dataDir)
	cancel()
	os.Exit(status)
}

func run(
	ctx context.Context,
	cancel context.CancelFunc,
	httpPort int,
	dataDir string,
) int {

	shutdownTracing, err := initTracing(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize tracing: %v\n", err)
		return 1
	}
	// fmt.Printf("initTracing(ctx)\n")
	defer func() {
		if err := shutdownTracing(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to shut down tracing: %v\n", err)
		}
	}()

	var initializeLoggerFile = getEnv("LINKO_LOG_FILE", "")

	// Ch 2. Logging Lv 5. Logger Configuration
	// Add an initializeLogger helper.
	logger, close, err := initializeLogger(initializeLoggerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		return 1
	}

	defer func() {
		if err := close(); err != nil {
			// and prints any cleanup error to STDERR.
			fmt.Fprintf(os.Stderr, "Failed to close logger: %v\n", err)
		}
	}()

	hostname, _ := os.Hostname()

	logger = logger.With(
		slog.String("git_sha", build.GitSHA),
		slog.String("build_time", build.BuildTime),
		slog.String("env", getEnv("ENV", "development")),
		slog.String("hostname", hostname),
	)

	st, err := store.New(dataDir, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create store: %v\n", err))
		return 1
	}
	// Ch 2. Logging Lv 4. Global Logger vs. Dependency Injection
	// Use the access logger for server/request logs
	s := newServer(*st, httpPort, cancel, logger)
	// fmt.Printf("s := newServer(*st, httpPort, cancel, logger)\n")
	var serverErr error
	go func() {
		// fmt.Printf("serverErr = s.start()\n")
		serverErr = s.start()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.shutdown(shutdownCtx); err != nil {
		logger.Error(fmt.Sprintf("failed to shutdown server: %v\n", err))
		return 1
	}

	logger.Debug("Linko is shutting down")
	if serverErr != nil {
		logger.Error(fmt.Sprintf("server error: %v\n", serverErr))
		return 1
	}

	return 0
}

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	// isTty := isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())

	// Ch 3. Structured Logging Lv 3. Log Levels
	handlers := []slog.Handler{
		// slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:       slog.LevelDebug,
			ReplaceAttr: replaceAttr,
			NoColor:     !(isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())),
		}),
		// slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:       slog.LevelError,
			ReplaceAttr: replaceAttr,
			NoColor:     !(isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())),
		}),
	}
	// Ch 3. Structured Logging Lv 3. Log Levels
	closers := []closeFunc{}

	if logFile != "" {
		rotatingFile := &lumberjack.Logger{
			Filename: logFile,
			// MaxSize:    1,
			MaxSize:    500,
			MaxAge:     28,
			MaxBackups: 10,
			LocalTime:  false,
			Compress:   true,
		}
		handlers = append(handlers, slog.NewJSONHandler(rotatingFile, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttr,
		}))
		close := func() error {
			if err := rotatingFile.Close(); err != nil {
				return fmt.Errorf("failed to close log file: %w", err)
			}
			return nil
		}
		closers = append(closers, close)
	}
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

type multiError interface {
	error
	Unwrap() []error
}

func errorAttrs(err error) []slog.Attr {
	attrs := []slog.Attr{
		{Key: "message", Value: slog.StringValue(err.Error())},
	}
	attrs = append(attrs, linkoerr.Attrs(err)...)
	if stackErr, ok := errors.AsType[stackTracer](err); ok {
		attrs = append(attrs, slog.Attr{
			Key:   "stack_trace",
			Value: slog.StringValue(fmt.Sprintf("%+v", stackErr.StackTrace())),
		})
	}
	return attrs
}

var sensitiveKeys = []string{"user", "password", "key", "apikey", "secret", "pin", "creditcardno"}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if slices.Contains(sensitiveKeys, a.Key) {
		return slog.String(a.Key, "[REDACTED]")
	}
	if a.Value.Kind() == slog.KindString {
		if u, err := url.Parse(a.Value.String()); err == nil {
			if _, hasPassword := u.User.Password(); hasPassword {
				u.User = url.UserPassword(u.User.Username(), "[REDACTED]")
				return slog.String(a.Key, u.String())
			}
		}
	}
	if a.Key == "error" {
		err, ok := a.Value.Any().(error)
		if !ok {
			return a
		}
		if multiErr, ok := errors.AsType[multiError](err); ok {
			var errAttrs []slog.Attr
			for i, e := range multiErr.Unwrap() {
				errAttrs = append(errAttrs, slog.GroupAttrs(fmt.Sprintf("error_%d", i+1), errorAttrs(e)...))
			}
			return slog.GroupAttrs("errors", errAttrs...)
		}

		return slog.GroupAttrs("error", errorAttrs(err)...)
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

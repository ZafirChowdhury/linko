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
	tint "github.com/lmittmann/tint"
	isatty "github.com/mattn/go-isatty"
	"github.com/natefinch/lumberjack"
	pkgerr "github.com/pkg/errors"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	httpPort := flag.Int("port", 8899, "port to listen on")
	dataDir := flag.String("data", "./data", "directory to store data")
	flag.Parse()

	status := run(ctx, cancel, *httpPort, *dataDir)
	cancel()
	os.Exit(status)
}

func run(ctx context.Context, cancel context.CancelFunc, httpPort int, dataDir string) int {
	env := os.Getenv("ENV")
	hostname, _ := os.Hostname()

	shutdown, err := initTracing(ctx)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "tracing shutdown: %v\n", err)
		}
	}()

	logger, closeLogger, err := initializeLogger(os.Getenv("LINKO_LOG_FILE"))

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		return 1
	}

	// build info
	logger = logger.With(
		slog.String("git_sha", build.GitSHA),
		slog.String("build_time", build.BuildTime),
		slog.String("env", env),
		slog.String("hostname", hostname),
	)

	defer func() {
		if err := closeLogger(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close logger: %v\n", err)
		}
	}()

	st, err := store.New(dataDir, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create store: %v", err))
		return 1
	}
	s := newServer(*st, httpPort, logger, cancel)
	var serverErr error
	go func() {
		serverErr = s.start()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger.Debug("Linko is shutting down")
	if err := s.shutdown(shutdownCtx); err != nil {
		logger.Error(fmt.Sprintf("failed to shutdown server: %v", err))
		return 1
	}
	if serverErr != nil {
		logger.Error(fmt.Sprintf("server error: %v", serverErr))
		return 1
	}
	return 0
}

type closeFunc func() error

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	colorEnabled :=
		isatty.IsTerminal(os.Stderr.Fd()) ||
			isatty.IsCygwinTerminal(os.Stderr.Fd())

	handlers := []slog.Handler{
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:       slog.LevelDebug,
			ReplaceAttr: replaceAttr,
			NoColor:     !colorEnabled,
		}),
	}

	closer := func() error { return nil }

	if logFile != "" {
		lj := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    1,
			MaxBackups: 10,
			MaxAge:     28,
			LocalTime:  false,
			Compress:   true,
		}
		handlers = append(handlers, slog.NewJSONHandler(lj, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttr,
		}))
		closer = lj.Close
	}

	return slog.New(slog.NewMultiHandler(handlers...)), closer, nil
}

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

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
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

	var sensitiveKeys = []string{"password", "key", "apikey", "secret", "pin", "creditcardno", "user"}
	if slices.Contains(sensitiveKeys, a.Key) {
		return slog.String(a.Key, "[REDACTED]")
	}

	if a.Value.Kind() == slog.KindString {
		u, err := url.Parse(a.Value.String())
		if err != nil {
			return a
		}

		_, ok := u.User.Password()
		if !ok {
			return a
		}

		u.User = url.UserPassword(u.User.Username(), "[REDACTED]")
		return slog.String(a.Key, u.String())
	}

	return a
}

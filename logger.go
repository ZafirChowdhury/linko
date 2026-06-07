package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
)

// very proud of the dog ass code I wrote
// but passing test requeres diffrent interface
// so not removing it
// dead code not used
func initializeLoggerOld() (*slog.Logger, func() error) {
	// noop = no op
	noop := func() error { return nil }

	fileName, ok := os.LookupEnv("LINKO_LOG_FILE")
	if !ok {
		fmt.Println("LOG: LINKO_LOG_FILE not found")
		debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:       slog.LevelDebug,
			ReplaceAttr: replaceStackTracePrint,
		})

		return slog.New(debugHandler), noop
	}

	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Println(err.Error())
		return nil, noop
	}

	bufferedFile := bufio.NewWriterSize(file, 8192)

	fmt.Println("LOG: LINKO_LOG_FILE found")
	closeBuffer := func() error {
		if err := bufferedFile.Flush(); err != nil {
			return err
		}

		file.Close()
		return nil
	}

	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceStackTracePrint,
	})

	infoHandler := slog.NewJSONHandler(bufferedFile, &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: replaceStackTracePrint,
	})

	return slog.New(slog.NewMultiHandler(debugHandler, infoHandler)), closeBuffer
}

func replaceStackTracePrint(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		err, ok := a.Value.Any().(error)
		if !ok {
			return a
		}
		return slog.String("error", fmt.Sprintf("%+v", err))
	}
	return a
}

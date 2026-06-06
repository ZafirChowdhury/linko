package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
)

func initializeLogger() (*slog.Logger, func() error) {
	// noop = no op
	noop := func() error { return nil }

	fileName, ok := os.LookupEnv("LINKO_LOG_FILE")
	if !ok {
		fmt.Println("LOG: LINKO_LOG_FILE not found")
		debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
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
		Level: slog.LevelDebug,
	})

	infoHandler := slog.NewJSONHandler(bufferedFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(slog.NewMultiHandler(debugHandler, infoHandler)), closeBuffer
}

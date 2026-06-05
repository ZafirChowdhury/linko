package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
)

func initializeLogger() (*slog.Logger, func() error) {
	// noop = no op
	noop := func() error { return nil }

	fileName, ok := os.LookupEnv("LINKO_LOG_FILE")
	if !ok {
		fmt.Println("LOG: LINKO_LOG_FILE not found")
		return slog.New(slog.NewTextHandler(os.Stderr, nil)), noop
	}

	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	bufferedFile := bufio.NewWriterSize(file, 8192)
	if err != nil {
		fmt.Println(err.Error())
		return nil, noop
	}
	fmt.Println("LOG: LINKO_LOG_FILE found")
	mw := io.MultiWriter(os.Stderr, bufferedFile)

	closeBuffer := func() error {
		if err := bufferedFile.Flush(); err != nil {
			return err
		}

		file.Close()
		return nil
	}

	return slog.New(slog.NewTextHandler(mw, nil)), closeBuffer
}

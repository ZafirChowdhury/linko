package main

import (
	"bufio"
	"io"
	"log"
	"os"
)

func initializeLogger() (*log.Logger, func() error) {
	// noop = no op
	noop := func() error { return nil }

	fileName, ok := os.LookupEnv("LINKO_LOG_FILE")
	if !ok {
		log.Println("LOG: LINKO_LOG_FILE not found")
		return log.New(os.Stderr, "LOG: ", log.LstdFlags), noop
	}

	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	bufferedFile := bufio.NewWriterSize(file, 8192)
	if err != nil {
		log.Println(err.Error())
		return nil, noop
	}
	log.Println("LOG: LINKO_LOG_FILE found")
	mw := io.MultiWriter(os.Stderr, bufferedFile)

	closeBuffer := func() error {
		if err := bufferedFile.Flush(); err != nil {
			return err
		}

		file.Close()
		return nil
	}

	return log.New(mw, "LOG: ", log.LstdFlags), closeBuffer
}

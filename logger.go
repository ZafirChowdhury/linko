package main

import (
	"io"
	"log"
	"os"
)

func initializeLogger() *log.Logger {
	fileName, ok := os.LookupEnv("LINKO_LOG_FILE")
	if !ok {
		log.Println("LOG: LINKO_LOG_FILE not found")
		return log.New(os.Stderr, "LOG: ", log.LstdFlags)
	}

	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatal(err.Error())
		return nil
	}
	log.Println("LOG: LINKO_LOG_FILE found")
	mw := io.MultiWriter(os.Stderr, file)
	return log.New(mw, "LOG: ", log.LstdFlags)
}

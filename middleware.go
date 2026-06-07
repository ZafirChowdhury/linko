package main

import (
	"log/slog"
	"net/http"
)

// very proud of the dog ass code I wrote
// but passing test requeres diffrent interface
// so not removing it
// dead code not used
func requestLoggerOld(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			logger.Info("Served request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.EscapedPath()),
				slog.String("client_ip", r.RemoteAddr),
			)

		})
	}
}

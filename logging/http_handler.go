package logging

import (
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
)

type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func LoggingMiddleware(h http.Handler, logger logr.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &StatusRecorder{
			ResponseWriter: w,
			Status:         200,
		}
		h.ServeHTTP(recorder, r)
		logger.Info(fmt.Sprintf("Handling request for %s from %s, status: %d", r.URL.Path, r.RemoteAddr, recorder.Status))
	})
}

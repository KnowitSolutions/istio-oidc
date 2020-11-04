package telemetry

import (
	"net/http"
)

func RegisterProbes(mux *http.ServeMux, init <-chan struct{}) {
	mux.HandleFunc("/health", health)

	rdy := ready{}
	mux.Handle("/ready", &rdy)
	go rdy.wait(init)
}

func health(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("Healthy"))
}

type ready struct {
	ready bool
}

func (r ready) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	if r.ready {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("Ready"))
	} else {
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte("Unavailable"))
	}
}

func (r *ready) wait(init <-chan struct{}) {
	<-init
	r.ready = true
}
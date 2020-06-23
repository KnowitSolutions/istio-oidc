package introspection

import (
	"net/http"
)

func RegisterProbes(mux *http.ServeMux) {
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/ready", ready)
}

func health(http.ResponseWriter, *http.Request) {}

func ready(http.ResponseWriter, *http.Request) {}

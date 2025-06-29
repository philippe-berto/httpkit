package utils

import (
	"net/http"
	"slices"

	otelcodes "go.opentelemetry.io/otel/codes"
)

const (
	OKStatusMsg       = "0K"
	ClientStatusMsg   = "Client Error"
	ServerStatusMsg   = "Server Error"
	DefaultrStatusMsg = "Unhandled Status"
)

var invalidPaths = []string{"/metrics", "/status", "/ready", "/", "/*"}

type StatusWriter struct {
	http.ResponseWriter
	StatusCode int
}

func CheckInValidPath(r *http.Request) bool {
	return slices.Contains(invalidPaths, r.URL.Path)
}

func (sw *StatusWriter) WriteHeader(code int) {
	sw.StatusCode = code
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *StatusWriter) GetStatus() (otelcodes.Code, string) {
	if sw.StatusCode >= 200 && sw.StatusCode < 300 {
		return otelcodes.Ok, OKStatusMsg
	}

	if sw.StatusCode >= 400 && sw.StatusCode < 500 {
		return otelcodes.Error, ClientStatusMsg
	}

	if sw.StatusCode >= 500 {
		return otelcodes.Error, ServerStatusMsg
	}

	return otelcodes.Unset, DefaultrStatusMsg
}

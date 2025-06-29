package tracing

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/philippe-berto/httpkit/utils"
)

const (
	tracerName = "github.com/AudioStreamTV/api-v2-package/tracing"
)

func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if utils.CheckInValidPath(r) {
			next.ServeHTTP(w, r)

			return
		}

		defaultCtx := baggage.ContextWithoutBaggage(r.Context())
		ww := &utils.StatusWriter{ResponseWriter: w, StatusCode: http.StatusOK}

		// Start a new span for the request
		tracer := otel.GetTracerProvider().Tracer(tracerName)
		ctx, span := tracer.Start(defaultCtx, r.URL.Path)
		defer span.End()

		next.ServeHTTP(ww, r.WithContext(ctx))

		routePattern := chi.RouteContext(defaultCtx).RoutePattern()
		span.SetStatus(ww.GetStatus())
		span.SetName(routePattern)
		span.SetAttributes(
			attribute.Key("extra_path").String(r.URL.Path),
			semconv.HTTPStatusCode(ww.StatusCode),
			semconv.HTTPMethod(r.Method),
			semconv.HTTPURL(getFullURL(r)),
		)
	})
}

func getFullURL(r *http.Request) string {
	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	return scheme + "://" + r.Host + r.RequestURI
}

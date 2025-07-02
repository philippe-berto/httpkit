package httpkit

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/philippe-berto/httpkit/metrics"
	"github.com/philippe-berto/httpkit/tracing"
	"github.com/philippe-berto/httpkit/utils"
)

var CorsAllowOrigins string

type (
	Handler struct {
		server *http.Server
		Router *chi.Mux
	}

	SubDomain struct {
		Domain string
		Router chi.Router
	}
)

func New(port int, tracerEnable, metricsEnable, setCors bool, corsAllowOrigins string, subdomains ...*SubDomain) *Handler {
	router := chi.NewRouter()

	router.Use(chimiddleware.StripSlashes)

	if metricsEnable {
		router.Use(metrics.MetricsMiddleware)
	}

	if tracerEnable {
		router.Use(tracing.TracingMiddleware)
	}

	if setCors {
		CorsAllowOrigins = corsAllowOrigins
		router.Use(cors)
	}

	router.Use(chimiddleware.RealIP)
	router.NotFoundHandler()
	router.MethodNotAllowedHandler()

	router.Get("/", GetStatus)
	router.Get("/ready", GetStatus)
	router.Get("/status", GetStatus)

	for _, subdomain := range subdomains {
		router.Mount(subdomain.Domain, subdomain.Router)
	}

	return &Handler{
		Router: router,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: router,
		},
	}
}

func NewEmpty(port, tracerEnable bool, metricsEnable bool) *Handler {
	router := chi.NewRouter()

	router.Use(chimiddleware.StripSlashes)

	if tracerEnable {
		router.Use(tracing.TracingMiddleware)
	}

	if metricsEnable {
		router.Use(metrics.MetricsMiddleware)
	}

	router.Use(chimiddleware.RealIP)
	router.NotFoundHandler()
	router.MethodNotAllowedHandler()

	router.Get("/", GetStatus)
	router.Get("/ready", GetStatus)
	router.Get("/status", GetStatus)

	return &Handler{
		Router: router,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: router,
		},
	}
}

func (h *Handler) Start() error {
	err := h.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (h *Handler) GracefulShutdown(ctx context.Context, gracefulTimeout int) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	ctx, shutdown := context.WithTimeout(ctx, time.Duration(gracefulTimeout)*time.Second)
	defer shutdown()

	err := h.server.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Router.ServeHTTP(w, r)
}

func GetStatus(w http.ResponseWriter, r *http.Request) {
	err := utils.WriteBody(w, http.StatusOK, map[string]string{"message": "OK"})
	if err != nil {
		return
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", CorsAllowOrigins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, User-Address, Token")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		next.ServeHTTP(w, r)
	})
}

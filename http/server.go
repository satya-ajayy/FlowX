package http

import (
	// Go Internal Packages
	"context"
	"net/http"
	"time"

	// Local Packages
	errors "flowx/errors"
	handlers "flowx/http/handlers"
	middlewares "flowx/http/middlewares"
	resp "flowx/http/response"

	// External Packages
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Server is the HTTP server that wires middleware, routes, and handlers.
type Server struct {
	close  func()
	prefix string
	logger *zap.Logger
	health *handlers.HealthCheckHandler
	run    *handlers.RunHandler
}

// NewServer creates a Server with all handler dependencies.
func NewServer(
	logger *zap.Logger,
	prefix string,
	health *handlers.HealthCheckHandler,
	run *handlers.RunHandler,
	close func(),
) *Server {
	return &Server{
		close:  close,
		logger: logger,
		prefix: prefix,
		health: health,
		run:    run,
	}
}

// Listen starts the HTTP server and blocks until shutdown.
func (s *Server) Listen(ctx context.Context, addr string) error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewares.HTTPMiddleware(s.logger))
	r.Use(middleware.Recoverer)

	r.Route(s.prefix, func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Get("/health", s.ToHTTPHandlerFunc(s.health.HealthCheck))
			r.Post("/runs", s.ToHTTPHandlerFunc(s.run.Create))
		})
	})

	errch := make(chan error)
	server := &http.Server{Addr: addr, Handler: r}
	go func() {
		s.logger.Info("Starting HTTP Server 🚀", zap.String("addr", addr))
		errch <- server.ListenAndServe()
	}()

	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if s.close != nil {
			s.close()
		}
		return nil
	}
}

// ToHTTPHandlerFunc converts a handler function to an http.HandlerFunc.
// This wrapper handles error classification and JSON response writing.
func (s *Server) ToHTTPHandlerFunc(handler func(w http.ResponseWriter, r *http.Request) (any, int, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response, status, err := handler(w, r)
		if err != nil {
			var typedErr *errors.Error
			switch {
			case errors.As(err, &typedErr):
				resp.RespondError(w, typedErr)
			default:
				s.logger.Error("internal error", zap.Error(err))
				resp.RespondMessage(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		if response != nil {
			resp.RespondJSON(w, status, response)
		}
		if status >= 100 && status < 600 {
			w.WriteHeader(status)
		}
	}
}

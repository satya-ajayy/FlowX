package http

import (
	// Go Internal Packages
	"context"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"

	// Local Packages
	errors "flowx/errors"
	handlers "flowx/http/handlers"
	middlewares "flowx/http/middlewares"
	resp "flowx/http/response"
	health "flowx/services/health"

	// External Packages
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Server struct follows the alphabet order
type Server struct {
	prefix string
	logger *zap.Logger
	health *health.HealthCheckService
	square *handlers.SquareHandler
}

func NewServer(
	prefix string,
	logger *zap.Logger,
	healthCheckService *health.HealthCheckService,
	square *handlers.SquareHandler,
) *Server {
	return &Server{
		prefix: prefix,
		logger: logger,
		health: healthCheckService,
		square: square,
	}
}

func (s *Server) Listen(ctx context.Context, addr string) error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewares.HTTPMiddleware(s.logger))
	r.Use(middleware.Recoverer)

	r.Route(s.prefix, func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Get("/health", s.HealthCheckHandler)
			r.Post("/square/init", s.ToHTTPHandlerFunc(s.square.InitWorkflow))
		})
	})

	errch := make(chan error)
	server := &http.Server{Addr: addr, Handler: r}
	go func() {
		s.logger.Info("Starting server", zap.String("addr", addr))
		errch <- server.ListenAndServe()
	}()

	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

// ToHTTPHandlerFunc converts a handler function to an http.HandlerFunc.
// This wrapper function is used to handle errors and respond to the client
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

// HealthCheckHandler returns the health status of the service
func (s *Server) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if ok := s.health.Health(r.Context()); !ok {
		resp.RespondMessage(w, http.StatusServiceUnavailable, "health check failed")
		return
	}
	resp.RespondMessage(w, http.StatusOK, "!!! We are RunninGoo !!!")
}

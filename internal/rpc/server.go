package rpc

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/nickheyer/discordiance/internal/rpc/services"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	handler http.Handler
}

func NewServer() *Server {
	s := &Server{}
	s.setupHandler()
	return s
}

func (s *Server) setupHandler() {
	mux := http.NewServeMux()

	interceptors := []connect.Interceptor{
		&loggingInterceptor{},
	}

	opts := []connect.HandlerOption{
		connect.WithInterceptors(interceptors...),
	}

	// Register RPC services
	healthPath, healthHandler := discordiancev1connect.NewHealthServiceHandler(
		services.NewHealthService(), opts...)
	mux.Handle(healthPath, healthHandler)

	s.handler = h2c.NewHandler(mux, &http2.Server{})
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

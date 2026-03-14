package rpc

import (
	"net/http"

	"connectrpc.com/connect"
	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/engine"
	"github.com/nickheyer/discordiance/internal/rpc/services"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	handler http.Handler
}

func NewServer(db *gorm.DB, eng *engine.Engine) *Server {
	s := &Server{}
	s.setupHandler(db, eng)
	return s
}

func (s *Server) setupHandler(db *gorm.DB, eng *engine.Engine) {
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

	productPath, productHandler := discordiancev1connect.NewProductServiceHandler(
		services.NewProductService(db), opts...)
	mux.Handle(productPath, productHandler)

	agentPath, agentHandler := discordiancev1connect.NewAgentServiceHandler(
		services.NewAgentService(db), opts...)
	mux.Handle(agentPath, agentHandler)

	platformPath, platformHandler := discordiancev1connect.NewPlatformServiceHandler(
		services.NewPlatformService(db), opts...)
	mux.Handle(platformPath, platformHandler)

	pipelinePath, pipelineHandler := discordiancev1connect.NewPipelineServiceHandler(
		services.NewPipelineService(db, eng), opts...)
	mux.Handle(pipelinePath, pipelineHandler)

	insightPath, insightHandler := discordiancev1connect.NewInsightServiceHandler(
		services.NewInsightService(db), opts...)
	mux.Handle(insightPath, insightHandler)

	reporterPath, reporterHandler := discordiancev1connect.NewReporterServiceHandler(
		services.NewReporterService(db), opts...)
	mux.Handle(reporterPath, reporterHandler)

	reportPath, reportHandler := discordiancev1connect.NewReportServiceHandler(
		services.NewReportService(db), opts...)
	mux.Handle(reportPath, reportHandler)

	s.handler = h2c.NewHandler(mux, &http2.Server{})
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

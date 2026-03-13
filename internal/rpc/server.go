package rpc

import (
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/nickheyer/discordiance/internal/engine"
	"github.com/nickheyer/discordiance/internal/rpc/services"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	web "github.com/nickheyer/discordiance/web/discordiance"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gorm.io/gorm"
)

type Server struct {
	db      *gorm.DB
	engine  *engine.Engine
	handler http.Handler
}

func NewServer(db *gorm.DB, eng *engine.Engine) *Server {
	s := &Server{
		db:     db,
		engine: eng,
	}
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
		services.NewHealthService(s.engine), opts...)
	mux.Handle(healthPath, healthHandler)

	productPath, productHandler := discordiancev1connect.NewProductServiceHandler(
		services.NewProductService(s.db), opts...)
	mux.Handle(productPath, productHandler)

	platformPath, platformHandler := discordiancev1connect.NewPlatformServiceHandler(
		services.NewPlatformService(s.db), opts...)
	mux.Handle(platformPath, platformHandler)

	agentPath, agentHandler := discordiancev1connect.NewAgentServiceHandler(
		services.NewAgentService(s.db), opts...)
	mux.Handle(agentPath, agentHandler)

	reporterPath, reporterHandler := discordiancev1connect.NewReporterServiceHandler(
		services.NewReporterService(s.db), opts...)
	mux.Handle(reporterPath, reporterHandler)

	pipelinePath, pipelineHandler := discordiancev1connect.NewPipelineServiceHandler(
		services.NewPipelineService(s.db, s.engine), opts...)
	mux.Handle(pipelinePath, pipelineHandler)

	insightPath, insightHandler := discordiancev1connect.NewInsightServiceHandler(
		services.NewInsightService(s.db), opts...)
	mux.Handle(insightPath, insightHandler)

	messagePath, messageHandler := discordiancev1connect.NewMessageServiceHandler(
		services.NewMessageService(s.db), opts...)
	mux.Handle(messagePath, messageHandler)

	productFilePath, productFileHandler := discordiancev1connect.NewProductFileServiceHandler(
		services.NewProductFileService(s.db), opts...)
	mux.Handle(productFilePath, productFileHandler)

	// Serve frontend for non-RPC routes
	s.setupFrontend(mux)

	s.handler = h2c.NewHandler(mux, &http2.Server{})
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) setupFrontend(mux *http.ServeMux) {
	fs := s.getFrontendFS()
	if fs == nil {
		return
	}
	mux.Handle("/", s.createFrontendHandler(fs))
}

func (s *Server) getFrontendFS() http.FileSystem {
	if buildFS, err := web.BuildFS(); err == nil {
		return http.FS(buildFS)
	}
	return nil
}

func (s *Server) createFrontendHandler(fs http.FileSystem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isConnectPath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		file, err := fs.Open(path)
		if err == nil {
			defer file.Close()
			stat, _ := file.Stat()
			http.ServeContent(w, r, path, stat.ModTime(), file)
			return
		}

		indexFile, err := fs.Open("/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer indexFile.Close()

		stat, _ := indexFile.Stat()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "/index.html", stat.ModTime(), indexFile)
	}
}

func isConnectPath(path string) bool {
	connectPrefixes := []string{
		"/discordiance.v1.",
		"/grpc.reflection.",
		"/connect.",
	}
	for _, prefix := range connectPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

package services

import (
	"context"
	"log/slog"
	"os"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ discordiancev1connect.HealthServiceHandler = (*HealthService)(nil)

type HealthService struct {
}

func NewHealthService() *HealthService {
	return &HealthService{}
}

func (s *HealthService) HealthCheck(ctx context.Context, req *connect.Request[v1.HealthCheckRequest]) (*connect.Response[v1.HealthCheckResponse], error) {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		version = "dev"
	}

	status := "ok"

	slog.Debug("rpc: HealthCheck", "status", status, "version", version)

	return connect.NewResponse(&v1.HealthCheckResponse{
		Status:    status,
		Timestamp: timestamppb.New(time.Now()),
		Version:   version,
	}), nil
}

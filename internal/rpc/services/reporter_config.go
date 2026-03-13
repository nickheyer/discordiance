package services

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/reporter"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"gorm.io/gorm"
)

var _ discordiancev1connect.ReporterConfigServiceHandler = (*ReporterConfigService)(nil)

type ReporterConfigService struct {
	db *gorm.DB
}

func NewReporterConfigService(db *gorm.DB) *ReporterConfigService {
	return &ReporterConfigService{db: db}
}

func (s *ReporterConfigService) CreateReporterConfig(ctx context.Context, req *connect.Request[v1.CreateReporterConfigRequest]) (*connect.Response[v1.CreateReporterConfigResponse], error) {
	if req.Msg.ProductId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("product_id is required"))
	}
	if req.Msg.Type == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("type is required"))
	}

	slog.Info("rpc: CreateReporterConfig", "product_id", req.Msg.ProductId, "type", req.Msg.Type)

	cfg := models.ReporterConfig{
		ProductID: uint(req.Msg.ProductId),
		Type:      req.Msg.Type,
		Enabled:   req.Msg.Enabled,
		Settings:  models.JSONMap(req.Msg.Settings),
	}
	if err := s.db.Create(&cfg).Error; err != nil {
		slog.Error("rpc: CreateReporterConfig failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating reporter config: %w", err))
	}

	slog.Info("rpc: CreateReporterConfig success", "id", cfg.ID, "type", cfg.Type)
	return connect.NewResponse(&v1.CreateReporterConfigResponse{
		ReporterConfig: reporterConfigToProto(cfg),
	}), nil
}

func (s *ReporterConfigService) UpdateReporterConfig(ctx context.Context, req *connect.Request[v1.UpdateReporterConfigRequest]) (*connect.Response[v1.UpdateReporterConfigResponse], error) {
	slog.Info("rpc: UpdateReporterConfig", "id", req.Msg.Id, "type", req.Msg.Type)

	var cfg models.ReporterConfig
	if err := s.db.First(&cfg, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: UpdateReporterConfig not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("reporter config %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cfg.Type = req.Msg.Type
	cfg.Enabled = req.Msg.Enabled
	cfg.Settings = models.JSONMap(req.Msg.Settings)
	if err := s.db.Save(&cfg).Error; err != nil {
		slog.Error("rpc: UpdateReporterConfig failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating reporter config: %w", err))
	}

	slog.Info("rpc: UpdateReporterConfig success", "id", cfg.ID, "type", cfg.Type)
	return connect.NewResponse(&v1.UpdateReporterConfigResponse{
		ReporterConfig: reporterConfigToProto(cfg),
	}), nil
}

func (s *ReporterConfigService) DeleteReporterConfig(ctx context.Context, req *connect.Request[v1.DeleteReporterConfigRequest]) (*connect.Response[v1.DeleteReporterConfigResponse], error) {
	slog.Info("rpc: DeleteReporterConfig", "id", req.Msg.Id)

	if err := s.db.Delete(&models.ReporterConfig{}, req.Msg.Id).Error; err != nil {
		slog.Error("rpc: DeleteReporterConfig failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("deleting reporter config: %w", err))
	}

	slog.Info("rpc: DeleteReporterConfig success", "id", req.Msg.Id)
	return connect.NewResponse(&v1.DeleteReporterConfigResponse{}), nil
}

func (s *ReporterConfigService) ListAvailableReporters(ctx context.Context, req *connect.Request[v1.ListAvailableReportersRequest]) (*connect.Response[v1.ListAvailableReportersResponse], error) {
	reporters := reporter.Available()
	slog.Info("rpc: ListAvailableReporters", "count", len(reporters))
	return connect.NewResponse(&v1.ListAvailableReportersResponse{
		Reporters: reporters,
	}), nil
}

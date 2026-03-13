package services

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/platform"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"gorm.io/gorm"
)

var _ discordiancev1connect.PlatformConfigServiceHandler = (*PlatformConfigService)(nil)

type PlatformConfigService struct {
	db *gorm.DB
}

func NewPlatformConfigService(db *gorm.DB) *PlatformConfigService {
	return &PlatformConfigService{db: db}
}

func (s *PlatformConfigService) CreatePlatformConfig(ctx context.Context, req *connect.Request[v1.CreatePlatformConfigRequest]) (*connect.Response[v1.CreatePlatformConfigResponse], error) {
	if req.Msg.ProductId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("product_id is required"))
	}
	if req.Msg.Type == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("type is required"))
	}

	slog.Info("rpc: CreatePlatformConfig", "product_id", req.Msg.ProductId, "type", req.Msg.Type)

	cfg := models.PlatformConfig{
		ProductID: uint(req.Msg.ProductId),
		Type:      req.Msg.Type,
		Enabled:   req.Msg.Enabled,
		Settings:  models.JSONMap(req.Msg.Settings),
	}
	if err := s.db.Create(&cfg).Error; err != nil {
		slog.Error("rpc: CreatePlatformConfig failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating platform config: %w", err))
	}

	slog.Info("rpc: CreatePlatformConfig success", "id", cfg.ID, "type", cfg.Type)
	return connect.NewResponse(&v1.CreatePlatformConfigResponse{
		PlatformConfig: platformConfigToProto(cfg),
	}), nil
}

func (s *PlatformConfigService) UpdatePlatformConfig(ctx context.Context, req *connect.Request[v1.UpdatePlatformConfigRequest]) (*connect.Response[v1.UpdatePlatformConfigResponse], error) {
	slog.Info("rpc: UpdatePlatformConfig", "id", req.Msg.Id, "type", req.Msg.Type)

	var cfg models.PlatformConfig
	if err := s.db.First(&cfg, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: UpdatePlatformConfig not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("platform config %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cfg.Type = req.Msg.Type
	cfg.Enabled = req.Msg.Enabled
	cfg.Settings = models.JSONMap(req.Msg.Settings)
	if err := s.db.Save(&cfg).Error; err != nil {
		slog.Error("rpc: UpdatePlatformConfig failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating platform config: %w", err))
	}

	slog.Info("rpc: UpdatePlatformConfig success", "id", cfg.ID, "type", cfg.Type)
	return connect.NewResponse(&v1.UpdatePlatformConfigResponse{
		PlatformConfig: platformConfigToProto(cfg),
	}), nil
}

func (s *PlatformConfigService) DeletePlatformConfig(ctx context.Context, req *connect.Request[v1.DeletePlatformConfigRequest]) (*connect.Response[v1.DeletePlatformConfigResponse], error) {
	slog.Info("rpc: DeletePlatformConfig", "id", req.Msg.Id)

	if err := s.db.Delete(&models.PlatformConfig{}, req.Msg.Id).Error; err != nil {
		slog.Error("rpc: DeletePlatformConfig failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("deleting platform config: %w", err))
	}

	slog.Info("rpc: DeletePlatformConfig success", "id", req.Msg.Id)
	return connect.NewResponse(&v1.DeletePlatformConfigResponse{}), nil
}

func (s *PlatformConfigService) ListAvailablePlatforms(ctx context.Context, req *connect.Request[v1.ListAvailablePlatformsRequest]) (*connect.Response[v1.ListAvailablePlatformsResponse], error) {
	platforms := platform.Available()
	slog.Info("rpc: ListAvailablePlatforms", "count", len(platforms))
	return connect.NewResponse(&v1.ListAvailablePlatformsResponse{
		Platforms: platforms,
	}), nil
}

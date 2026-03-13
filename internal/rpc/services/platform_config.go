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
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

var _ discordiancev1connect.PlatformServiceHandler = (*PlatformService)(nil)

type PlatformService struct {
	db *gorm.DB
}

func NewPlatformService(db *gorm.DB) *PlatformService {
	return &PlatformService{db: db}
}

func (s *PlatformService) ListPlatforms(ctx context.Context, req *connect.Request[v1.ListPlatformsRequest]) (*connect.Response[v1.ListPlatformsResponse], error) {
	var platforms []models.Platform
	if err := s.db.Find(&platforms).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("listing platforms: %w", err))
	}
	slog.Info("rpc: ListPlatforms", "count", len(platforms))
	resp := &v1.ListPlatformsResponse{
		Platforms: make([]*v1.Platform, len(platforms)),
	}
	for i, c := range platforms {
		resp.Platforms[i] = platformToProto(c)
	}
	return connect.NewResponse(resp), nil
}

func (s *PlatformService) GetPlatform(ctx context.Context, req *connect.Request[v1.GetPlatformRequest]) (*connect.Response[v1.GetPlatformResponse], error) {
	var cfg models.Platform
	if err := s.db.First(&cfg, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("platform %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetPlatformResponse{
		Platform: platformToProto(cfg),
	}), nil
}

func (s *PlatformService) CreatePlatform(ctx context.Context, req *connect.Request[v1.CreatePlatformRequest]) (*connect.Response[v1.CreatePlatformResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	if req.Msg.Type == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("type is required"))
	}

	slog.Info("rpc: CreatePlatform", "name", req.Msg.Name, "type", req.Msg.Type)

	cfg := models.Platform{
		Name:    req.Msg.Name,
		Type:    req.Msg.Type,
		Enabled: req.Msg.Enabled,
	}
	applyPlatformSettings(&cfg, req.Msg.GetDiscordSettings())

	if err := s.db.Create(&cfg).Error; err != nil {
		slog.Error("rpc: CreatePlatform failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating platform: %w", err))
	}

	slog.Info("rpc: CreatePlatform success", "id", cfg.ID, "type", cfg.Type)
	return connect.NewResponse(&v1.CreatePlatformResponse{
		Platform: platformToProto(cfg),
	}), nil
}

func (s *PlatformService) UpdatePlatform(ctx context.Context, req *connect.Request[v1.UpdatePlatformRequest]) (*connect.Response[v1.UpdatePlatformResponse], error) {
	slog.Info("rpc: UpdatePlatform", "id", req.Msg.Id, "type", req.Msg.Type)

	var cfg models.Platform
	if err := s.db.First(&cfg, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: UpdatePlatform not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("platform %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cfg.Name = req.Msg.Name
	cfg.Type = req.Msg.Type
	cfg.Enabled = req.Msg.Enabled
	applyPlatformSettings(&cfg, req.Msg.GetDiscordSettings())

	if err := s.db.Save(&cfg).Error; err != nil {
		slog.Error("rpc: UpdatePlatform failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating platform: %w", err))
	}

	slog.Info("rpc: UpdatePlatform success", "id", cfg.ID, "type", cfg.Type)
	return connect.NewResponse(&v1.UpdatePlatformResponse{
		Platform: platformToProto(cfg),
	}), nil
}

func (s *PlatformService) DeletePlatform(ctx context.Context, req *connect.Request[v1.DeletePlatformRequest]) (*connect.Response[v1.DeletePlatformResponse], error) {
	slog.Info("rpc: DeletePlatform", "id", req.Msg.Id)

	if err := s.db.Delete(&models.Platform{}, req.Msg.Id).Error; err != nil {
		slog.Error("rpc: DeletePlatform failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("deleting platform: %w", err))
	}

	slog.Info("rpc: DeletePlatform success", "id", req.Msg.Id)
	return connect.NewResponse(&v1.DeletePlatformResponse{}), nil
}

func (s *PlatformService) ListAvailablePlatforms(ctx context.Context, req *connect.Request[v1.ListAvailablePlatformsRequest]) (*connect.Response[v1.ListAvailablePlatformsResponse], error) {
	platforms := platform.Available()
	slog.Info("rpc: ListAvailablePlatforms", "count", len(platforms))
	return connect.NewResponse(&v1.ListAvailablePlatformsResponse{
		Platforms: platforms,
	}), nil
}

func applyPlatformSettings(cfg *models.Platform, ds *v1.DiscordPlatformSettings) {
	if ds != nil {
		cfg.Token = ds.Token
		cfg.ChannelIds = ds.ChannelIds
	}
}

func platformToProto(pc models.Platform) *v1.Platform {
	out := &v1.Platform{
		Id:        uint64(pc.ID),
		CreatedAt: timestamppb.New(pc.CreatedAt),
		UpdatedAt: timestamppb.New(pc.UpdatedAt),
		Name:      pc.Name,
		Type:      pc.Type,
		Enabled:   pc.Enabled,
	}
	switch pc.Type {
	case "discord":
		out.Settings = &v1.Platform_DiscordSettings{
			DiscordSettings: &v1.DiscordPlatformSettings{
				Token:      pc.Token,
				ChannelIds: pc.ChannelIds,
			},
		}
	}
	return out
}

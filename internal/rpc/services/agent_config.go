package services

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"gorm.io/gorm"
)

var _ discordiancev1connect.AgentConfigServiceHandler = (*AgentConfigService)(nil)

type AgentConfigService struct {
	db *gorm.DB
}

func NewAgentConfigService(db *gorm.DB) *AgentConfigService {
	return &AgentConfigService{db: db}
}

func (s *AgentConfigService) UpsertAgentConfig(ctx context.Context, req *connect.Request[v1.UpsertAgentConfigRequest]) (*connect.Response[v1.UpsertAgentConfigResponse], error) {
	if req.Msg.ProductId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("product_id is required"))
	}

	slog.Info("rpc: UpsertAgentConfig", "product_id", req.Msg.ProductId, "model", req.Msg.Model)

	var cfg models.AgentConfig
	err := s.db.Where("product_id = ?", req.Msg.ProductId).First(&cfg).Error

	if err == gorm.ErrRecordNotFound {
		slog.Info("rpc: UpsertAgentConfig creating new config", "product_id", req.Msg.ProductId)
		cfg = models.AgentConfig{
			ProductID:    uint(req.Msg.ProductId),
			BaseURL:      req.Msg.BaseUrl,
			APIKey:       req.Msg.ApiKey,
			OrgID:        req.Msg.OrgId,
			Model:        req.Msg.Model,
			SystemPrompt: req.Msg.SystemPrompt,
			BatchSize:    int(req.Msg.BatchSize),
			BatchTimeout: int(req.Msg.BatchTimeout),
		}
		if err := s.db.Create(&cfg).Error; err != nil {
			slog.Error("rpc: UpsertAgentConfig create failed", "product_id", req.Msg.ProductId, "error", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating agent config: %w", err))
		}
		slog.Info("rpc: UpsertAgentConfig created", "id", cfg.ID, "product_id", cfg.ProductID)
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	} else {
		slog.Info("rpc: UpsertAgentConfig updating existing config", "id", cfg.ID, "product_id", req.Msg.ProductId)
		cfg.BaseURL = req.Msg.BaseUrl
		cfg.APIKey = req.Msg.ApiKey
		cfg.OrgID = req.Msg.OrgId
		cfg.Model = req.Msg.Model
		cfg.SystemPrompt = req.Msg.SystemPrompt
		cfg.BatchSize = int(req.Msg.BatchSize)
		cfg.BatchTimeout = int(req.Msg.BatchTimeout)
		if err := s.db.Save(&cfg).Error; err != nil {
			slog.Error("rpc: UpsertAgentConfig update failed", "id", cfg.ID, "error", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating agent config: %w", err))
		}
		slog.Info("rpc: UpsertAgentConfig updated", "id", cfg.ID)
	}

	return connect.NewResponse(&v1.UpsertAgentConfigResponse{
		AgentConfig: agentConfigToProto(cfg),
	}), nil
}

package services

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

var _ discordiancev1connect.AgentServiceHandler = (*AgentService)(nil)

type AgentService struct {
	db *gorm.DB
}

func NewAgentService(db *gorm.DB) *AgentService {
	return &AgentService{db: db}
}

func (s *AgentService) ListAgents(ctx context.Context, req *connect.Request[v1.ListAgentsRequest]) (*connect.Response[v1.ListAgentsResponse], error) {
	var agents []models.Agent
	if err := s.db.Find(&agents).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("listing agents: %w", err))
	}
	slog.Info("rpc: ListAgents", "count", len(agents))
	resp := &v1.ListAgentsResponse{
		Agents: make([]*v1.Agent, len(agents)),
	}
	for i, c := range agents {
		resp.Agents[i] = agentToProto(c)
	}
	return connect.NewResponse(resp), nil
}

func (s *AgentService) GetAgent(ctx context.Context, req *connect.Request[v1.GetAgentRequest]) (*connect.Response[v1.GetAgentResponse], error) {
	var cfg models.Agent
	if err := s.db.First(&cfg, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetAgentResponse{
		Agent: agentToProto(cfg),
	}), nil
}

func (s *AgentService) CreateAgent(ctx context.Context, req *connect.Request[v1.CreateAgentRequest]) (*connect.Response[v1.CreateAgentResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	slog.Info("rpc: CreateAgent", "name", req.Msg.Name, "model", req.Msg.Model)

	cfg := models.Agent{
		Name:         req.Msg.Name,
		BaseURL:      req.Msg.BaseUrl,
		APIKey:       req.Msg.ApiKey,
		OrgID:        req.Msg.OrgId,
		Model:        req.Msg.Model,
		SystemPrompt: req.Msg.SystemPrompt,
		BatchSize:    int(req.Msg.BatchSize),
		BatchTimeout: int(req.Msg.BatchTimeout),
	}
	if err := s.db.Create(&cfg).Error; err != nil {
		slog.Error("rpc: CreateAgent failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating agent: %w", err))
	}

	slog.Info("rpc: CreateAgent success", "id", cfg.ID)
	return connect.NewResponse(&v1.CreateAgentResponse{
		Agent: agentToProto(cfg),
	}), nil
}

func (s *AgentService) UpdateAgent(ctx context.Context, req *connect.Request[v1.UpdateAgentRequest]) (*connect.Response[v1.UpdateAgentResponse], error) {
	slog.Info("rpc: UpdateAgent", "id", req.Msg.Id)

	var cfg models.Agent
	if err := s.db.First(&cfg, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cfg.Name = req.Msg.Name
	cfg.BaseURL = req.Msg.BaseUrl
	cfg.APIKey = req.Msg.ApiKey
	cfg.OrgID = req.Msg.OrgId
	cfg.Model = req.Msg.Model
	cfg.SystemPrompt = req.Msg.SystemPrompt
	cfg.BatchSize = int(req.Msg.BatchSize)
	cfg.BatchTimeout = int(req.Msg.BatchTimeout)
	if err := s.db.Save(&cfg).Error; err != nil {
		slog.Error("rpc: UpdateAgent failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating agent: %w", err))
	}

	slog.Info("rpc: UpdateAgent success", "id", cfg.ID)
	return connect.NewResponse(&v1.UpdateAgentResponse{
		Agent: agentToProto(cfg),
	}), nil
}

func (s *AgentService) DeleteAgent(ctx context.Context, req *connect.Request[v1.DeleteAgentRequest]) (*connect.Response[v1.DeleteAgentResponse], error) {
	slog.Info("rpc: DeleteAgent", "id", req.Msg.Id)

	if err := s.db.Delete(&models.Agent{}, req.Msg.Id).Error; err != nil {
		slog.Error("rpc: DeleteAgent failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("deleting agent: %w", err))
	}

	slog.Info("rpc: DeleteAgent success", "id", req.Msg.Id)
	return connect.NewResponse(&v1.DeleteAgentResponse{}), nil
}

func agentToProto(ac models.Agent) *v1.Agent {
	return &v1.Agent{
		Id:           uint64(ac.ID),
		CreatedAt:    timestamppb.New(ac.CreatedAt),
		UpdatedAt:    timestamppb.New(ac.UpdatedAt),
		Name:         ac.Name,
		BaseUrl:      ac.BaseURL,
		ApiKey:       ac.APIKey,
		OrgId:        ac.OrgID,
		Model:        ac.Model,
		SystemPrompt: ac.SystemPrompt,
		BatchSize:    int32(ac.BatchSize),
		BatchTimeout: int32(ac.BatchTimeout),
	}
}

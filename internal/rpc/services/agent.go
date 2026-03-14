package services

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AgentService struct {
	discordiancev1connect.UnimplementedAgentServiceHandler
	db *gorm.DB
}

func NewAgentService(db *gorm.DB) *AgentService {
	return &AgentService{db: db}
}

func (s *AgentService) CreateAgent(_ context.Context, req *connect.Request[v1.CreateAgentRequest]) (*connect.Response[v1.CreateAgentResponse], error) {
	agent := models.Agent{
		ID:          uuid.NewString(),
		Name:        req.Msg.Name,
		BaseURL:     req.Msg.BaseUrl,
		Model:       req.Msg.Model,
		APIKey:      req.Msg.ApiKey,
		MaxTokens:   req.Msg.MaxTokens,
		Temperature: req.Msg.Temperature,
	}

	if err := s.db.Create(&agent).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.CreateAgentResponse{
		Agent: agentToProto(&agent),
	}), nil
}

func (s *AgentService) GetAgent(_ context.Context, req *connect.Request[v1.GetAgentRequest]) (*connect.Response[v1.GetAgentResponse], error) {
	var agent models.Agent
	if err := s.db.First(&agent, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&v1.GetAgentResponse{
		Agent: agentToProto(&agent),
	}), nil
}

func (s *AgentService) ListAgents(_ context.Context, req *connect.Request[v1.ListAgentsRequest]) (*connect.Response[v1.ListAgentsResponse], error) {
	pageSize, offset := parsePagination(req.Msg.Pagination)

	var agents []models.Agent
	var total int64
	s.db.Model(&models.Agent{}).Count(&total)
	s.db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&agents)

	protos := make([]*v1.Agent, len(agents))
	for i := range agents {
		protos[i] = agentToProto(&agents[i])
	}

	return connect.NewResponse(&v1.ListAgentsResponse{
		Agents:     protos,
		Pagination: buildPaginationResponse(offset, pageSize, int(total)),
	}), nil
}

func (s *AgentService) UpdateAgent(_ context.Context, req *connect.Request[v1.UpdateAgentRequest]) (*connect.Response[v1.UpdateAgentResponse], error) {
	var agent models.Agent
	if err := s.db.First(&agent, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	agent.Name = req.Msg.Name
	agent.BaseURL = req.Msg.BaseUrl
	agent.Model = req.Msg.Model
	if req.Msg.ApiKey != "" {
		agent.APIKey = req.Msg.ApiKey
	}
	agent.MaxTokens = req.Msg.MaxTokens
	agent.Temperature = req.Msg.Temperature

	if err := s.db.Save(&agent).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.UpdateAgentResponse{
		Agent: agentToProto(&agent),
	}), nil
}

func (s *AgentService) DeleteAgent(_ context.Context, req *connect.Request[v1.DeleteAgentRequest]) (*connect.Response[v1.DeleteAgentResponse], error) {
	if err := s.db.Delete(&models.Agent{}, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteAgentResponse{}), nil
}

func (s *AgentService) TestAgentConnection(_ context.Context, req *connect.Request[v1.TestAgentConnectionRequest]) (*connect.Response[v1.TestAgentConnectionResponse], error) {
	var agent models.Agent
	if err := s.db.First(&agent, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// TODO: implement actual connection test via OpenAI-compatible models endpoint
	return connect.NewResponse(&v1.TestAgentConnectionResponse{
		Success: true,
		Message: "connection test not yet implemented",
	}), nil
}

func agentToProto(a *models.Agent) *v1.Agent {
	return &v1.Agent{
		Id:          a.ID,
		Name:        a.Name,
		BaseUrl:     a.BaseURL,
		Model:       a.Model,
		ApiKey:      redactKey(a.APIKey),
		MaxTokens:   a.MaxTokens,
		Temperature: a.Temperature,
		CreatedAt:   timestamppb.New(a.CreatedAt),
		UpdatedAt:   timestamppb.New(a.UpdatedAt),
	}
}

func redactKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

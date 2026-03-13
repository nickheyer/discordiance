package services

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/nickheyer/discordiance/internal/engine"
	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

var _ discordiancev1connect.PipelineServiceHandler = (*PipelineService)(nil)

type PipelineService struct {
	db     *gorm.DB
	engine *engine.Engine
}

func NewPipelineService(db *gorm.DB, eng *engine.Engine) *PipelineService {
	return &PipelineService{db: db, engine: eng}
}

func (s *PipelineService) ListPipelines(ctx context.Context, req *connect.Request[v1.ListPipelinesRequest]) (*connect.Response[v1.ListPipelinesResponse], error) {
	var pipelines []models.Pipeline
	if err := s.db.Preload("Product").Preload("Platform").Preload("Agent").Preload("Reporters").
		Find(&pipelines).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("listing pipelines: %w", err))
	}
	slog.Info("rpc: ListPipelines", "count", len(pipelines))
	resp := &v1.ListPipelinesResponse{
		Pipelines: make([]*v1.Pipeline, len(pipelines)),
	}
	for i, p := range pipelines {
		resp.Pipelines[i] = pipelineToProto(p)
	}
	return connect.NewResponse(resp), nil
}

func (s *PipelineService) GetPipeline(ctx context.Context, req *connect.Request[v1.GetPipelineRequest]) (*connect.Response[v1.GetPipelineResponse], error) {
	var pipeline models.Pipeline
	if err := s.db.Preload("Product").Preload("Platform").Preload("Agent").Preload("Reporters").
		First(&pipeline, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("pipeline %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetPipelineResponse{
		Pipeline: pipelineToProto(pipeline),
	}), nil
}

func (s *PipelineService) CreatePipeline(ctx context.Context, req *connect.Request[v1.CreatePipelineRequest]) (*connect.Response[v1.CreatePipelineResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	slog.Info("rpc: CreatePipeline", "name", req.Msg.Name)

	pipeline := models.Pipeline{
		Name:       req.Msg.Name,
		Enabled:    req.Msg.Enabled,
		ProductID:  uint(req.Msg.ProductId),
		PlatformID: uint(req.Msg.PlatformId),
		AgentID:    uint(req.Msg.AgentId),
	}

	if err := s.db.Create(&pipeline).Error; err != nil {
		slog.Error("rpc: CreatePipeline failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating pipeline: %w", err))
	}

	// Associate reporters
	if len(req.Msg.ReporterIds) > 0 {
		var reporters []models.Reporter
		for _, id := range req.Msg.ReporterIds {
			reporters = append(reporters, models.Reporter{ID: uint(id)})
		}
		if err := s.db.Model(&pipeline).Association("Reporters").Replace(reporters); err != nil {
			slog.Error("rpc: CreatePipeline reporter association failed", "error", err)
		}
	}

	// Reload with associations
	s.db.Preload("Product").Preload("Platform").Preload("Agent").Preload("Reporters").
		First(&pipeline, pipeline.ID)

	slog.Info("rpc: CreatePipeline success", "id", pipeline.ID)
	return connect.NewResponse(&v1.CreatePipelineResponse{
		Pipeline: pipelineToProto(pipeline),
	}), nil
}

func (s *PipelineService) UpdatePipeline(ctx context.Context, req *connect.Request[v1.UpdatePipelineRequest]) (*connect.Response[v1.UpdatePipelineResponse], error) {
	slog.Info("rpc: UpdatePipeline", "id", req.Msg.Id)

	var pipeline models.Pipeline
	if err := s.db.First(&pipeline, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("pipeline %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pipeline.Name = req.Msg.Name
	pipeline.Enabled = req.Msg.Enabled
	pipeline.ProductID = uint(req.Msg.ProductId)
	pipeline.PlatformID = uint(req.Msg.PlatformId)
	pipeline.AgentID = uint(req.Msg.AgentId)

	if err := s.db.Save(&pipeline).Error; err != nil {
		slog.Error("rpc: UpdatePipeline failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating pipeline: %w", err))
	}

	// Update reporter associations
	var reporters []models.Reporter
	for _, id := range req.Msg.ReporterIds {
		reporters = append(reporters, models.Reporter{ID: uint(id)})
	}
	if err := s.db.Model(&pipeline).Association("Reporters").Replace(reporters); err != nil {
		slog.Error("rpc: UpdatePipeline reporter association failed", "error", err)
	}

	// Reload
	s.db.Preload("Product").Preload("Platform").Preload("Agent").Preload("Reporters").
		First(&pipeline, pipeline.ID)

	slog.Info("rpc: UpdatePipeline success", "id", pipeline.ID)
	return connect.NewResponse(&v1.UpdatePipelineResponse{
		Pipeline: pipelineToProto(pipeline),
	}), nil
}

func (s *PipelineService) DeletePipeline(ctx context.Context, req *connect.Request[v1.DeletePipelineRequest]) (*connect.Response[v1.DeletePipelineResponse], error) {
	slog.Info("rpc: DeletePipeline", "id", req.Msg.Id)

	// Stop pipeline if running
	_ = s.engine.StopPipeline(ctx, uint(req.Msg.Id))

	// Clear associations first
	var pipeline models.Pipeline
	if err := s.db.First(&pipeline, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("pipeline %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.db.Model(&pipeline).Association("Reporters").Clear()
	s.db.Delete(&pipeline)

	slog.Info("rpc: DeletePipeline success", "id", req.Msg.Id)
	return connect.NewResponse(&v1.DeletePipelineResponse{}), nil
}

func (s *PipelineService) StartPipeline(ctx context.Context, req *connect.Request[v1.StartPipelineRequest]) (*connect.Response[v1.StartPipelineResponse], error) {
	if req.Msg.PipelineId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("pipeline_id is required"))
	}

	slog.Info("rpc: StartPipeline", "pipeline_id", req.Msg.PipelineId)
	if err := s.engine.StartPipeline(ctx, uint(req.Msg.PipelineId)); err != nil {
		slog.Error("rpc: StartPipeline failed", "pipeline_id", req.Msg.PipelineId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("starting pipeline: %w", err))
	}

	slog.Info("rpc: StartPipeline success", "pipeline_id", req.Msg.PipelineId)
	return connect.NewResponse(&v1.StartPipelineResponse{}), nil
}

func (s *PipelineService) StopPipeline(ctx context.Context, req *connect.Request[v1.StopPipelineRequest]) (*connect.Response[v1.StopPipelineResponse], error) {
	if req.Msg.PipelineId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("pipeline_id is required"))
	}

	slog.Info("rpc: StopPipeline", "pipeline_id", req.Msg.PipelineId)
	if err := s.engine.StopPipeline(ctx, uint(req.Msg.PipelineId)); err != nil {
		slog.Error("rpc: StopPipeline failed", "pipeline_id", req.Msg.PipelineId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("stopping pipeline: %w", err))
	}

	slog.Info("rpc: StopPipeline success", "pipeline_id", req.Msg.PipelineId)
	return connect.NewResponse(&v1.StopPipelineResponse{}), nil
}

func (s *PipelineService) RestartPipeline(ctx context.Context, req *connect.Request[v1.RestartPipelineRequest]) (*connect.Response[v1.RestartPipelineResponse], error) {
	if req.Msg.PipelineId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("pipeline_id is required"))
	}

	slog.Info("rpc: RestartPipeline", "pipeline_id", req.Msg.PipelineId)
	if err := s.engine.RestartPipeline(ctx, uint(req.Msg.PipelineId)); err != nil {
		slog.Error("rpc: RestartPipeline failed", "pipeline_id", req.Msg.PipelineId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("restarting pipeline: %w", err))
	}

	slog.Info("rpc: RestartPipeline success", "pipeline_id", req.Msg.PipelineId)
	return connect.NewResponse(&v1.RestartPipelineResponse{}), nil
}

func (s *PipelineService) BackfillPipeline(ctx context.Context, req *connect.Request[v1.BackfillPipelineRequest]) (*connect.Response[v1.BackfillPipelineResponse], error) {
	if req.Msg.PipelineId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("pipeline_id is required"))
	}

	slog.Info("rpc: BackfillPipeline", "pipeline_id", req.Msg.PipelineId)
	if err := s.engine.BackfillPipeline(ctx, uint(req.Msg.PipelineId)); err != nil {
		slog.Error("rpc: BackfillPipeline failed", "pipeline_id", req.Msg.PipelineId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("triggering backfill: %w", err))
	}

	slog.Info("rpc: BackfillPipeline triggered", "pipeline_id", req.Msg.PipelineId)
	return connect.NewResponse(&v1.BackfillPipelineResponse{}), nil
}

func (s *PipelineService) GetPipelineStatus(ctx context.Context, req *connect.Request[v1.GetPipelineStatusRequest]) (*connect.Response[v1.GetPipelineStatusResponse], error) {
	statuses := s.engine.Status()

	slog.Info("rpc: GetPipelineStatus", "pipeline_count", len(statuses))

	resp := &v1.GetPipelineStatusResponse{
		Statuses: make([]*v1.PipelineStatus, len(statuses)),
	}
	for i, st := range statuses {
		resp.Statuses[i] = &v1.PipelineStatus{
			PipelineId:   uint64(st.PipelineID),
			PipelineName: st.PipelineName,
			ProductName:  st.ProductName,
			Running:      st.Running,
			Healthy:      st.Healthy,
		}
	}

	return connect.NewResponse(resp), nil
}

func pipelineToProto(p models.Pipeline) *v1.Pipeline {
	proto := &v1.Pipeline{
		Id:         uint64(p.ID),
		CreatedAt:  timestamppb.New(p.CreatedAt),
		UpdatedAt:  timestamppb.New(p.UpdatedAt),
		Name:       p.Name,
		Enabled:    p.Enabled,
		ProductId:  uint64(p.ProductID),
		PlatformId: uint64(p.PlatformID),
		AgentId:    uint64(p.AgentID),
	}

	if p.Product.ID != 0 {
		proto.Product = productToProto(p.Product)
	}
	if p.Platform.ID != 0 {
		proto.Platform = platformToProto(p.Platform)
	}
	if p.Agent.ID != 0 {
		proto.Agent = agentToProto(p.Agent)
	}

	for _, rc := range p.Reporters {
		proto.ReporterIds = append(proto.ReporterIds, uint64(rc.ID))
		proto.Reporters = append(proto.Reporters, reporterToProto(rc))
	}

	return proto
}

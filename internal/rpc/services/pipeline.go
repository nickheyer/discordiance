package services

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/engine"
	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PipelineService struct {
	discordiancev1connect.UnimplementedPipelineServiceHandler
	db     *gorm.DB
	engine *engine.Engine
}

func NewPipelineService(db *gorm.DB, eng *engine.Engine) *PipelineService {
	return &PipelineService{db: db, engine: eng}
}

func (s *PipelineService) CreatePipeline(_ context.Context, req *connect.Request[v1.CreatePipelineRequest]) (*connect.Response[v1.CreatePipelineResponse], error) {
	id := uuid.NewString()

	pipeline := models.Pipeline{
		ID:          id,
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
		ProductID:   req.Msg.ProductId,
		Status:      int32(v1.PipelineStatus_PIPELINE_STATUS_IDLE),
	}

	if cs := req.Msg.ClassificationStrategy; cs != nil {
		pipeline.BatchMode = int32(cs.BatchMode)
		pipeline.BatchSize = cs.BatchSize
		pipeline.TimeWindowSeconds = cs.TimeWindowSeconds
	} else {
		pipeline.BatchMode = int32(v1.BatchMode_BATCH_MODE_SINGLE)
	}

	if err := s.db.Create(&pipeline).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := savePipelineJoins(s.db, id, req.Msg.AgentIds, req.Msg.PlatformIds, req.Msg.ReporterIds); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	full, err := loadPipelineFull(s.db, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.CreatePipelineResponse{
		Pipeline: pipelineToProto(full),
	}), nil
}

func (s *PipelineService) GetPipeline(_ context.Context, req *connect.Request[v1.GetPipelineRequest]) (*connect.Response[v1.GetPipelineResponse], error) {
	p, err := loadPipelineFull(s.db, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&v1.GetPipelineResponse{
		Pipeline: pipelineToProto(p),
	}), nil
}

func (s *PipelineService) ListPipelines(_ context.Context, req *connect.Request[v1.ListPipelinesRequest]) (*connect.Response[v1.ListPipelinesResponse], error) {
	pageSize, offset := parsePagination(req.Msg.Pagination)

	var pipelines []models.Pipeline
	var total int64
	s.db.Model(&models.Pipeline{}).Count(&total)
	s.db.Preload("Agents").Preload("Platforms").Preload("Reporters").
		Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&pipelines)

	protos := make([]*v1.Pipeline, len(pipelines))
	for i := range pipelines {
		protos[i] = pipelineToProto(&pipelines[i])
	}

	return connect.NewResponse(&v1.ListPipelinesResponse{
		Pipelines:  protos,
		Pagination: buildPaginationResponse(offset, pageSize, int(total)),
	}), nil
}

func (s *PipelineService) UpdatePipeline(_ context.Context, req *connect.Request[v1.UpdatePipelineRequest]) (*connect.Response[v1.UpdatePipelineResponse], error) {
	var pipeline models.Pipeline
	if err := s.db.First(&pipeline, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	pipeline.Name = req.Msg.Name
	pipeline.Description = req.Msg.Description
	pipeline.ProductID = req.Msg.ProductId

	if cs := req.Msg.ClassificationStrategy; cs != nil {
		pipeline.BatchMode = int32(cs.BatchMode)
		pipeline.BatchSize = cs.BatchSize
		pipeline.TimeWindowSeconds = cs.TimeWindowSeconds
	}

	if err := s.db.Save(&pipeline).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Replace join tables
	s.db.Where("pipeline_id = ?", req.Msg.Id).Delete(&models.PipelineAgent{})
	s.db.Where("pipeline_id = ?", req.Msg.Id).Delete(&models.PipelinePlatform{})
	s.db.Where("pipeline_id = ?", req.Msg.Id).Delete(&models.PipelineReporter{})

	if err := savePipelineJoins(s.db, req.Msg.Id, req.Msg.AgentIds, req.Msg.PlatformIds, req.Msg.ReporterIds); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	full, err := loadPipelineFull(s.db, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.UpdatePipelineResponse{
		Pipeline: pipelineToProto(full),
	}), nil
}

func (s *PipelineService) DeletePipeline(_ context.Context, req *connect.Request[v1.DeletePipelineRequest]) (*connect.Response[v1.DeletePipelineResponse], error) {
	if err := s.db.Delete(&models.Pipeline{}, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeletePipelineResponse{}), nil
}

func (s *PipelineService) StartPipeline(_ context.Context, req *connect.Request[v1.StartPipelineRequest]) (*connect.Response[v1.StartPipelineResponse], error) {
	if err := s.db.Model(&models.Pipeline{}).Where("id = ?", req.Msg.Id).Update("status", int32(v1.PipelineStatus_PIPELINE_STATUS_RUNNING)).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.engine.StartPipeline(req.Msg.Id)
	full, err := loadPipelineFull(s.db, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.StartPipelineResponse{
		Pipeline: pipelineToProto(full),
	}), nil
}

func (s *PipelineService) StopPipeline(_ context.Context, req *connect.Request[v1.StopPipelineRequest]) (*connect.Response[v1.StopPipelineResponse], error) {
	s.engine.StopPipeline(req.Msg.Id)
	if err := s.db.Model(&models.Pipeline{}).Where("id = ?", req.Msg.Id).Update("status", int32(v1.PipelineStatus_PIPELINE_STATUS_IDLE)).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	full, err := loadPipelineFull(s.db, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.StopPipelineResponse{
		Pipeline: pipelineToProto(full),
	}), nil
}

func (s *PipelineService) PausePipeline(_ context.Context, req *connect.Request[v1.PausePipelineRequest]) (*connect.Response[v1.PausePipelineResponse], error) {
	s.engine.StopPipeline(req.Msg.Id)
	if err := s.db.Model(&models.Pipeline{}).Where("id = ?", req.Msg.Id).Update("status", int32(v1.PipelineStatus_PIPELINE_STATUS_PAUSED)).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	full, err := loadPipelineFull(s.db, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.PausePipelineResponse{
		Pipeline: pipelineToProto(full),
	}), nil
}

func (s *PipelineService) GetPipelineStatus(_ context.Context, req *connect.Request[v1.GetPipelineStatusRequest]) (*connect.Response[v1.GetPipelineStatusResponse], error) {
	full, err := loadPipelineFull(s.db, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&v1.GetPipelineStatusResponse{
		Pipeline: pipelineToProto(full),
	}), nil
}

func savePipelineJoins(db *gorm.DB, pipelineID string, agentIDs, platformIDs, reporterIDs []string) error {
	for _, id := range agentIDs {
		if err := db.Create(&models.PipelineAgent{PipelineID: pipelineID, AgentID: id}).Error; err != nil {
			return err
		}
	}
	for _, id := range platformIDs {
		if err := db.Create(&models.PipelinePlatform{PipelineID: pipelineID, PlatformID: id}).Error; err != nil {
			return err
		}
	}
	for _, id := range reporterIDs {
		if err := db.Create(&models.PipelineReporter{PipelineID: pipelineID, ReporterID: id}).Error; err != nil {
			return err
		}
	}
	return nil
}

func loadPipelineFull(db *gorm.DB, id string) (*models.Pipeline, error) {
	var p models.Pipeline
	err := db.Preload("Agents").Preload("Platforms").Preload("Reporters").
		First(&p, "id = ?", id).Error
	return &p, err
}

func pipelineToProto(p *models.Pipeline) *v1.Pipeline {
	agentIDs := make([]string, len(p.Agents))
	for i, a := range p.Agents {
		agentIDs[i] = a.AgentID
	}
	platformIDs := make([]string, len(p.Platforms))
	for i, pl := range p.Platforms {
		platformIDs[i] = pl.PlatformID
	}
	reporterIDs := make([]string, len(p.Reporters))
	for i, r := range p.Reporters {
		reporterIDs[i] = r.ReporterID
	}

	proto := &v1.Pipeline{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		ProductId:   p.ProductID,
		AgentIds:    agentIDs,
		PlatformIds: platformIDs,
		ReporterIds: reporterIDs,
		Status:      v1.PipelineStatus(p.Status),
		ClassificationStrategy: &v1.ClassificationStrategy{
			BatchMode:         v1.BatchMode(p.BatchMode),
			BatchSize:         p.BatchSize,
			TimeWindowSeconds: p.TimeWindowSeconds,
		},
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}

	return proto
}

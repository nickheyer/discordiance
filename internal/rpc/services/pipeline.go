package services

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/nickheyer/discordiance/internal/engine"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
)

var _ discordiancev1connect.PipelineServiceHandler = (*PipelineService)(nil)

type PipelineService struct {
	engine *engine.Engine
}

func NewPipelineService(eng *engine.Engine) *PipelineService {
	return &PipelineService{engine: eng}
}

func (s *PipelineService) StartPipeline(ctx context.Context, req *connect.Request[v1.StartPipelineRequest]) (*connect.Response[v1.StartPipelineResponse], error) {
	if req.Msg.ProductId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("product_id is required"))
	}

	slog.Info("rpc: StartPipeline", "product_id", req.Msg.ProductId)
	if err := s.engine.StartProduct(ctx, uint(req.Msg.ProductId)); err != nil {
		slog.Error("rpc: StartPipeline failed", "product_id", req.Msg.ProductId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("starting pipeline: %w", err))
	}

	slog.Info("rpc: StartPipeline success", "product_id", req.Msg.ProductId)
	return connect.NewResponse(&v1.StartPipelineResponse{}), nil
}

func (s *PipelineService) StopPipeline(ctx context.Context, req *connect.Request[v1.StopPipelineRequest]) (*connect.Response[v1.StopPipelineResponse], error) {
	if req.Msg.ProductId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("product_id is required"))
	}

	slog.Info("rpc: StopPipeline", "product_id", req.Msg.ProductId)
	if err := s.engine.StopProduct(ctx, uint(req.Msg.ProductId)); err != nil {
		slog.Error("rpc: StopPipeline failed", "product_id", req.Msg.ProductId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("stopping pipeline: %w", err))
	}

	slog.Info("rpc: StopPipeline success", "product_id", req.Msg.ProductId)
	return connect.NewResponse(&v1.StopPipelineResponse{}), nil
}

func (s *PipelineService) RestartPipeline(ctx context.Context, req *connect.Request[v1.RestartPipelineRequest]) (*connect.Response[v1.RestartPipelineResponse], error) {
	if req.Msg.ProductId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("product_id is required"))
	}

	slog.Info("rpc: RestartPipeline", "product_id", req.Msg.ProductId)
	if err := s.engine.RestartProduct(ctx, uint(req.Msg.ProductId)); err != nil {
		slog.Error("rpc: RestartPipeline failed", "product_id", req.Msg.ProductId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("restarting pipeline: %w", err))
	}

	slog.Info("rpc: RestartPipeline success", "product_id", req.Msg.ProductId)
	return connect.NewResponse(&v1.RestartPipelineResponse{}), nil
}

func (s *PipelineService) BackfillPipeline(ctx context.Context, req *connect.Request[v1.BackfillPipelineRequest]) (*connect.Response[v1.BackfillPipelineResponse], error) {
	if req.Msg.ProductId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("product_id is required"))
	}

	slog.Info("rpc: BackfillPipeline", "product_id", req.Msg.ProductId)
	if err := s.engine.BackfillProduct(ctx, uint(req.Msg.ProductId)); err != nil {
		slog.Error("rpc: BackfillPipeline failed", "product_id", req.Msg.ProductId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("triggering backfill: %w", err))
	}

	slog.Info("rpc: BackfillPipeline triggered", "product_id", req.Msg.ProductId)
	return connect.NewResponse(&v1.BackfillPipelineResponse{}), nil
}

func (s *PipelineService) GetPipelineStatus(ctx context.Context, req *connect.Request[v1.GetPipelineStatusRequest]) (*connect.Response[v1.GetPipelineStatusResponse], error) {
	statuses := s.engine.Status()

	slog.Info("rpc: GetPipelineStatus", "pipeline_count", len(statuses))

	resp := &v1.GetPipelineStatusResponse{
		Statuses: make([]*v1.ProductStatus, len(statuses)),
	}
	for i, st := range statuses {
		resp.Statuses[i] = &v1.ProductStatus{
			ProductId:   uint64(st.ProductID),
			ProductName: st.ProductName,
			Running:     st.Running,
			Healthy:     st.Healthy,
		}
	}

	return connect.NewResponse(resp), nil
}

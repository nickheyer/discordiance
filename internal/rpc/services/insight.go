package services

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InsightService struct {
	discordiancev1connect.UnimplementedInsightServiceHandler
	db *gorm.DB
}

func NewInsightService(db *gorm.DB) *InsightService {
	return &InsightService{db: db}
}

func (s *InsightService) GetInsight(_ context.Context, req *connect.Request[v1.GetInsightRequest]) (*connect.Response[v1.GetInsightResponse], error) {
	var insight models.Insight
	if err := s.db.First(&insight, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&v1.GetInsightResponse{
		Insight: insightToProto(&insight),
	}), nil
}

func (s *InsightService) ListInsights(_ context.Context, req *connect.Request[v1.ListInsightsRequest]) (*connect.Response[v1.ListInsightsResponse], error) {
	pageSize, offset := parsePagination(req.Msg.Pagination)

	query := s.db.Model(&models.Insight{})

	if req.Msg.PipelineId != "" {
		query = query.Where("pipeline_id = ?", req.Msg.PipelineId)
	}
	if req.Msg.PlatformId != "" {
		query = query.Where("platform_id = ?", req.Msg.PlatformId)
	}
	if req.Msg.State != v1.InsightState_INSIGHT_STATE_UNSPECIFIED {
		query = query.Where("state = ?", int32(req.Msg.State))
	}
	if req.Msg.Medium != v1.InsightMedium_INSIGHT_MEDIUM_UNSPECIFIED {
		query = query.Where("medium = ?", int32(req.Msg.Medium))
	}
	if req.Msg.FromTimestamp != nil {
		query = query.Where("source_timestamp >= ?", req.Msg.FromTimestamp.AsTime())
	}
	if req.Msg.ToTimestamp != nil {
		query = query.Where("source_timestamp <= ?", req.Msg.ToTimestamp.AsTime())
	}

	var total int64
	query.Count(&total)

	var insights []models.Insight
	query.Limit(pageSize).Offset(offset).Order("ingested_at DESC").Find(&insights)

	protos := make([]*v1.Insight, len(insights))
	for i := range insights {
		protos[i] = insightToProto(&insights[i])
	}

	return connect.NewResponse(&v1.ListInsightsResponse{
		Insights:   protos,
		Pagination: buildPaginationResponse(offset, pageSize, int(total)),
	}), nil
}

func (s *InsightService) ClassifyInsight(_ context.Context, req *connect.Request[v1.ClassifyInsightRequest]) (*connect.Response[v1.ClassifyInsightResponse], error) {
	var insight models.Insight
	if err := s.db.First(&insight, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	now := time.Now()
	insight.State = int32(req.Msg.State)
	insight.ClassificationReason = req.Msg.Reason
	insight.ClassifyingAgentID = "manual"
	insight.ClassifiedAt = &now

	if err := s.db.Save(&insight).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.ClassifyInsightResponse{
		Insight: insightToProto(&insight),
	}), nil
}

func (s *InsightService) GetInsightStats(_ context.Context, req *connect.Request[v1.GetInsightStatsRequest]) (*connect.Response[v1.GetInsightStatsResponse], error) {
	query := s.db.Model(&models.Insight{})

	if req.Msg.PipelineId != "" {
		query = query.Where("pipeline_id = ?", req.Msg.PipelineId)
	}
	if req.Msg.PlatformId != "" {
		query = query.Where("platform_id = ?", req.Msg.PlatformId)
	}
	if req.Msg.FromTimestamp != nil {
		query = query.Where("source_timestamp >= ?", req.Msg.FromTimestamp.AsTime())
	}
	if req.Msg.ToTimestamp != nil {
		query = query.Where("source_timestamp <= ?", req.Msg.ToTimestamp.AsTime())
	}

	var total int64
	query.Session(&gorm.Session{}).Count(&total)

	countByState := func(state v1.InsightState) int32 {
		var c int64
		query.Session(&gorm.Session{}).Where("state = ?", int32(state)).Count(&c)
		return int32(c)
	}

	stats := &v1.InsightStats{
		Total:     int32(total),
		RawCount:  countByState(v1.InsightState_INSIGHT_STATE_RAW),
		ColdCount: countByState(v1.InsightState_INSIGHT_STATE_COLD),
		HotCount:  countByState(v1.InsightState_INSIGHT_STATE_HOT),
		BurnCount: countByState(v1.InsightState_INSIGHT_STATE_BURN),
	}

	return connect.NewResponse(&v1.GetInsightStatsResponse{
		Stats: stats,
	}), nil
}

func insightToProto(i *models.Insight) *v1.Insight {
	proto := &v1.Insight{
		Id:                   i.ID,
		PipelineId:           i.PipelineID,
		PlatformId:           i.PlatformID,
		State:                v1.InsightState(i.State),
		Medium:               v1.InsightMedium(i.Medium),
		Content:              i.Content,
		Author:               i.Author,
		SourceUrl:            i.SourceURL,
		SourceId:             i.SourceID,
		ConversationId:       i.ConversationID,
		ClassificationReason: i.ClassificationReason,
		ClassifyingAgentId:   i.ClassifyingAgentID,
		SourceTimestamp:      timestamppb.New(i.SourceTimestamp),
		IngestedAt:           timestamppb.New(i.IngestedAt),
	}

	if i.ClassifiedAt != nil {
		proto.ClassifiedAt = timestamppb.New(*i.ClassifiedAt)
	}

	return proto
}

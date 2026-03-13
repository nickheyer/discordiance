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
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

var _ discordiancev1connect.InsightServiceHandler = (*InsightService)(nil)

type InsightService struct {
	db *gorm.DB
}

func NewInsightService(db *gorm.DB) *InsightService {
	return &InsightService{db: db}
}

func (s *InsightService) ListInsights(ctx context.Context, req *connect.Request[v1.ListInsightsRequest]) (*connect.Response[v1.ListInsightsResponse], error) {
	slog.Info("rpc: ListInsights", "product_id", req.Msg.ProductId, "status", req.Msg.Status, "page", req.Msg.Page, "page_size", req.Msg.PageSize)

	query := s.db.Model(&models.Insight{}).Preload("Product").Preload("Reports")

	if req.Msg.ProductId > 0 {
		query = query.Where("product_id = ?", req.Msg.ProductId)
	}
	if req.Msg.Status != "" {
		query = query.Where("status = ?", req.Msg.Status)
	}

	var totalCount int64
	query.Count(&totalCount)

	pageSize := int(req.Msg.PageSize)
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := int(req.Msg.Page)
	if page <= 0 {
		page = 1
	}

	var insights []models.Insight
	if err := query.
		Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&insights).Error; err != nil {
		slog.Error("rpc: ListInsights query failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("listing insights: %w", err))
	}

	slog.Info("rpc: ListInsights result", "returned", len(insights), "total", totalCount)

	resp := &v1.ListInsightsResponse{
		Insights:   make([]*v1.Insight, len(insights)),
		TotalCount: int32(totalCount),
	}
	for i, insight := range insights {
		resp.Insights[i] = insightToProto(insight)
	}

	return connect.NewResponse(resp), nil
}

func (s *InsightService) GetInsight(ctx context.Context, req *connect.Request[v1.GetInsightRequest]) (*connect.Response[v1.GetInsightResponse], error) {
	slog.Info("rpc: GetInsight", "id", req.Msg.Id)

	var insight models.Insight
	if err := s.db.
		Preload("Product").
		Preload("Reports").
		First(&insight, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: GetInsight not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("insight %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("rpc: GetInsight found", "id", insight.ID, "title", insight.Title, "status", insight.Status)
	return connect.NewResponse(&v1.GetInsightResponse{
		Insight: insightToProto(insight),
	}), nil
}

func (s *InsightService) UpdateInsightStatus(ctx context.Context, req *connect.Request[v1.UpdateInsightStatusRequest]) (*connect.Response[v1.UpdateInsightStatusResponse], error) {
	slog.Info("rpc: UpdateInsightStatus", "id", req.Msg.Id, "new_status", req.Msg.Status)

	validStatuses := map[string]bool{
		"open": true, "acknowledged": true, "resolved": true, "closed": true,
	}
	if !validStatuses[req.Msg.Status] {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid status: %s (must be open, acknowledged, resolved, or closed)", req.Msg.Status))
	}

	var insight models.Insight
	if err := s.db.First(&insight, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: UpdateInsightStatus not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("insight %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	oldStatus := insight.Status
	insight.Status = req.Msg.Status
	if err := s.db.Save(&insight).Error; err != nil {
		slog.Error("rpc: UpdateInsightStatus failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating insight status: %w", err))
	}

	slog.Info("rpc: UpdateInsightStatus success", "id", insight.ID, "old_status", oldStatus, "new_status", insight.Status)

	s.db.Preload("Product").Preload("Reports").First(&insight, insight.ID)

	return connect.NewResponse(&v1.UpdateInsightStatusResponse{
		Insight: insightToProto(insight),
	}), nil
}

func (s *InsightService) FileInsightToReporter(ctx context.Context, req *connect.Request[v1.FileInsightToReporterRequest]) (*connect.Response[v1.FileInsightToReporterResponse], error) {
	slog.Info("rpc: FileInsightToReporter", "insight_id", req.Msg.InsightId, "reporter_id", req.Msg.ReporterId)

	// Load the insight
	var insight models.Insight
	if err := s.db.First(&insight, req.Msg.InsightId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("insight %d not found", req.Msg.InsightId))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Load the reporter
	var rc models.Reporter
	if err := s.db.First(&rc, req.Msg.ReporterId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("reporter %d not found", req.Msg.ReporterId))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Instantiate and init the reporter
	r, err := reporter.Create(rc.Type)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating reporter %s: %w", rc.Type, err))
	}
	if err := r.Init(ctx, rc); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("initializing reporter %s: %w", rc.Type, err))
	}
	defer r.Close(ctx)

	// Check for duplicate
	dupID, err := r.FindDuplicate(ctx, insight)
	if err != nil {
		slog.Warn("rpc: FileInsightToReporter duplicate check failed", "error", err)
	}
	if dupID != "" {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("duplicate found in %s: external ID %s", rc.Type, dupID))
	}

	// File the report
	report, err := r.FileReport(ctx, insight)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("filing report: %w", err))
	}

	if err := s.db.Create(report).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("saving report: %w", err))
	}

	slog.Info("rpc: FileInsightToReporter success", "insight_id", insight.ID, "reporter", rc.Type, "external_url", report.ExternalURL)

	return connect.NewResponse(&v1.FileInsightToReporterResponse{
		Report: reportToProto(*report),
	}), nil
}

func insightToProto(i models.Insight) *v1.Insight {
	proto := &v1.Insight{
		Id:           uint64(i.ID),
		CreatedAt:    timestamppb.New(i.CreatedAt),
		UpdatedAt:    timestamppb.New(i.UpdatedAt),
		ProductId:    uint64(i.ProductID),
		Title:        i.Title,
		Description:  i.Description,
		Severity:     i.Severity,
		Category:     i.Category,
		Fingerprint:  i.Fingerprint,
		Status:       i.Status,
		SourceMsgIds: i.SourceMsgIDs,
		ProductName:  i.Product.Name,
	}

	for _, r := range i.Reports {
		proto.Reports = append(proto.Reports, reportToProto(r))
	}

	return proto
}

func reportToProto(r models.Report) *v1.Report {
	return &v1.Report{
		Id:           uint64(r.ID),
		CreatedAt:    timestamppb.New(r.CreatedAt),
		UpdatedAt:    timestamppb.New(r.UpdatedAt),
		InsightId:    uint64(r.InsightID),
		ReporterType: r.ReporterType,
		ExternalId:   r.ExternalID,
		ExternalUrl:  r.ExternalURL,
		Status:       r.Status,
	}
}

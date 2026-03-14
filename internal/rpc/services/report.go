package services

import (
	"context"

	"connectrpc.com/connect"
	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ReportService struct {
	discordiancev1connect.UnimplementedReportServiceHandler
	db *gorm.DB
}

func NewReportService(db *gorm.DB) *ReportService {
	return &ReportService{db: db}
}

func (s *ReportService) GetReport(_ context.Context, req *connect.Request[v1.GetReportRequest]) (*connect.Response[v1.GetReportResponse], error) {
	var report models.Report
	if err := s.db.Preload("Entries").First(&report, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&v1.GetReportResponse{
		Report: reportToProto(&report),
	}), nil
}

func (s *ReportService) ListReports(_ context.Context, req *connect.Request[v1.ListReportsRequest]) (*connect.Response[v1.ListReportsResponse], error) {
	pageSize, offset := parsePagination(req.Msg.Pagination)

	query := s.db.Model(&models.Report{})
	if req.Msg.PipelineId != "" {
		query = query.Where("pipeline_id = ?", req.Msg.PipelineId)
	}

	var total int64
	query.Count(&total)

	var reports []models.Report
	query.Preload("Entries").Limit(pageSize).Offset(offset).Order("generated_at DESC").Find(&reports)

	protos := make([]*v1.Report, len(reports))
	for i := range reports {
		protos[i] = reportToProto(&reports[i])
	}

	return connect.NewResponse(&v1.ListReportsResponse{
		Reports:    protos,
		Pagination: buildPaginationResponse(offset, pageSize, int(total)),
	}), nil
}

func reportToProto(r *models.Report) *v1.Report {
	entries := make([]*v1.ReportEntry, len(r.Entries))
	for i, e := range r.Entries {
		entries[i] = &v1.ReportEntry{
			InsightId:            e.InsightID,
			State:                v1.InsightState(e.State),
			ContentPreview:       e.ContentPreview,
			ClassificationReason: e.ClassificationReason,
			PlatformId:           e.PlatformID,
			SourceUrl:            e.SourceURL,
		}
	}

	return &v1.Report{
		Id:         r.ID,
		PipelineId: r.PipelineID,
		ProductId:  r.ProductID,
		Summary: &v1.ReportSummary{
			TotalInsights: r.TotalInsights,
			HotCount:      r.HotCount,
			BurnCount:     r.BurnCount,
			ColdCount:     r.ColdCount,
		},
		Entries:     entries,
		GeneratedAt: timestamppb.New(r.GeneratedAt),
	}
}

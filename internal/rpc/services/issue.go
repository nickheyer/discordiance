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

var _ discordiancev1connect.IssueServiceHandler = (*IssueService)(nil)

type IssueService struct {
	db *gorm.DB
}

func NewIssueService(db *gorm.DB) *IssueService {
	return &IssueService{db: db}
}

func (s *IssueService) ListIssues(ctx context.Context, req *connect.Request[v1.ListIssuesRequest]) (*connect.Response[v1.ListIssuesResponse], error) {
	slog.Info("rpc: ListIssues", "product_id", req.Msg.ProductId, "status", req.Msg.Status, "page", req.Msg.Page, "page_size", req.Msg.PageSize)

	query := s.db.Model(&models.Issue{}).Preload("Product").Preload("Reports")

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

	var issues []models.Issue
	if err := query.
		Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&issues).Error; err != nil {
		slog.Error("rpc: ListIssues query failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("listing issues: %w", err))
	}

	slog.Info("rpc: ListIssues result", "returned", len(issues), "total", totalCount)

	resp := &v1.ListIssuesResponse{
		Issues:     make([]*v1.Issue, len(issues)),
		TotalCount: int32(totalCount),
	}
	for i, issue := range issues {
		resp.Issues[i] = issueToProto(issue)
	}

	return connect.NewResponse(resp), nil
}

func (s *IssueService) GetIssue(ctx context.Context, req *connect.Request[v1.GetIssueRequest]) (*connect.Response[v1.GetIssueResponse], error) {
	slog.Info("rpc: GetIssue", "id", req.Msg.Id)

	var issue models.Issue
	if err := s.db.
		Preload("Product").
		Preload("Reports").
		First(&issue, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: GetIssue not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("issue %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("rpc: GetIssue found", "id", issue.ID, "title", issue.Title, "status", issue.Status)
	return connect.NewResponse(&v1.GetIssueResponse{
		Issue: issueToProto(issue),
	}), nil
}

func (s *IssueService) UpdateIssueStatus(ctx context.Context, req *connect.Request[v1.UpdateIssueStatusRequest]) (*connect.Response[v1.UpdateIssueStatusResponse], error) {
	slog.Info("rpc: UpdateIssueStatus", "id", req.Msg.Id, "new_status", req.Msg.Status)

	validStatuses := map[string]bool{
		"open": true, "acknowledged": true, "resolved": true, "closed": true,
	}
	if !validStatuses[req.Msg.Status] {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid status: %s (must be open, acknowledged, resolved, or closed)", req.Msg.Status))
	}

	var issue models.Issue
	if err := s.db.First(&issue, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: UpdateIssueStatus not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("issue %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	oldStatus := issue.Status
	issue.Status = req.Msg.Status
	if err := s.db.Save(&issue).Error; err != nil {
		slog.Error("rpc: UpdateIssueStatus failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating issue status: %w", err))
	}

	slog.Info("rpc: UpdateIssueStatus success", "id", issue.ID, "old_status", oldStatus, "new_status", issue.Status)

	s.db.Preload("Product").Preload("Reports").First(&issue, issue.ID)

	return connect.NewResponse(&v1.UpdateIssueStatusResponse{
		Issue: issueToProto(issue),
	}), nil
}

func issueToProto(i models.Issue) *v1.Issue {
	proto := &v1.Issue{
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
		IssueId:      uint64(r.IssueID),
		ReporterType: r.ReporterType,
		ExternalId:   r.ExternalID,
		ExternalUrl:  r.ExternalURL,
		Status:       r.Status,
	}
}

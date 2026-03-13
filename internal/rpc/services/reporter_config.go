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

var _ discordiancev1connect.ReporterServiceHandler = (*ReporterService)(nil)

type ReporterService struct {
	db *gorm.DB
}

func NewReporterService(db *gorm.DB) *ReporterService {
	return &ReporterService{db: db}
}

func (s *ReporterService) ListReporters(ctx context.Context, req *connect.Request[v1.ListReportersRequest]) (*connect.Response[v1.ListReportersResponse], error) {
	var reporters []models.Reporter
	if err := s.db.Find(&reporters).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("listing reporters: %w", err))
	}
	slog.Info("rpc: ListReporters", "count", len(reporters))
	resp := &v1.ListReportersResponse{
		Reporters: make([]*v1.Reporter, len(reporters)),
	}
	for i, c := range reporters {
		resp.Reporters[i] = reporterToProto(c)
	}
	return connect.NewResponse(resp), nil
}

func (s *ReporterService) GetReporter(ctx context.Context, req *connect.Request[v1.GetReporterRequest]) (*connect.Response[v1.GetReporterResponse], error) {
	var cfg models.Reporter
	if err := s.db.First(&cfg, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("reporter %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetReporterResponse{
		Reporter: reporterToProto(cfg),
	}), nil
}

func (s *ReporterService) CreateReporter(ctx context.Context, req *connect.Request[v1.CreateReporterRequest]) (*connect.Response[v1.CreateReporterResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	if req.Msg.Type == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("type is required"))
	}

	slog.Info("rpc: CreateReporter", "name", req.Msg.Name, "type", req.Msg.Type)

	cfg := models.Reporter{
		Name:    req.Msg.Name,
		Type:    req.Msg.Type,
		Enabled: req.Msg.Enabled,
	}
	applyReporterSettings(&cfg, req.Msg.GetGithubSettings())

	if err := s.db.Create(&cfg).Error; err != nil {
		slog.Error("rpc: CreateReporter failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating reporter: %w", err))
	}

	slog.Info("rpc: CreateReporter success", "id", cfg.ID, "type", cfg.Type)
	return connect.NewResponse(&v1.CreateReporterResponse{
		Reporter: reporterToProto(cfg),
	}), nil
}

func (s *ReporterService) UpdateReporter(ctx context.Context, req *connect.Request[v1.UpdateReporterRequest]) (*connect.Response[v1.UpdateReporterResponse], error) {
	slog.Info("rpc: UpdateReporter", "id", req.Msg.Id, "type", req.Msg.Type)

	var cfg models.Reporter
	if err := s.db.First(&cfg, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: UpdateReporter not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("reporter %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cfg.Name = req.Msg.Name
	cfg.Type = req.Msg.Type
	cfg.Enabled = req.Msg.Enabled
	applyReporterSettings(&cfg, req.Msg.GetGithubSettings())

	if err := s.db.Save(&cfg).Error; err != nil {
		slog.Error("rpc: UpdateReporter failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating reporter: %w", err))
	}

	slog.Info("rpc: UpdateReporter success", "id", cfg.ID, "type", cfg.Type)
	return connect.NewResponse(&v1.UpdateReporterResponse{
		Reporter: reporterToProto(cfg),
	}), nil
}

func (s *ReporterService) DeleteReporter(ctx context.Context, req *connect.Request[v1.DeleteReporterRequest]) (*connect.Response[v1.DeleteReporterResponse], error) {
	slog.Info("rpc: DeleteReporter", "id", req.Msg.Id)

	if err := s.db.Delete(&models.Reporter{}, req.Msg.Id).Error; err != nil {
		slog.Error("rpc: DeleteReporter failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("deleting reporter: %w", err))
	}

	slog.Info("rpc: DeleteReporter success", "id", req.Msg.Id)
	return connect.NewResponse(&v1.DeleteReporterResponse{}), nil
}

func (s *ReporterService) ListAvailableReporters(ctx context.Context, req *connect.Request[v1.ListAvailableReportersRequest]) (*connect.Response[v1.ListAvailableReportersResponse], error) {
	reporters := reporter.Available()
	slog.Info("rpc: ListAvailableReporters", "count", len(reporters))
	return connect.NewResponse(&v1.ListAvailableReportersResponse{
		Reporters: reporters,
	}), nil
}

func applyReporterSettings(cfg *models.Reporter, gs *v1.GitHubReporterSettings) {
	if gs != nil {
		cfg.Token = gs.Token
		cfg.Repo = gs.Repo
		cfg.Labels = gs.Labels
		cfg.AutoFile = gs.AutoFile
	}
}

func reporterToProto(rc models.Reporter) *v1.Reporter {
	out := &v1.Reporter{
		Id:        uint64(rc.ID),
		CreatedAt: timestamppb.New(rc.CreatedAt),
		UpdatedAt: timestamppb.New(rc.UpdatedAt),
		Name:      rc.Name,
		Type:      rc.Type,
		Enabled:   rc.Enabled,
	}
	switch rc.Type {
	case "github":
		out.Settings = &v1.Reporter_GithubSettings{
			GithubSettings: &v1.GitHubReporterSettings{
				Token:    rc.Token,
				Repo:     rc.Repo,
				Labels:   rc.Labels,
				AutoFile: rc.AutoFile,
			},
		}
	}
	return out
}

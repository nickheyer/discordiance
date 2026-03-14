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

type ReporterService struct {
	discordiancev1connect.UnimplementedReporterServiceHandler
	db *gorm.DB
}

func NewReporterService(db *gorm.DB) *ReporterService {
	return &ReporterService{db: db}
}

func (s *ReporterService) CreateReporter(_ context.Context, req *connect.Request[v1.CreateReporterRequest]) (*connect.Response[v1.CreateReporterResponse], error) {
	id := uuid.NewString()

	reporter := models.Reporter{
		ID:   id,
		Name: req.Msg.Name,
		Type: int32(req.Msg.Type),
	}

	if err := s.db.Create(&reporter).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := saveReporterFilter(s.db, id, req.Msg.Filter); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := saveReporterConfig(s.db, id, req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	full, err := loadReporterFull(s.db, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.CreateReporterResponse{
		Reporter: reporterToProto(full),
	}), nil
}

func (s *ReporterService) GetReporter(_ context.Context, req *connect.Request[v1.GetReporterRequest]) (*connect.Response[v1.GetReporterResponse], error) {
	full, err := loadReporterFull(s.db, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&v1.GetReporterResponse{
		Reporter: reporterToProto(full),
	}), nil
}

func (s *ReporterService) ListReporters(_ context.Context, req *connect.Request[v1.ListReportersRequest]) (*connect.Response[v1.ListReportersResponse], error) {
	pageSize, offset := parsePagination(req.Msg.Pagination)

	var reporters []models.Reporter
	var total int64
	s.db.Model(&models.Reporter{}).Count(&total)
	s.db.Preload("FilterStates").Preload("FilterPlatforms").
		Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&reporters)

	protos := make([]*v1.Reporter, len(reporters))
	for i := range reporters {
		full, _ := loadReporterFull(s.db, reporters[i].ID)
		if full != nil {
			protos[i] = reporterToProto(full)
		}
	}

	return connect.NewResponse(&v1.ListReportersResponse{
		Reporters:  protos,
		Pagination: buildPaginationResponse(offset, pageSize, int(total)),
	}), nil
}

func (s *ReporterService) UpdateReporter(_ context.Context, req *connect.Request[v1.UpdateReporterRequest]) (*connect.Response[v1.UpdateReporterResponse], error) {
	var reporter models.Reporter
	if err := s.db.First(&reporter, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	reporter.Name = req.Msg.Name
	reporter.Type = int32(req.Msg.Type)

	if err := s.db.Save(&reporter).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Replace filters
	s.db.Where("reporter_id = ?", req.Msg.Id).Delete(&models.ReporterFilterState{})
	s.db.Where("reporter_id = ?", req.Msg.Id).Delete(&models.ReporterFilterPlatform{})
	if err := saveReporterFilter(s.db, req.Msg.Id, req.Msg.Filter); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := saveReporterConfig(s.db, req.Msg.Id, req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	full, err := loadReporterFull(s.db, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.UpdateReporterResponse{
		Reporter: reporterToProto(full),
	}), nil
}

func (s *ReporterService) DeleteReporter(_ context.Context, req *connect.Request[v1.DeleteReporterRequest]) (*connect.Response[v1.DeleteReporterResponse], error) {
	if err := s.db.Delete(&models.Reporter{}, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteReporterResponse{}), nil
}

func (s *ReporterService) TestReporter(_ context.Context, req *connect.Request[v1.TestReporterRequest]) (*connect.Response[v1.TestReporterResponse], error) {
	var reporter models.Reporter
	if err := s.db.First(&reporter, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&v1.TestReporterResponse{
		Success: true,
		Message: "reporter test not yet implemented",
	}), nil
}

func saveReporterFilter(db *gorm.DB, reporterID string, filter *v1.ReporterFilter) error {
	if filter == nil {
		return nil
	}
	for _, state := range filter.States {
		if err := db.Create(&models.ReporterFilterState{
			ReporterID: reporterID,
			State:      int32(state),
		}).Error; err != nil {
			return err
		}
	}
	for _, pid := range filter.PlatformIds {
		if err := db.Create(&models.ReporterFilterPlatform{
			ReporterID: reporterID,
			PlatformID: pid,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

type reporterConfigSource interface {
	GetWebhook() *v1.WebhookReporterConfig
	GetEmail() *v1.EmailReporterConfig
	GetDiscord() *v1.DiscordReporterConfig
	GetGithubIssue() *v1.GitHubIssueReporterConfig
}

func saveReporterConfig(db *gorm.DB, reporterID string, src reporterConfigSource) error {
	if c := src.GetWebhook(); c != nil {
		return db.Save(&models.WebhookReporterConfig{
			ReporterID: reporterID,
			URL:        c.Url,
			Secret:     c.Secret,
		}).Error
	}
	if c := src.GetEmail(); c != nil {
		return db.Save(&models.EmailReporterConfig{
			ReporterID:  reporterID,
			SMTPHost:    c.SmtpHost,
			SMTPPort:    c.SmtpPort,
			FromAddress: c.FromAddress,
			ToAddresses: marshalStringSlice(c.ToAddresses),
			Username:    c.Username,
			Password:    c.Password,
		}).Error
	}
	if c := src.GetDiscord(); c != nil {
		return db.Save(&models.DiscordReporterConfig{
			ReporterID: reporterID,
			WebhookURL: c.WebhookUrl,
		}).Error
	}
	if c := src.GetGithubIssue(); c != nil {
		return db.Save(&models.GitHubIssueReporterConfig{
			ReporterID: reporterID,
			Token:      c.Token,
			Owner:      c.Owner,
			Repo:       c.Repo,
			Labels:     marshalStringSlice(c.Labels),
		}).Error
	}
	return nil
}

func loadReporterFull(db *gorm.DB, id string) (*models.Reporter, error) {
	var r models.Reporter
	if err := db.Preload("FilterStates").Preload("FilterPlatforms").
		First(&r, "id = ?", id).Error; err != nil {
		return nil, err
	}

	switch v1.ReporterType(r.Type) {
	case v1.ReporterType_REPORTER_TYPE_WEBHOOK:
		var cfg models.WebhookReporterConfig
		if err := db.First(&cfg, "reporter_id = ?", id).Error; err == nil {
			r.WebhookConfig = &cfg
		}
	case v1.ReporterType_REPORTER_TYPE_EMAIL:
		var cfg models.EmailReporterConfig
		if err := db.First(&cfg, "reporter_id = ?", id).Error; err == nil {
			r.EmailConfig = &cfg
		}
	case v1.ReporterType_REPORTER_TYPE_DISCORD:
		var cfg models.DiscordReporterConfig
		if err := db.First(&cfg, "reporter_id = ?", id).Error; err == nil {
			r.DiscordConfig = &cfg
		}
	case v1.ReporterType_REPORTER_TYPE_GITHUB_ISSUE:
		var cfg models.GitHubIssueReporterConfig
		if err := db.First(&cfg, "reporter_id = ?", id).Error; err == nil {
			r.GitHubIssueConfig = &cfg
		}
	}

	return &r, nil
}

func reporterToProto(r *models.Reporter) *v1.Reporter {
	proto := &v1.Reporter{
		Id:        r.ID,
		Name:      r.Name,
		Type:      v1.ReporterType(r.Type),
		CreatedAt: timestamppb.New(r.CreatedAt),
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}

	// Filter
	if len(r.FilterStates) > 0 || len(r.FilterPlatforms) > 0 {
		filter := &v1.ReporterFilter{}
		for _, s := range r.FilterStates {
			filter.States = append(filter.States, v1.InsightState(s.State))
		}
		for _, p := range r.FilterPlatforms {
			filter.PlatformIds = append(filter.PlatformIds, p.PlatformID)
		}
		proto.Filter = filter
	}

	// Config
	if c := r.WebhookConfig; c != nil {
		proto.Config = &v1.Reporter_Webhook{
			Webhook: &v1.WebhookReporterConfig{
				Url:    c.URL,
				Secret: c.Secret,
			},
		}
	}
	if c := r.EmailConfig; c != nil {
		proto.Config = &v1.Reporter_Email{
			Email: &v1.EmailReporterConfig{
				SmtpHost:    c.SMTPHost,
				SmtpPort:    c.SMTPPort,
				FromAddress: c.FromAddress,
				ToAddresses: unmarshalStringSlice(c.ToAddresses),
				Username:    c.Username,
				Password:    c.Password,
			},
		}
	}
	if c := r.DiscordConfig; c != nil {
		proto.Config = &v1.Reporter_Discord{
			Discord: &v1.DiscordReporterConfig{
				WebhookUrl: c.WebhookURL,
			},
		}
	}
	if c := r.GitHubIssueConfig; c != nil {
		proto.Config = &v1.Reporter_GithubIssue{
			GithubIssue: &v1.GitHubIssueReporterConfig{
				Token:  c.Token,
				Owner:  c.Owner,
				Repo:   c.Repo,
				Labels: unmarshalStringSlice(c.Labels),
			},
		}
	}

	return proto
}

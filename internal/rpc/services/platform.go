package services

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/models"
	discordadapter "github.com/nickheyer/discordiance/internal/platform/discord"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PlatformService struct {
	discordiancev1connect.UnimplementedPlatformServiceHandler
	db *gorm.DB
}

func NewPlatformService(db *gorm.DB) *PlatformService {
	return &PlatformService{db: db}
}

func (s *PlatformService) CreatePlatform(_ context.Context, req *connect.Request[v1.CreatePlatformRequest]) (*connect.Response[v1.CreatePlatformResponse], error) {
	platform := models.Platform{
		ID:   uuid.NewString(),
		Name: req.Msg.Name,
		Type: int32(req.Msg.Type),
	}

	if err := s.db.Create(&platform).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := savePlatformConfig(s.db, platform.ID, req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return s.getPlatformResponse(platform.ID)
}

func (s *PlatformService) GetPlatform(_ context.Context, req *connect.Request[v1.GetPlatformRequest]) (*connect.Response[v1.GetPlatformResponse], error) {
	p, err := loadPlatformFull(s.db, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&v1.GetPlatformResponse{
		Platform: platformToProto(p),
	}), nil
}

func (s *PlatformService) ListPlatforms(_ context.Context, req *connect.Request[v1.ListPlatformsRequest]) (*connect.Response[v1.ListPlatformsResponse], error) {
	pageSize, offset := parsePagination(req.Msg.Pagination)

	var platforms []models.Platform
	var total int64
	s.db.Model(&models.Platform{}).Count(&total)
	s.db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&platforms)

	protos := make([]*v1.Platform, len(platforms))
	for i := range platforms {
		full, _ := loadPlatformFull(s.db, platforms[i].ID)
		if full != nil {
			protos[i] = platformToProto(full)
		}
	}

	return connect.NewResponse(&v1.ListPlatformsResponse{
		Platforms:  protos,
		Pagination: buildPaginationResponse(offset, pageSize, int(total)),
	}), nil
}

func (s *PlatformService) UpdatePlatform(_ context.Context, req *connect.Request[v1.UpdatePlatformRequest]) (*connect.Response[v1.UpdatePlatformResponse], error) {
	var platform models.Platform
	if err := s.db.First(&platform, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	platform.Name = req.Msg.Name
	platform.Type = int32(req.Msg.Type)

	if err := s.db.Save(&platform).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := savePlatformConfig(s.db, platform.ID, req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp, err := s.getPlatformResponse(platform.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.UpdatePlatformResponse{
		Platform: resp.Msg.Platform,
	}), nil
}

func (s *PlatformService) DeletePlatform(_ context.Context, req *connect.Request[v1.DeletePlatformRequest]) (*connect.Response[v1.DeletePlatformResponse], error) {
	if err := s.db.Delete(&models.Platform{}, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeletePlatformResponse{}), nil
}

func (s *PlatformService) TestPlatformConnection(_ context.Context, req *connect.Request[v1.TestPlatformConnectionRequest]) (*connect.Response[v1.TestPlatformConnectionResponse], error) {
	p, err := loadPlatformFull(s.db, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	switch v1.PlatformType(p.Type) {
	case v1.PlatformType_PLATFORM_TYPE_DISCORD:
		if p.DiscordConfig == nil {
			return connect.NewResponse(&v1.TestPlatformConnectionResponse{
				Success: false,
				Message: "discord config not found",
			}), nil
		}
		if err := discordadapter.TestConnection(p.DiscordConfig.BotToken); err != nil {
			return connect.NewResponse(&v1.TestPlatformConnectionResponse{
				Success: false,
				Message: err.Error(),
			}), nil
		}
		return connect.NewResponse(&v1.TestPlatformConnectionResponse{
			Success: true,
			Message: "connected successfully",
		}), nil
	default:
		return connect.NewResponse(&v1.TestPlatformConnectionResponse{
			Success: false,
			Message: "connection test not implemented for this platform type",
		}), nil
	}
}

func (s *PlatformService) getPlatformResponse(id string) (*connect.Response[v1.CreatePlatformResponse], error) {
	p, err := loadPlatformFull(s.db, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.CreatePlatformResponse{
		Platform: platformToProto(p),
	}), nil
}

// savePlatformConfig persists the typed config for a platform create or update request.
type platformConfigSource interface {
	GetDiscord() *v1.DiscordPlatformConfig
	GetReddit() *v1.RedditPlatformConfig
	GetTwitter() *v1.TwitterPlatformConfig
	GetLinkedin() *v1.LinkedInPlatformConfig
	GetGithub() *v1.GitHubPlatformConfig
}

func savePlatformConfig(db *gorm.DB, platformID string, src platformConfigSource) error {
	if c := src.GetDiscord(); c != nil {
		cfg := models.DiscordPlatformConfig{
			PlatformID: platformID,
			BotToken:   c.BotToken,
			GuildIDs:   marshalStringSlice(c.GuildIds),
			ChannelIDs: marshalStringSlice(c.ChannelIds),
		}
		return db.Save(&cfg).Error
	}
	if c := src.GetReddit(); c != nil {
		cfg := models.RedditPlatformConfig{
			PlatformID:   platformID,
			ClientID:     c.ClientId,
			ClientSecret: c.ClientSecret,
			Subreddits:   marshalStringSlice(c.Subreddits),
		}
		return db.Save(&cfg).Error
	}
	if c := src.GetTwitter(); c != nil {
		cfg := models.TwitterPlatformConfig{
			PlatformID:  platformID,
			BearerToken: c.BearerToken,
			Keywords:    marshalStringSlice(c.Keywords),
			Accounts:    marshalStringSlice(c.Accounts),
		}
		return db.Save(&cfg).Error
	}
	if c := src.GetLinkedin(); c != nil {
		cfg := models.LinkedInPlatformConfig{
			PlatformID:  platformID,
			AccessToken: c.AccessToken,
			CompanyIDs:  marshalStringSlice(c.CompanyIds),
		}
		return db.Save(&cfg).Error
	}
	if c := src.GetGithub(); c != nil {
		cfg := models.GitHubPlatformConfig{
			PlatformID:         platformID,
			Token:              c.Token,
			Repositories:       marshalStringSlice(c.Repositories),
			IncludeIssues:      c.IncludeIssues,
			IncludeDiscussions: c.IncludeDiscussions,
		}
		return db.Save(&cfg).Error
	}
	return nil
}

func loadPlatformFull(db *gorm.DB, id string) (*models.Platform, error) {
	var p models.Platform
	if err := db.First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}

	switch v1.PlatformType(p.Type) {
	case v1.PlatformType_PLATFORM_TYPE_DISCORD:
		var cfg models.DiscordPlatformConfig
		if err := db.First(&cfg, "platform_id = ?", id).Error; err == nil {
			p.DiscordConfig = &cfg
		}
	case v1.PlatformType_PLATFORM_TYPE_REDDIT:
		var cfg models.RedditPlatformConfig
		if err := db.First(&cfg, "platform_id = ?", id).Error; err == nil {
			p.RedditConfig = &cfg
		}
	case v1.PlatformType_PLATFORM_TYPE_TWITTER:
		var cfg models.TwitterPlatformConfig
		if err := db.First(&cfg, "platform_id = ?", id).Error; err == nil {
			p.TwitterConfig = &cfg
		}
	case v1.PlatformType_PLATFORM_TYPE_LINKEDIN:
		var cfg models.LinkedInPlatformConfig
		if err := db.First(&cfg, "platform_id = ?", id).Error; err == nil {
			p.LinkedInConfig = &cfg
		}
	case v1.PlatformType_PLATFORM_TYPE_GITHUB:
		var cfg models.GitHubPlatformConfig
		if err := db.First(&cfg, "platform_id = ?", id).Error; err == nil {
			p.GitHubConfig = &cfg
		}
	}

	return &p, nil
}

func platformToProto(p *models.Platform) *v1.Platform {
	proto := &v1.Platform{
		Id:        p.ID,
		Name:      p.Name,
		Type:      v1.PlatformType(p.Type),
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}

	if c := p.DiscordConfig; c != nil {
		proto.Config = &v1.Platform_Discord{
			Discord: &v1.DiscordPlatformConfig{
				BotToken:   c.BotToken,
				GuildIds:   unmarshalStringSlice(c.GuildIDs),
				ChannelIds: unmarshalStringSlice(c.ChannelIDs),
			},
		}
	}
	if c := p.RedditConfig; c != nil {
		proto.Config = &v1.Platform_Reddit{
			Reddit: &v1.RedditPlatformConfig{
				ClientId:     c.ClientID,
				ClientSecret: c.ClientSecret,
				Subreddits:   unmarshalStringSlice(c.Subreddits),
			},
		}
	}
	if c := p.TwitterConfig; c != nil {
		proto.Config = &v1.Platform_Twitter{
			Twitter: &v1.TwitterPlatformConfig{
				BearerToken: c.BearerToken,
				Keywords:    unmarshalStringSlice(c.Keywords),
				Accounts:    unmarshalStringSlice(c.Accounts),
			},
		}
	}
	if c := p.LinkedInConfig; c != nil {
		proto.Config = &v1.Platform_Linkedin{
			Linkedin: &v1.LinkedInPlatformConfig{
				AccessToken: c.AccessToken,
				CompanyIds:  unmarshalStringSlice(c.CompanyIDs),
			},
		}
	}
	if c := p.GitHubConfig; c != nil {
		proto.Config = &v1.Platform_Github{
			Github: &v1.GitHubPlatformConfig{
				Token:              c.Token,
				Repositories:       unmarshalStringSlice(c.Repositories),
				IncludeIssues:      c.IncludeIssues,
				IncludeDiscussions: c.IncludeDiscussions,
			},
		}
	}

	return proto
}

func marshalStringSlice(s []string) string {
	if len(s) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(s)
	return string(b)
}

func unmarshalStringSlice(s string) []string {
	var result []string
	_ = json.Unmarshal([]byte(s), &result)
	return result
}

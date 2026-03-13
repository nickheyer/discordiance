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

var _ discordiancev1connect.MessageServiceHandler = (*MessageService)(nil)

type MessageService struct {
	db *gorm.DB
}

func NewMessageService(db *gorm.DB) *MessageService {
	return &MessageService{db: db}
}

func (s *MessageService) ListMessages(ctx context.Context, req *connect.Request[v1.ListMessagesRequest]) (*connect.Response[v1.ListMessagesResponse], error) {
	slog.Info("rpc: ListMessages", "product_id", req.Msg.ProductId, "page", req.Msg.Page, "page_size", req.Msg.PageSize)

	query := s.db.Model(&models.Message{})

	if req.Msg.ProductId > 0 {
		query = query.Where("product_id = ?", req.Msg.ProductId)
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

	var messages []models.Message
	if err := query.
		Order("timestamp desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&messages).Error; err != nil {
		slog.Error("rpc: ListMessages query failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("listing messages: %w", err))
	}

	slog.Info("rpc: ListMessages result", "returned", len(messages), "total", totalCount)

	resp := &v1.ListMessagesResponse{
		Messages:   make([]*v1.Message, len(messages)),
		TotalCount: int32(totalCount),
	}
	for i, msg := range messages {
		resp.Messages[i] = messageToProto(msg)
	}

	return connect.NewResponse(resp), nil
}

func messageToProto(m models.Message) *v1.Message {
	return &v1.Message{
		Id:           uint64(m.ID),
		CreatedAt:    timestamppb.New(m.CreatedAt),
		ProductId:    uint64(m.ProductID),
		PlatformType: m.PlatformType,
		ExternalId:   m.ExternalID,
		ChannelId:    m.ChannelID,
		AuthorId:     m.AuthorID,
		AuthorName:   m.AuthorName,
		Content:      m.Content,
		Timestamp:    timestamppb.New(m.Timestamp),
		Processed:    m.Processed,
	}
}

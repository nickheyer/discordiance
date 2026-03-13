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
	slog.Info("rpc: ListMessages", "product_id", req.Msg.ProductId, "page", req.Msg.Page, "page_size", req.Msg.PageSize, "processed_filter", req.Msg.ProcessedFilter)

	query := s.db.Model(&models.Message{})

	if req.Msg.ProductId > 0 {
		query = query.Where("product_id = ?", req.Msg.ProductId)
	}

	switch req.Msg.ProcessedFilter {
	case "processed":
		query = query.Where("processed = ?", true)
	case "unprocessed":
		query = query.Where("processed = ?", false)
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

func (s *MessageService) DeleteMessages(ctx context.Context, req *connect.Request[v1.DeleteMessagesRequest]) (*connect.Response[v1.DeleteMessagesResponse], error) {
	slog.Info("rpc: DeleteMessages", "product_id", req.Msg.ProductId, "processed_filter", req.Msg.ProcessedFilter, "ids", len(req.Msg.Ids))

	query := s.db.Model(&models.Message{})

	if len(req.Msg.Ids) > 0 {
		query = query.Where("id IN ?", req.Msg.Ids)
	} else {
		if req.Msg.ProductId > 0 {
			query = query.Where("product_id = ?", req.Msg.ProductId)
		}
		switch req.Msg.ProcessedFilter {
		case "processed":
			query = query.Where("processed = ?", true)
		case "unprocessed":
			query = query.Where("processed = ?", false)
		}
	}

	result := query.Delete(&models.Message{})
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("deleting messages: %w", result.Error))
	}

	slog.Info("rpc: DeleteMessages success", "deleted", result.RowsAffected)

	return connect.NewResponse(&v1.DeleteMessagesResponse{
		DeletedCount: int32(result.RowsAffected),
	}), nil
}

func (s *MessageService) GetMessageStats(ctx context.Context, req *connect.Request[v1.GetMessageStatsRequest]) (*connect.Response[v1.GetMessageStatsResponse], error) {
	slog.Info("rpc: GetMessageStats", "product_id", req.Msg.ProductId)

	query := s.db.Model(&models.Message{})
	if req.Msg.ProductId > 0 {
		query = query.Where("product_id = ?", req.Msg.ProductId)
	}

	var total int64
	query.Count(&total)

	var processed int64
	pq := s.db.Model(&models.Message{}).Where("processed = ?", true)
	if req.Msg.ProductId > 0 {
		pq = pq.Where("product_id = ?", req.Msg.ProductId)
	}
	pq.Count(&processed)

	return connect.NewResponse(&v1.GetMessageStatsResponse{
		Total:       int32(total),
		Processed:   int32(processed),
		Unprocessed: int32(total - processed),
	}), nil
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

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

var _ discordiancev1connect.ProductFileServiceHandler = (*ProductFileService)(nil)

type ProductFileService struct {
	db *gorm.DB
}

func NewProductFileService(db *gorm.DB) *ProductFileService {
	return &ProductFileService{db: db}
}

func (s *ProductFileService) UploadProductFile(ctx context.Context, req *connect.Request[v1.UploadProductFileRequest]) (*connect.Response[v1.UploadProductFileResponse], error) {
	slog.Info("rpc: UploadProductFile", "product_id", req.Msg.ProductId, "filename", req.Msg.Filename, "size", len(req.Msg.Content))

	if req.Msg.Filename == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("filename is required"))
	}

	content := req.Msg.Content
	if int64(len(content)) > models.MaxFileSize {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("file too large: %d bytes (max %d)", len(content), models.MaxFileSize))
	}

	// Check total product file size
	var totalSize int64
	s.db.Model(&models.ProductFile{}).Where("product_id = ?", req.Msg.ProductId).
		Select("COALESCE(SUM(size), 0)").Scan(&totalSize)
	if totalSize+int64(len(content)) > models.MaxProductFileSum {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("product file limit exceeded: %d + %d > %d", totalSize, len(content), models.MaxProductFileSum))
	}

	file := models.ProductFile{
		ProductID: uint(req.Msg.ProductId),
		Filename:  req.Msg.Filename,
		MimeType:  req.Msg.MimeType,
		Size:      int64(len(content)),
		Content:   content,
	}

	if err := s.db.Create(&file).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating product file: %w", err))
	}

	slog.Info("rpc: UploadProductFile success", "id", file.ID, "filename", file.Filename, "size", file.Size)

	return connect.NewResponse(&v1.UploadProductFileResponse{
		File: productFileToProto(file),
	}), nil
}

func (s *ProductFileService) ListProductFiles(ctx context.Context, req *connect.Request[v1.ListProductFilesRequest]) (*connect.Response[v1.ListProductFilesResponse], error) {
	slog.Info("rpc: ListProductFiles", "product_id", req.Msg.ProductId)

	var files []models.ProductFile
	if err := s.db.Select("id, created_at, product_id, filename, mime_type, size").
		Where("product_id = ?", req.Msg.ProductId).
		Order("created_at desc").
		Find(&files).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("listing product files: %w", err))
	}

	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}

	resp := &v1.ListProductFilesResponse{
		Files:     make([]*v1.ProductFile, len(files)),
		TotalSize: totalSize,
	}
	for i, f := range files {
		resp.Files[i] = productFileToProto(f)
	}

	return connect.NewResponse(resp), nil
}

func (s *ProductFileService) DeleteProductFile(ctx context.Context, req *connect.Request[v1.DeleteProductFileRequest]) (*connect.Response[v1.DeleteProductFileResponse], error) {
	slog.Info("rpc: DeleteProductFile", "id", req.Msg.Id)

	if err := s.db.Delete(&models.ProductFile{}, req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("deleting product file: %w", err))
	}

	return connect.NewResponse(&v1.DeleteProductFileResponse{}), nil
}

func (s *ProductFileService) GetProductFileContent(ctx context.Context, req *connect.Request[v1.GetProductFileContentRequest]) (*connect.Response[v1.GetProductFileContentResponse], error) {
	slog.Info("rpc: GetProductFileContent", "id", req.Msg.Id)

	var file models.ProductFile
	if err := s.db.First(&file, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("file %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.GetProductFileContentResponse{
		File:    productFileToProto(file),
		Content: file.Content,
	}), nil
}

func productFileToProto(f models.ProductFile) *v1.ProductFile {
	return &v1.ProductFile{
		Id:        uint64(f.ID),
		CreatedAt: timestamppb.New(f.CreatedAt),
		ProductId: uint64(f.ProductID),
		Filename:  f.Filename,
		MimeType:  f.MimeType,
		Size:      f.Size,
	}
}

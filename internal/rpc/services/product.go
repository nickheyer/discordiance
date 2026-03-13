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

var _ discordiancev1connect.ProductServiceHandler = (*ProductService)(nil)

type ProductService struct {
	db *gorm.DB
}

func NewProductService(db *gorm.DB) *ProductService {
	return &ProductService{db: db}
}

func (s *ProductService) ListProducts(ctx context.Context, req *connect.Request[v1.ListProductsRequest]) (*connect.Response[v1.ListProductsResponse], error) {
	var products []models.Product
	if err := s.db.Find(&products).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("listing products: %w", err))
	}

	slog.Info("rpc: ListProducts", "count", len(products))

	resp := &v1.ListProductsResponse{
		Products: make([]*v1.Product, len(products)),
	}
	for i, p := range products {
		resp.Products[i] = productToProto(p)
	}
	return connect.NewResponse(resp), nil
}

func (s *ProductService) GetProduct(ctx context.Context, req *connect.Request[v1.GetProductRequest]) (*connect.Response[v1.GetProductResponse], error) {
	slog.Info("rpc: GetProduct", "id", req.Msg.Id)

	var product models.Product
	if err := s.db.First(&product, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: GetProduct not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("product %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("rpc: GetProduct found", "id", product.ID, "name", product.Name)
	return connect.NewResponse(&v1.GetProductResponse{
		Product: productToProto(product),
	}), nil
}

func (s *ProductService) CreateProduct(ctx context.Context, req *connect.Request[v1.CreateProductRequest]) (*connect.Response[v1.CreateProductResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	slog.Info("rpc: CreateProduct", "name", req.Msg.Name, "enabled", req.Msg.Enabled)

	product := models.Product{
		Name:        req.Msg.Name,
		Enabled:     req.Msg.Enabled,
		Description: req.Msg.Description,
		RepoURL:     req.Msg.RepoUrl,
		GitHubToken: req.Msg.GithubToken,
	}
	if err := s.db.Create(&product).Error; err != nil {
		slog.Error("rpc: CreateProduct failed", "name", req.Msg.Name, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating product: %w", err))
	}

	slog.Info("rpc: CreateProduct success", "id", product.ID, "name", product.Name)
	return connect.NewResponse(&v1.CreateProductResponse{
		Product: productToProto(product),
	}), nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, req *connect.Request[v1.UpdateProductRequest]) (*connect.Response[v1.UpdateProductResponse], error) {
	slog.Info("rpc: UpdateProduct", "id", req.Msg.Id, "name", req.Msg.Name, "enabled", req.Msg.Enabled)

	var product models.Product
	if err := s.db.First(&product, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: UpdateProduct not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("product %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	product.Name = req.Msg.Name
	product.Enabled = req.Msg.Enabled
	product.Description = req.Msg.Description
	product.RepoURL = req.Msg.RepoUrl
	if req.Msg.GithubToken != "" {
		product.GitHubToken = req.Msg.GithubToken
	}
	if err := s.db.Save(&product).Error; err != nil {
		slog.Error("rpc: UpdateProduct failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating product: %w", err))
	}

	slog.Info("rpc: UpdateProduct success", "id", product.ID, "name", product.Name)
	return connect.NewResponse(&v1.UpdateProductResponse{
		Product: productToProto(product),
	}), nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, req *connect.Request[v1.DeleteProductRequest]) (*connect.Response[v1.DeleteProductResponse], error) {
	slog.Info("rpc: DeleteProduct", "id", req.Msg.Id)

	var product models.Product
	if err := s.db.First(&product, req.Msg.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Warn("rpc: DeleteProduct not found", "id", req.Msg.Id)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("product %d not found", req.Msg.Id))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Cascade delete related records
	slog.Info("rpc: DeleteProduct cascade deleting", "id", req.Msg.Id)
	s.db.Where("product_id = ?", product.ID).Delete(&models.Pipeline{})
	s.db.Where("product_id = ?", product.ID).Delete(&models.Message{})
	s.db.Where("product_id = ?", product.ID).Delete(&models.Insight{})
	s.db.Where("product_id = ?", product.ID).Delete(&models.ProductFile{})
	s.db.Where("product_id = ?", product.ID).Delete(&models.ProductContextCache{})
	s.db.Delete(&product)

	slog.Info("rpc: DeleteProduct success", "id", req.Msg.Id, "name", product.Name)
	return connect.NewResponse(&v1.DeleteProductResponse{}), nil
}

func productToProto(p models.Product) *v1.Product {
	return &v1.Product{
		Id:             uint64(p.ID),
		CreatedAt:      timestamppb.New(p.CreatedAt),
		UpdatedAt:      timestamppb.New(p.UpdatedAt),
		Name:           p.Name,
		Enabled:        p.Enabled,
		Description:    p.Description,
		RepoUrl:        p.RepoURL,
		HasGithubToken: p.GitHubToken != "",
	}
}

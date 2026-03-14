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

type ProductService struct {
	discordiancev1connect.UnimplementedProductServiceHandler
	db *gorm.DB
}

func NewProductService(db *gorm.DB) *ProductService {
	return &ProductService{db: db}
}

func (s *ProductService) CreateProduct(_ context.Context, req *connect.Request[v1.CreateProductRequest]) (*connect.Response[v1.CreateProductResponse], error) {
	product := models.Product{
		ID:          uuid.NewString(),
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
	}

	for _, c := range req.Msg.Contexts {
		product.Contexts = append(product.Contexts, models.ProductContext{
			ID:        uuid.NewString(),
			ProductID: product.ID,
			Type:      int32(c.Type),
			Value:     c.Value,
			Label:     c.Label,
		})
	}

	if err := s.db.Create(&product).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.CreateProductResponse{
		Product: productToProto(&product),
	}), nil
}

func (s *ProductService) GetProduct(_ context.Context, req *connect.Request[v1.GetProductRequest]) (*connect.Response[v1.GetProductResponse], error) {
	var product models.Product
	if err := s.db.Preload("Contexts").First(&product, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&v1.GetProductResponse{
		Product: productToProto(&product),
	}), nil
}

func (s *ProductService) ListProducts(_ context.Context, req *connect.Request[v1.ListProductsRequest]) (*connect.Response[v1.ListProductsResponse], error) {
	pageSize, offset := parsePagination(req.Msg.Pagination)

	var products []models.Product
	var total int64
	s.db.Model(&models.Product{}).Count(&total)
	s.db.Preload("Contexts").Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&products)

	protos := make([]*v1.Product, len(products))
	for i := range products {
		protos[i] = productToProto(&products[i])
	}

	return connect.NewResponse(&v1.ListProductsResponse{
		Products:   protos,
		Pagination: buildPaginationResponse(offset, pageSize, int(total)),
	}), nil
}

func (s *ProductService) UpdateProduct(_ context.Context, req *connect.Request[v1.UpdateProductRequest]) (*connect.Response[v1.UpdateProductResponse], error) {
	var product models.Product
	if err := s.db.Preload("Contexts").First(&product, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	product.Name = req.Msg.Name
	product.Description = req.Msg.Description

	if err := s.db.Save(&product).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.UpdateProductResponse{
		Product: productToProto(&product),
	}), nil
}

func (s *ProductService) DeleteProduct(_ context.Context, req *connect.Request[v1.DeleteProductRequest]) (*connect.Response[v1.DeleteProductResponse], error) {
	if err := s.db.Delete(&models.Product{}, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteProductResponse{}), nil
}

func (s *ProductService) AddProductContext(_ context.Context, req *connect.Request[v1.AddProductContextRequest]) (*connect.Response[v1.AddProductContextResponse], error) {
	var product models.Product
	if err := s.db.First(&product, "id = ?", req.Msg.ProductId).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	ctx := models.ProductContext{
		ID:        uuid.NewString(),
		ProductID: req.Msg.ProductId,
		Type:      int32(req.Msg.Type),
		Value:     req.Msg.Value,
		Label:     req.Msg.Label,
	}

	if err := s.db.Create(&ctx).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.db.Preload("Contexts").First(&product, "id = ?", req.Msg.ProductId)
	return connect.NewResponse(&v1.AddProductContextResponse{
		Product: productToProto(&product),
	}), nil
}

func (s *ProductService) RemoveProductContext(_ context.Context, req *connect.Request[v1.RemoveProductContextRequest]) (*connect.Response[v1.RemoveProductContextResponse], error) {
	if err := s.db.Delete(&models.ProductContext{}, "id = ? AND product_id = ?", req.Msg.ContextId, req.Msg.ProductId).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var product models.Product
	s.db.Preload("Contexts").First(&product, "id = ?", req.Msg.ProductId)
	return connect.NewResponse(&v1.RemoveProductContextResponse{
		Product: productToProto(&product),
	}), nil
}

func productToProto(p *models.Product) *v1.Product {
	contexts := make([]*v1.ProductContext, len(p.Contexts))
	for i, c := range p.Contexts {
		contexts[i] = &v1.ProductContext{
			Id:    c.ID,
			Type:  v1.ProductContextType(c.Type),
			Value: c.Value,
			Label: c.Label,
		}
	}
	return &v1.Product{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Contexts:    contexts,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}

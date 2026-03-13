package services

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/nickheyer/discordiance/internal/engine"
	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
	"github.com/nickheyer/discordiance/pkg/proto/discordiance/v1/discordiancev1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

var _ discordiancev1connect.ProductServiceHandler = (*ProductService)(nil)

type ProductService struct {
	db     *gorm.DB
	engine *engine.Engine
}

func NewProductService(db *gorm.DB, eng *engine.Engine) *ProductService {
	return &ProductService{db: db, engine: eng}
}

func (s *ProductService) ListProducts(ctx context.Context, req *connect.Request[v1.ListProductsRequest]) (*connect.Response[v1.ListProductsResponse], error) {
	var products []models.Product
	if err := s.db.
		Preload("PlatformConfigs").
		Preload("AgentConfig").
		Preload("ReporterConfigs").
		Find(&products).Error; err != nil {
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
	if err := s.db.
		Preload("PlatformConfigs").
		Preload("AgentConfig").
		Preload("ReporterConfigs").
		First(&product, req.Msg.Id).Error; err != nil {
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
		Name:    req.Msg.Name,
		Enabled: req.Msg.Enabled,
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
	if err := s.db.Save(&product).Error; err != nil {
		slog.Error("rpc: UpdateProduct failed", "id", req.Msg.Id, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("updating product: %w", err))
	}

	// Reload with relations
	s.db.Preload("PlatformConfigs").Preload("AgentConfig").Preload("ReporterConfigs").First(&product, product.ID)

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

	// Stop pipeline before deleting (ignore error if not running)
	slog.Info("rpc: DeleteProduct stopping pipeline", "id", req.Msg.Id, "name", product.Name)
	_ = s.engine.StopProduct(ctx, uint(req.Msg.Id))

	// Cascade delete
	slog.Info("rpc: DeleteProduct cascade deleting", "id", req.Msg.Id)
	s.db.Where("product_id = ?", product.ID).Delete(&models.PlatformConfig{})
	s.db.Where("product_id = ?", product.ID).Delete(&models.AgentConfig{})
	s.db.Where("product_id = ?", product.ID).Delete(&models.ReporterConfig{})
	s.db.Where("product_id = ?", product.ID).Delete(&models.Message{})
	s.db.Where("product_id = ?", product.ID).Delete(&models.Issue{})
	s.db.Delete(&product)

	slog.Info("rpc: DeleteProduct success", "id", req.Msg.Id, "name", product.Name)
	return connect.NewResponse(&v1.DeleteProductResponse{}), nil
}

func productToProto(p models.Product) *v1.Product {
	proto := &v1.Product{
		Id:        uint64(p.ID),
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
		Name:      p.Name,
		Enabled:   p.Enabled,
	}

	for _, pc := range p.PlatformConfigs {
		proto.PlatformConfigs = append(proto.PlatformConfigs, platformConfigToProto(pc))
	}

	if p.AgentConfig != nil {
		proto.AgentConfig = agentConfigToProto(*p.AgentConfig)
	}

	for _, rc := range p.ReporterConfigs {
		proto.ReporterConfigs = append(proto.ReporterConfigs, reporterConfigToProto(rc))
	}

	return proto
}

func platformConfigToProto(pc models.PlatformConfig) *v1.PlatformConfig {
	return &v1.PlatformConfig{
		Id:        uint64(pc.ID),
		CreatedAt: timestamppb.New(pc.CreatedAt),
		UpdatedAt: timestamppb.New(pc.UpdatedAt),
		ProductId: uint64(pc.ProductID),
		Type:      pc.Type,
		Enabled:   pc.Enabled,
		Settings:  map[string]string(pc.Settings),
	}
}

func agentConfigToProto(ac models.AgentConfig) *v1.AgentConfig {
	return &v1.AgentConfig{
		Id:           uint64(ac.ID),
		CreatedAt:    timestamppb.New(ac.CreatedAt),
		UpdatedAt:    timestamppb.New(ac.UpdatedAt),
		ProductId:    uint64(ac.ProductID),
		BaseUrl:      ac.BaseURL,
		ApiKey:       ac.APIKey,
		OrgId:        ac.OrgID,
		Model:        ac.Model,
		SystemPrompt: ac.SystemPrompt,
		BatchSize:    int32(ac.BatchSize),
		BatchTimeout: int32(ac.BatchTimeout),
	}
}

func reporterConfigToProto(rc models.ReporterConfig) *v1.ReporterConfig {
	return &v1.ReporterConfig{
		Id:        uint64(rc.ID),
		CreatedAt: timestamppb.New(rc.CreatedAt),
		UpdatedAt: timestamppb.New(rc.UpdatedAt),
		ProductId: uint64(rc.ProductID),
		Type:      rc.Type,
		Enabled:   rc.Enabled,
		Settings:  map[string]string(rc.Settings),
	}
}

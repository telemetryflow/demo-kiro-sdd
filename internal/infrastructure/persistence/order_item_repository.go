// Package persistence implements OrderItem repository with GORM.
//
// TelemetryFlow Order Service - AI-Powered Observability & Incident Response Management (IRM) Platform
// Copyright (c) 2024-2026 Telemetri Data Indonesia. All rights reserved.
// Open Source Software built by Telemetri Data Indonesia.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/telemetryflow/order-service/internal/domain/entity"
	"github.com/telemetryflow/order-service/internal/domain/repository"
	"gorm.io/gorm"
)

// orderItemRepository implements repository.OrderItemRepository using GORM
type orderItemRepository struct {
	db *gorm.DB
}

// NewOrderItemRepository creates a new OrderItem repository
func NewOrderItemRepository(db *gorm.DB) repository.OrderItemRepository {
	return &orderItemRepository{
		db: db,
	}
}

// Create creates a new orderItem
func (r *orderItemRepository) Create(ctx context.Context, orderItem *entity.OrderItem) error {
	return r.db.WithContext(ctx).Create(orderItem).Error
}

// FindByID retrieves an orderItem by ID
func (r *orderItemRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.OrderItem, error) {
	var orderItem entity.OrderItem
	err := r.db.WithContext(ctx).First(&orderItem, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("orderItem not found")
		}
		return nil, err
	}
	return &orderItem, nil
}

// FindAll retrieves all orderItems with pagination
func (r *orderItemRepository) FindAll(ctx context.Context, offset, limit int) ([]entity.OrderItem, int64, error) {
	var orderItems []entity.OrderItem
	var total int64

	// Count total records
	if err := r.db.WithContext(ctx).Model(&entity.OrderItem{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated records
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orderItems).Error; err != nil {
		return nil, 0, err
	}

	return orderItems, total, nil
}

// Update updates an orderItem
func (r *orderItemRepository) Update(ctx context.Context, orderItem *entity.OrderItem) error {
	return r.db.WithContext(ctx).Save(orderItem).Error
}

// Delete soft-deletes an orderItem by ID
func (r *orderItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.OrderItem{}, "id = ?", id).Error
}

// HardDelete permanently deletes an orderItem by ID
func (r *orderItemRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&entity.OrderItem{}, "id = ?", id).Error
}

// FindByOrderID finds all items for an order
func (r *orderItemRepository) FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]entity.OrderItem, error) {
	var items []entity.OrderItem
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// FindByProductID finds all items for a product
func (r *orderItemRepository) FindByProductID(ctx context.Context, productID uuid.UUID) ([]entity.OrderItem, error) {
	var items []entity.OrderItem
	err := r.db.WithContext(ctx).
		Where("product_id = ?", productID).
		Order("created_at DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// CreateBatch creates multiple orderItems in a single transaction
func (r *orderItemRepository) CreateBatch(ctx context.Context, items []entity.OrderItem) error {
	return r.db.WithContext(ctx).Create(&items).Error
}

// DeleteByOrderID deletes all items for an order
func (r *orderItemRepository) DeleteByOrderID(ctx context.Context, orderID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.OrderItem{}, "order_id = ?", orderID).Error
}

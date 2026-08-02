// Package repository defines the OrderItem repository interface.
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

package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/telemetryflow/order-service/internal/domain/entity"
)

// OrderItemRepository defines the repository interface for OrderItem
type OrderItemRepository interface {
	// Create creates a new orderItem
	Create(ctx context.Context, e *entity.OrderItem) error

	// FindByID finds a orderItem by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entity.OrderItem, error)

	// FindAll finds all orderItems with pagination
	FindAll(ctx context.Context, offset, limit int) ([]entity.OrderItem, int64, error)

	// Update updates an existing orderItem
	Update(ctx context.Context, e *entity.OrderItem) error

	// Delete soft-deletes a orderItem by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// HardDelete permanently deletes a orderItem
	HardDelete(ctx context.Context, id uuid.UUID) error

	// FindByOrderID finds all items for an order
	FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]entity.OrderItem, error)

	// FindByProductID finds all items for a product
	FindByProductID(ctx context.Context, productID uuid.UUID) ([]entity.OrderItem, error)

	// CreateBatch creates multiple orderItems in a single transaction
	CreateBatch(ctx context.Context, items []entity.OrderItem) error

	// DeleteByOrderID deletes all items for an order
	DeleteByOrderID(ctx context.Context, orderID uuid.UUID) error
}

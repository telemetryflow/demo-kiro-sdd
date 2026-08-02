// Package dto contains DTOs for OrderItem endpoints.
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

package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/telemetryflow/order-service/internal/domain/entity"
)

// OrderItemResponse represents the orderItem API response
type OrderItemResponse struct {
	ID        uuid.UUID `json:"id"`
	OrderID   uuid.UUID `json:"order_id"`
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromOrderItem converts entity to response DTO
func FromOrderItem(e *entity.OrderItem) OrderItemResponse {
	return OrderItemResponse{
		ID:        e.ID,
		OrderID:   e.OrderID,
		ProductID: e.ProductID,
		Quantity:  e.Quantity,
		Price:     e.Price,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// FromOrderItems converts entities to response DTOs
func FromOrderItems(entities []entity.OrderItem) []OrderItemResponse {
	responses := make([]OrderItemResponse, len(entities))
	for i, e := range entities {
		responses[i] = FromOrderItem(&e)
	}
	return responses
}

// CreateOrderItemRequest represents the create orderItem request.
// order_id is NOT in the body — it comes from the URL path
// (POST /orders/{order_id}/items).
type CreateOrderItemRequest struct {
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity" validate:"required"`
	Price     float64   `json:"price" validate:"required"`
}

// UpdateOrderItemRequest represents the update orderItem request.
// order_id is NOT in the body — it comes from the URL path
// (PUT /orders/{order_id}/items/{id}).
type UpdateOrderItemRequest struct {
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity" validate:"required"`
	Price     float64   `json:"price" validate:"required"`
}

// OrderItemToResponse converts entity pointer to response DTO pointer
func OrderItemToResponse(e *entity.OrderItem) *OrderItemResponse {
	if e == nil {
		return nil
	}
	resp := FromOrderItem(e)
	return &resp
}

// OrderItemListResponse represents the list orderItem API response
type OrderItemListResponse struct {
	Data   []*OrderItemResponse `json:"data"`
	Total  int                  `json:"total"`
	Offset int                  `json:"offset"`
	Limit  int                  `json:"limit"`
}

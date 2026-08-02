// Package command contains CQRS commands for OrderItem.
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

package command

import (
	"github.com/google/uuid"
	"github.com/telemetryflow/order-service/internal/domain/entity"
)

// CreateOrderItemCommand represents the create orderItem command
type CreateOrderItemCommand struct {
	OrderID   uuid.UUID `json:"order_id" validate:"required"`
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity" validate:"required"`
	Price     float64   `json:"price" validate:"required"`
}

// Validate validates the create command
func (c *CreateOrderItemCommand) Validate() error {
	// Add validation logic
	return nil
}

// ToEntity converts the command to an entity
func (c *CreateOrderItemCommand) ToEntity() *entity.OrderItem {
	return entity.NewOrderItem(c.OrderID, c.ProductID, c.Quantity, c.Price)
}

// UpdateOrderItemCommand represents the update orderItem command
type UpdateOrderItemCommand struct {
	ID        uuid.UUID `json:"id" validate:"required"`
	OrderID   uuid.UUID `json:"order_id" validate:"required"`
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity" validate:"required"`
	Price     float64   `json:"price" validate:"required"`
}

// Validate validates the update command
func (c *UpdateOrderItemCommand) Validate() error {
	if c.ID == uuid.Nil {
		return ErrInvalidID
	}
	return nil
}

// ToEntity converts the command to an entity
func (c *UpdateOrderItemCommand) ToEntity() *entity.OrderItem {
	e := entity.NewOrderItem(c.OrderID, c.ProductID, c.Quantity, c.Price)
	e.ID = c.ID
	return e
}

// DeleteOrderItemCommand represents the delete orderItem command
type DeleteOrderItemCommand struct {
	ID uuid.UUID `json:"id" validate:"required"`
}

// Validate validates the delete command
func (c *DeleteOrderItemCommand) Validate() error {
	if c.ID == uuid.Nil {
		return ErrInvalidID
	}
	return nil
}

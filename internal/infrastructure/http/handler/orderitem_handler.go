// Package handler provides HTTP handlers for OrderItem endpoints.
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

package handler

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/telemetryflow/order-service/internal/application/command"
	"github.com/telemetryflow/order-service/internal/application/dto"
	"github.com/telemetryflow/order-service/internal/application/handler"
	"github.com/telemetryflow/order-service/internal/application/query"
	"github.com/telemetryflow/order-service/pkg/response"
)

// OrderitemHandler handles orderitem HTTP requests
type OrderitemHandler struct {
	commandHandler *handler.OrderitemCommandHandler
	queryHandler   *handler.OrderitemQueryHandler
}

// NewOrderitemHandler creates a new orderitem handler
func NewOrderitemHandler(
	cmdHandler *handler.OrderitemCommandHandler,
	qryHandler *handler.OrderitemQueryHandler,
) *OrderitemHandler {
	return &OrderitemHandler{
		commandHandler: cmdHandler,
		queryHandler:   qryHandler,
	}
}

// RegisterRoutes registers orderitem routes as a nested sub-resource of orders:
//   POST   /orders/:order_id/items
//   GET    /orders/:order_id/items
//   GET    /orders/:order_id/items/:id
//   PUT    /orders/:order_id/items/:id
//   DELETE /orders/:order_id/items/:id
//
// order_id is always taken from the URL path, never from the request body.
func (h *OrderitemHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/orders/:order_id/items", h.Create)
	g.GET("/orders/:order_id/items", h.List)
	g.GET("/orders/:order_id/items/:id", h.GetByID)
	g.PUT("/orders/:order_id/items/:id", h.Update)
	g.DELETE("/orders/:order_id/items/:id", h.Delete)
}

// parseOrderID extracts and validates the :order_id path parameter.
func parseOrderID(c echo.Context) (uuid.UUID, error) {
	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		return uuid.Nil, err
	}
	return orderID, nil
}

// Create handles POST /orders/:order_id/items
func (h *OrderitemHandler) Create(c echo.Context) error {
	orderID, err := parseOrderID(c)
	if err != nil {
		return response.BadRequest(c, "Invalid order_id format")
	}

	var req dto.CreateOrderitemRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}

	cmd := &command.CreateOrderitemCommand{
		OrderID:   orderID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Price:     req.Price,
	}

	if err := h.commandHandler.HandleOrderitemCreate(c.Request().Context(), cmd); err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Created(c, nil, "Order item created successfully")
}

// List handles GET /orders/:order_id/items
func (h *OrderitemHandler) List(c echo.Context) error {
	orderID, err := parseOrderID(c)
	if err != nil {
		return response.BadRequest(c, "Invalid order_id format")
	}

	result, err := h.queryHandler.HandleOrderitemGetByOrderID(c.Request().Context(), orderID)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, result, "")
}

// GetByID handles GET /orders/:order_id/items/:id
func (h *OrderitemHandler) GetByID(c echo.Context) error {
	orderID, err := parseOrderID(c)
	if err != nil {
		return response.BadRequest(c, "Invalid order_id format")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID format")
	}

	q := &query.GetOrderitemByIDQuery{ID: id}
	result, err := h.queryHandler.HandleOrderitemGetByID(c.Request().Context(), q)
	if err != nil {
		return response.NotFound(c, "Order item not found")
	}

	// Ownership check: the item must belong to the order in the path.
	if result.OrderID != orderID {
		return response.NotFound(c, "Order item not found")
	}

	return response.Success(c, result, "")
}

// Update handles PUT /orders/:order_id/items/:id
func (h *OrderitemHandler) Update(c echo.Context) error {
	orderID, err := parseOrderID(c)
	if err != nil {
		return response.BadRequest(c, "Invalid order_id format")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID format")
	}

	var req dto.UpdateOrderitemRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}

	cmd := &command.UpdateOrderitemCommand{
		ID:        id,
		OrderID:   orderID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Price:     req.Price,
	}

	if err := h.commandHandler.HandleOrderitemUpdate(c.Request().Context(), cmd); err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, nil, "Order item updated successfully")
}

// Delete handles DELETE /orders/:order_id/items/:id
func (h *OrderitemHandler) Delete(c echo.Context) error {
	orderID, err := parseOrderID(c)
	if err != nil {
		return response.BadRequest(c, "Invalid order_id format")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID format")
	}

	// Ownership check before delete: verify the item belongs to this order.
	existing, err := h.queryHandler.HandleOrderitemGetByID(c.Request().Context(), &query.GetOrderitemByIDQuery{ID: id})
	if err != nil || existing == nil || existing.OrderID != orderID {
		return response.NotFound(c, "Order item not found")
	}

	cmd := &command.DeleteOrderitemCommand{ID: id}
	if err := h.commandHandler.HandleOrderitemDelete(c.Request().Context(), cmd); err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.NoContent(c)
}

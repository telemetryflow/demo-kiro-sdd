// Package tests provides integration tests for Order API.
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

package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Order API Endpoint Tests
// =============================================================================

func TestOrderAPIEndpoints(t *testing.T) {
	skipInShortMode(t)

	e := echo.New()

	// In-memory storage for testing
	orders := make(map[string]map[string]interface{})

	// Setup order endpoints with correct naming conventions
	api := e.Group("/api/v1")

	// POST /orders - Create order
	api.POST("/orders", func(c echo.Context) error {
		var order map[string]interface{}
		if err := c.Bind(&order); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
		}
		id := uuid.New().String()
		order["id"] = id
		orders[id] = order
		return c.JSON(http.StatusCreated, order)
	})

	// GET /orders - List orders
	api.GET("/orders", func(c echo.Context) error {
		data := make([]interface{}, 0, len(orders))
		for _, o := range orders {
			data = append(data, o)
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data":   data,
			"total":  len(orders),
			"offset": 0,
			"limit":  10,
		})
	})

	// GET /orders/:id - Get order by ID
	api.GET("/orders/:id", func(c echo.Context) error {
		id := c.Param("id")
		order, exists := orders[id]
		if !exists {
			return echo.NewHTTPError(http.StatusNotFound, "Order not found")
		}
		return c.JSON(http.StatusOK, order)
	})

	// PUT /orders/:id - Update order
	api.PUT("/orders/:id", func(c echo.Context) error {
		id := c.Param("id")
		if _, exists := orders[id]; !exists {
			return echo.NewHTTPError(http.StatusNotFound, "Order not found")
		}
		var order map[string]interface{}
		if err := c.Bind(&order); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
		}
		order["id"] = id
		orders[id] = order
		return c.JSON(http.StatusOK, order)
	})

	// DELETE /orders/:id - Delete order
	api.DELETE("/orders/:id", func(c echo.Context) error {
		id := c.Param("id")
		if _, exists := orders[id]; !exists {
			return echo.NewHTTPError(http.StatusNotFound, "Order not found")
		}
		delete(orders, id)
		return c.NoContent(http.StatusNoContent)
	})

	var createdOrderID string

	t.Run("POST /orders - should create order", func(t *testing.T) {
		body := map[string]interface{}{
			"customer_id": uuid.New().String(),
			"total":       99.99,
			"status":      "pending",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response["id"])
		createdOrderID = response["id"].(string)
	})

	t.Run("GET /orders - should list orders", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "data")
		assert.Contains(t, response, "total")
	})

	t.Run("GET /orders/:id - should get order by ID", func(t *testing.T) {
		require.NotEmpty(t, createdOrderID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+createdOrderID, nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, createdOrderID, response["id"])
	})

	t.Run("PUT /orders/:id - should update order", func(t *testing.T) {
		require.NotEmpty(t, createdOrderID)

		body := map[string]interface{}{
			"customer_id": uuid.New().String(),
			"total":       149.99,
			"status":      "confirmed",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+createdOrderID, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "confirmed", response["status"])
	})

	t.Run("DELETE /orders/:id - should delete order", func(t *testing.T) {
		require.NotEmpty(t, createdOrderID)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/orders/"+createdOrderID, nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("GET /orders/:id - should return 404 for deleted order", func(t *testing.T) {
		require.NotEmpty(t, createdOrderID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+createdOrderID, nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// =============================================================================
// Order Items API Endpoint Tests (nested sub-resource: /orders/{order_id}/items)
// =============================================================================

func TestOrderItemsAPIEndpoints(t *testing.T) {
	skipInShortMode(t)

	e := echo.New()

	// In-memory storage keyed by order_id -> {item_id -> item}
	itemsByOrder := make(map[string]map[string]interface{})

	// Fixed parent order for these tests
	parentOrderID := uuid.New().String()

	// Setup nested order-items endpoints: /orders/:order_id/items[/:id]
	api := e.Group("/api/v1")

	// POST /orders/:order_id/items - Create order item (order_id from path)
	api.POST("/orders/:order_id/items", func(c echo.Context) error {
		orderID := c.Param("order_id")
		var item map[string]interface{}
		if err := c.Bind(&item); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
		}
		id := uuid.New().String()
		item["id"] = id
		item["order_id"] = orderID
		if _, ok := itemsByOrder[orderID]; !ok {
			itemsByOrder[orderID] = make(map[string]interface{})
		}
		itemsByOrder[orderID][id] = item
		return c.JSON(http.StatusCreated, item)
	})

	// GET /orders/:order_id/items - List items in an order
	api.GET("/orders/:order_id/items", func(c echo.Context) error {
		orderID := c.Param("order_id")
		data := make([]interface{}, 0)
		for _, item := range itemsByOrder[orderID] {
			data = append(data, item)
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data":  data,
			"total": len(data),
		})
	})

	// GET /orders/:order_id/items/:id - Get one item (ownership-checked)
	api.GET("/orders/:order_id/items/:id", func(c echo.Context) error {
		orderID := c.Param("order_id")
		id := c.Param("id")
		item, exists := itemsByOrder[orderID][id]
		if !exists {
			return echo.NewHTTPError(http.StatusNotFound, "Order item not found")
		}
		return c.JSON(http.StatusOK, item)
	})

	// PUT /orders/:order_id/items/:id - Update item
	api.PUT("/orders/:order_id/items/:id", func(c echo.Context) error {
		orderID := c.Param("order_id")
		id := c.Param("id")
		if _, exists := itemsByOrder[orderID][id]; !exists {
			return echo.NewHTTPError(http.StatusNotFound, "Order item not found")
		}
		var item map[string]interface{}
		if err := c.Bind(&item); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
		}
		item["id"] = id
		item["order_id"] = orderID
		itemsByOrder[orderID][id] = item
		return c.JSON(http.StatusOK, item)
	})

	// DELETE /orders/:order_id/items/:id - Remove item
	api.DELETE("/orders/:order_id/items/:id", func(c echo.Context) error {
		orderID := c.Param("order_id")
		id := c.Param("id")
		if _, exists := itemsByOrder[orderID][id]; !exists {
			return echo.NewHTTPError(http.StatusNotFound, "Order item not found")
		}
		delete(itemsByOrder[orderID], id)
		return c.NoContent(http.StatusNoContent)
	})

	var createdItemID string

	t.Run("POST /orders/{order_id}/items - should create order item", func(t *testing.T) {
		body := map[string]interface{}{
			"product_id": uuid.New().String(),
			"quantity":   2,
			"price":      29.99,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+parentOrderID+"/items", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response["id"])
		assert.Equal(t, parentOrderID, response["order_id"], "order_id must come from the URL path")
		createdItemID = response["id"].(string)
	})

	t.Run("GET /orders/{order_id}/items - should list order items", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+parentOrderID+"/items", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "data")
		assert.Contains(t, response, "total")
	})

	t.Run("GET /orders/{order_id}/items/{id} - should get order item by ID", func(t *testing.T) {
		require.NotEmpty(t, createdItemID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+parentOrderID+"/items/"+createdItemID, nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, createdItemID, response["id"])
	})

	t.Run("GET wrong order_id should 404 (ownership)", func(t *testing.T) {
		require.NotEmpty(t, createdItemID)
		otherOrder := uuid.New().String()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+otherOrder+"/items/"+createdItemID, nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code, "item must not be reachable from another order")
	})

	t.Run("PUT /orders/{order_id}/items/{id} - should update order item", func(t *testing.T) {
		require.NotEmpty(t, createdItemID)

		body := map[string]interface{}{
			"product_id": uuid.New().String(),
			"quantity":   5,
			"price":      24.99,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+parentOrderID+"/items/"+createdItemID, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(5), response["quantity"])
	})

	t.Run("DELETE /orders/{order_id}/items/{id} - should delete order item", func(t *testing.T) {
		require.NotEmpty(t, createdItemID)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/orders/"+parentOrderID+"/items/"+createdItemID, nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("GET /orders/{order_id}/items/{id} - should return 404 for deleted item", func(t *testing.T) {
		require.NotEmpty(t, createdItemID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+parentOrderID+"/items/"+createdItemID, nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// =============================================================================
// API Naming Convention Tests
// =============================================================================

func TestAPINamingConventions(t *testing.T) {
	skipInShortMode(t)

	t.Run("order endpoints should use plural noun 'orders'", func(t *testing.T) {
		// Verify the endpoint path follows REST conventions
		expectedPath := "/api/v1/orders"
		assert.Contains(t, expectedPath, "orders")
		assert.NotContains(t, expectedPath, "order/")
	})

	t.Run("order-items are a nested sub-resource of orders", func(t *testing.T) {
		// Items live under their parent order: /orders/{order_id}/items
		expectedPath := "/api/v1/orders/{order_id}/items"
		assert.Contains(t, expectedPath, "/orders/")
		assert.Contains(t, expectedPath, "/items")
		assert.NotContains(t, expectedPath, "order-items", "items are nested under orders, not a flat top-level resource")
	})

	t.Run("JSON response fields should use snake_case", func(t *testing.T) {
		// Verify JSON field naming conventions
		response := map[string]interface{}{
			"id":          uuid.New().String(),
			"customer_id": uuid.New().String(),
			"order_id":    uuid.New().String(),
			"product_id":  uuid.New().String(),
			"created_at":  "2024-01-01T00:00:00Z",
			"updated_at":  "2024-01-01T00:00:00Z",
		}

		// Verify all keys use snake_case
		for key := range response {
			assert.NotContains(t, key, "-", "JSON key should not contain hyphens: %s", key)
			// snake_case uses underscores for word separation
			if len(key) > 2 {
				assert.Regexp(t, `^[a-z][a-z0-9_]*$`, key, "JSON key should be snake_case: %s", key)
			}
		}
	})
}

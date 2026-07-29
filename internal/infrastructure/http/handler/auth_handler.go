// Package handler provides HTTP handlers for Auth endpoints.
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
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/telemetryflow/order-service/internal/infrastructure/config"
	"github.com/telemetryflow/order-service/internal/infrastructure/http/middleware"
	"github.com/telemetryflow/order-service/pkg/response"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	jwtConfig config.JWTConfig
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(jwtConfig config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		jwtConfig: jwtConfig,
	}
}

// TokenRequest represents the request body for token generation
type TokenRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin user viewer"`
}

// TokenResponse represents the token generation response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// RegisterRoutes registers auth routes on the given group
func (h *AuthHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/auth/token", h.CreateToken)
}

// CreateToken handles POST /api/v1/auth/token
func (h *AuthHandler) CreateToken(c echo.Context) error {
	var req TokenRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}

	// Auto-generate user_id
	userID := uuid.New().String()

	now := time.Now()
	expiration := now.Add(h.jwtConfig.Expiration)

	claims := &middleware.JWTClaims{
		UserID: userID,
		Email:  req.Email,
		Role:   req.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "order-service",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtConfig.Secret))
	if err != nil {
		return response.InternalError(c, "Failed to generate token")
	}

	return response.Success(c, TokenResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.jwtConfig.Expiration.Seconds()),
	}, "Token generated successfully")
}

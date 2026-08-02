// Package main provides the Order Service RESTful API server.
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

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/telemetryflow/order-service/internal/domain/entity"
	"github.com/telemetryflow/order-service/internal/infrastructure/config"
	"github.com/telemetryflow/order-service/internal/infrastructure/http"
	"github.com/telemetryflow/order-service/internal/infrastructure/persistence"
	"github.com/telemetryflow/order-service/telemetry"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	// Initialize TelemetryFlow
	if err := telemetry.Init(); err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer telemetry.Shutdown()

	// Initialize database
	db, err := persistence.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		sqlDB, err := db.DB()
		if err != nil {
			log.Printf("Failed to get underlying sql.DB: %v", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			log.Printf("Failed to close database connection: %v", err)
		}
	}()

	// Auto-migrate ensures the orders and order_items tables exist (self-healing
	// when the DB is reset). GORM only creates missing tables/columns/indexes;
	// it never drops data. Disable for environments that manage schema externally.
	if err := persistence.AutoMigrate(db, &entity.Order{}, &entity.OrderItem{}); err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}
	log.Println("Database schema verified (orders, order_items)")

	// Create HTTP server
	server := http.NewServer(cfg, db)

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Printf("Order-Service API v1.4.4 started on port %s", cfg.Server.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Copyright Dose de Telemetria GmbH
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"net/http"

	"github.com/dosedetelemetria/projeto-otel-na-pratica/api"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/config"
	grpchandler "github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/handler/grpc"
	planhttp "github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/handler/http"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/store"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/store/memory"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"go.opentelemetry.io/otel"
)

type Plan struct {
	Handler     *planhttp.PlanHandler
	GRPCHandler api.PlanServiceServer
	Store       store.Plan
}

func NewPlan(ctx context.Context, _ *config.Plans) *Plan {
	ctx, span := otel.Tracer("plan").Start(ctx, "NewPlan")
	defer span.End()
	store := memory.NewPlanStore()
	return &Plan{
		Handler:     planhttp.NewPlanHandler(store),
		GRPCHandler: grpchandler.NewPlanServer(store),
		Store:       store,
	}
}

func (a *Plan) RegisterRoutes(mux *http.ServeMux, grpcSrv *grpc.Server) {
	mux.Handle("GET /plans", otelhttp.NewHandler(http.HandlerFunc(a.Handler.List), "GET /plans"))
	mux.Handle("POST /plans", otelhttp.NewHandler(http.HandlerFunc(a.Handler.List), "POST /plans"))
	mux.Handle("GET /plans/{id}", otelhttp.NewHandler(http.HandlerFunc(a.Handler.List), "GET /plans/{id}"))
	mux.Handle("PUT /plans/{id}", otelhttp.NewHandler(http.HandlerFunc(a.Handler.List), "PUT /plans/{id}"))
	mux.Handle("DELETE /plans/{id}", otelhttp.NewHandler(http.HandlerFunc(a.Handler.List), "DELETE /plans/{id}"))

	api.RegisterPlanServiceServer(grpcSrv, a.GRPCHandler)
}

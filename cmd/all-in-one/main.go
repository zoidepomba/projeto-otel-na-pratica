// Copyright Dose de Telemetria GmbH
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"

	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/app"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/config"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/telemetry"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

func main() {
	configFlag := flag.String("config", "", "path to the config file")
	otelConfigFlag := flag.String("otel", "otel.yaml", "path to the config file")
	flag.Parse()

	closer, err := telemetry.Setup(context.Background(), *otelConfigFlag)
	if err != nil {
		panic(err)
	}
	defer closer(context.Background())

	ctx, span := otel.Tracer("all-in-one").Start(context.Background(), "main")

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("all-in-one", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	logger := zap.New(core)

	logger.Info("starting the all-in-one service")
	span.AddEvent("starting the all-in-one service")
	c, err := config.LoadConfig(*configFlag)
	if err != nil {
		span.AddEvent("failed to load the config", trace.WithAttributes(attribute.String("error", err.Error())))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error("failed to load the config", zap.Error(err))
	}

	mux := http.NewServeMux()

	// starts the gRPC server
	lis, err := net.Listen("tcp", c.Server.Endpoint.GRPC)
	if err != nil {
		span.AddEvent("failed to load the config", trace.WithAttributes(attribute.String("error", err.Error())))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Fatal("failed to listen", zap.Error(err))
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	{
		logger.Info("Starting the user service")
		span.AddEvent("starting the user service")
		a := app.NewUser(ctx, &c.Users)
		a.RegisterRoutes(mux)
	}

	{
		logger.Info("Starting the plan service")
		span.AddEvent("starting the plan service")
		a := app.NewPlan(ctx, &c.Plans)
		a.RegisterRoutes(mux, grpcServer)
	}

	{
		logger.Info("Starting the payment service")
		span.AddEvent("starting the payment service")
		a, err := app.NewPayment(ctx, &c.Payments)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.Error("failed to create the payment service", zap.Error(err))
		}
		a.RegisterRoutes(mux)
		defer func() {
			err = a.Shutdown()
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				logger.Error("failed to shutdown the payment service", zap.Error(err))
			}
		}()
	}

	{
		logger.Info("Starting the subscription service")
		span.AddEvent("starting the subscription service")
		a := app.NewSubscription(ctx, &c.Subscriptions)
		a.RegisterRoutes(mux)
	}

	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.Error("failed to serve", zap.Error(err))
		}
	}()

	span.End()
	err = http.ListenAndServe(c.Server.Endpoint.HTTP, mux)
	if err != nil && err != http.ErrServerClosed {
		span.RecordError(err)
		logger.Error("failed to serve", zap.Error(err))
	}
	logger.Info("shutting down the all-in-one service")
}

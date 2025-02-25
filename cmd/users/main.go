// Copyright Dose de Telemetria GmbH
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/app"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/config"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/telemetry"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/attribute"
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

	ctx, span := otel.Tracer("users").Start(context.Background(), "main")

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("users", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	logger := zap.New(core)

	logger.Info("starting the users service")
	span.AddEvent("starting the users service")
	c, err := config.LoadConfig(*configFlag)
	if err != nil {
		span.AddEvent("failed to load the config", trace.WithAttributes(attribute.String("error", err.Error())))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error("failed to load the config", zap.Error(err))
	}
	
	logger.Info("starting the users service")
	span.AddEvent("starting the users service")
	a := app.NewUser(ctx, &c.Users)
	a.RegisterRoutes(http.DefaultServeMux)
	
	err = http.ListenAndServe(c.Server.Endpoint.HTTP, http.DefaultServeMux)
	if err != nil {
		span.AddEvent("failed to start the server", trace.WithAttributes(attribute.String("error", err.Error())))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error("failed to start the server", zap.Error(err))
	}
}

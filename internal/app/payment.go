// Copyright Dose de Telemetria GmbH
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"net/http"
	"os"

	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/config"
	planhttp "github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/handler/http"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/model"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/store"
	storegorm "github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/store/gorm"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

)

type Payment struct {
	Handler  *planhttp.PaymentHandler
	Store    store.Payment
	natsConn *nats.Conn
	cctx     jetstream.ConsumeContext
}

func NewPayment(ctx context.Context, cfg *config.Payments) (*Payment, error) {

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("payment", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	logger := zap.New(core)

	ctx, span := otel.Tracer("payment").Start(ctx, "NewPayment")
	
	defer span.End()

	traceID := span.SpanContext().TraceID().String()

	db, err := gorm.Open(sqlite.Open(cfg.SQLLite.DSN))
	if err != nil {
		logger.Error("failed to open the database", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to open the database", trace.WithAttributes(attribute.String("error", err.Error())))
		return nil, err
	}

	logger.Info("opened the database", zap.String("dsn", cfg.SQLLite.DSN), zap.String("trace_id", traceID) )
	span.AddEvent("opened the database", trace.WithAttributes(attribute.String("dsn", cfg.SQLLite.DSN)))
	err = db.AutoMigrate(&model.Payment{})
	if err != nil {
		logger.Error("failed to migrate the database", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to migrate the database", trace.WithAttributes(attribute.String("error", err.Error())))
		return nil, err
	}

	logger.Info("migrated the database", zap.String("trace_id", traceID))
	span.AddEvent("migrated the database")
	nc, err := nats.Connect(cfg.NATS.Endpoint)
	if err != nil {
		logger.Error("failed to connect to nats", zap.Error(err), zap.String("trace_id", traceID))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to connect to nats", trace.WithAttributes(attribute.String("error", err.Error())))
		return nil, err
	}

	logger.Info("connected to nats", zap.String("url", cfg.NATS.Endpoint), zap.String("trace_id", traceID))
	span.AddEvent("connected to nats", trace.WithAttributes(attribute.String("url", cfg.NATS.Endpoint)))
	js, err := jetstream.New(nc)
	if err != nil {
		logger.Error("failed to create jetstream", zap.Error(err), zap.String("trace_id", traceID))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to create jetstream", trace.WithAttributes(attribute.String("error", err.Error())))
		return nil, err
	}

	logger.Info("created jetstream", zap.String("trace_id", traceID))
	span.AddEvent("created jetstream")
	stream, err := js.Stream(ctx, cfg.NATS.Stream)
	if err != nil {
		logger.Error("failed to create jetstream stream traceID", zap.Error(err), zap.String("trace_id", traceID))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to create jetstream stream", trace.WithAttributes(attribute.String("error", err.Error())))
		return nil, err
	}

	// this is only relevant for the consumer
	logger.Info("created jetstream stream", zap.String("stream", cfg.NATS.Stream))
	span.AddEvent("created jetstream stream", trace.WithAttributes(attribute.String("stream", cfg.NATS.Stream)))
	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          cfg.NATS.ConsumerName,
		Durable:       cfg.NATS.ConsumerName,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		logger.Error("failed to create jetstream consumer", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to create jetstream consumer", trace.WithAttributes(attribute.String("error", err.Error())))
		return nil, err
	}

	logger.Info("created jetstream consumer", zap.String("consumer", cfg.NATS.ConsumerName))
	span.AddEvent("created jetstream consumer", trace.WithAttributes(attribute.String("consumer", cfg.NATS.ConsumerName)))
	store := storegorm.NewPaymentStore(db)
	pmt := &Payment{
		Handler:  planhttp.NewPaymentHandler(store, js, cfg.NATS.Subject, cfg.SubscriptionsEndpoint),
		Store:    store,
		natsConn: nc,
	}

	logger.Info("created payment handler")
	span.AddEvent("created payment handler")
	pmt.cctx, err = cons.Consume(pmt.Handler.OnMessage)
	if err != nil {
		logger.Error("failed to consume messages", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to consume messages", trace.WithAttributes(attribute.String("error", err.Error())))
		return nil, err
	}
	logger.Info("consumed messages")
	return pmt, nil
}

func (a *Payment) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("GET /payments", otelhttp.NewHandler(http.HandlerFunc(a.Handler.List), "GET /payments"))
	mux.Handle("POST /payments", otelhttp.NewHandler(http.HandlerFunc(a.Handler.Create), "POST /payments"))
	mux.Handle("GET /payments/{id}", otelhttp.NewHandler(http.HandlerFunc(a.Handler.Get), "GET /payments/{id}"))	
	mux.Handle("PUT /payments/{id}", otelhttp.NewHandler(http.HandlerFunc(a.Handler.Update), "PUT /payments/{id}"))
	mux.Handle("DELETE /payments/{id}", otelhttp.NewHandler(http.HandlerFunc(a.Handler.Delete), "DELETE /payments/{id}"))

}

func (a *Payment) Shutdown() error {

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("payment", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	logger := zap.New(core)

	if a.cctx != nil {
		logger.Info("draining the consumer")
		a.cctx.Drain()
	}
	return a.natsConn.Drain()
}

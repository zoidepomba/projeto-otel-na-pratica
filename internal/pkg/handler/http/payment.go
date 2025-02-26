// Copyright Dose de Telemetria GmbH
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/model"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/store"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// PaymentHandler is an HTTP handler that performs CRUD operations for model.Payment using a store.Payment
type PaymentHandler struct {
	store                 store.Payment
	js                    jetstream.JetStream
	jsSubject             string
	subscriptionsEndpoint string
}

// NewPaymentHandler returns a new PaymentHandler
func NewPaymentHandler(store store.Payment, js jetstream.JetStream, jsSubject string, subscriptionsEndpoint string) *PaymentHandler {
	return &PaymentHandler{
		store:                 store,
		js:                    js,
		jsSubject:             jsSubject,
		subscriptionsEndpoint: subscriptionsEndpoint,
	}
}

func (h *PaymentHandler) List(w http.ResponseWriter, r *http.Request) {
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("payment", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	logger := zap.New(core)

	ctx, span := otel.Tracer("payment").Start(r.Context(), "List")

	traceID := span.SpanContext().TraceID().String()

	defer span.End()

	payments, err := h.store.List(ctx)
	if err != nil {
		logger.Error("failed to list payments", zap.Error(err), zap.String("trace_id:", traceID))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to list payments", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("payments listed", zap.String("trace_id", traceID))

	err = json.NewEncoder(w).Encode(payments)
	if err != nil {
		logger.Error("failed to encode payments", zap.Error(err), zap.String("trace_id", traceID))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to encode payments", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *PaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("payment", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	logger := zap.New(core)

	ctx, span := otel.Tracer("payment").Start(r.Context(), "Create")

	traceID := span.SpanContext().TraceID().String()
	defer span.End()

	var payment model.Payment
	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		logger.Error("failed to decode payment", zap.Error(err), zap.String("trace_id", traceID))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to decode payment", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	logger.Info("payment decoded", zap.String("trace_id", traceID))
	// Check if subscription exists
	sub, err := otelhttp.Get(ctx, h.subscriptionsEndpoint + "/" + payment.SubscriptionID)
	if err != nil {
		logger.Error("failed to get subscription", zap.Error(err), zap.String("trace_id", traceID))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to get subscription", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("subscription retrieved", zap.String("trace_id", traceID))
	defer sub.Body.Close()
	if sub.StatusCode != http.StatusOK {
		logger.Error("subscription not found", zap.String("subscription_id", payment.SubscriptionID), zap.String("trace_id", traceID))
		span.SetStatus(codes.Error, "Subscription not found")
		span.RecordError(err)
		span.AddEvent("subscription not found")
		http.Error(w, "Subscription not found", http.StatusBadRequest)
		return
	}
	logger.Info("subscription found", zap.String("trace_id", traceID))
	defer sub.Body.Close()

	payload, err := json.Marshal(payment)
	if err != nil {
		logger.Error("failed to marshal payment", zap.Error(err), zap.String("trace_id", traceID))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to marshal payment", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("payment marshaled", zap.String("trace_id", traceID))

	_, err = h.js.PublishMsgAsync(&nats.Msg{
		Subject: h.jsSubject,
		Data:    payload,
	})
	if err != nil {
		logger.Error("failed to publish message", zap.Error(err), zap.String("trace_id", traceID))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to publish message", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("message published", zap.String("trace_id", traceID))

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(payment)
	if err != nil {
		logger.Error("failed to encode payment", zap.Error(err), zap.String("trace_id", traceID))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to encode payment", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("payment encoded", zap.String("trace_id", traceID))
}

func (h *PaymentHandler) Get(w http.ResponseWriter, r *http.Request) {

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("payment", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	logger := zap.New(core)

	ctx, span := otel.Tracer("payment").Start(r.Context(), "Create")

	defer span.End()

	id := r.PathValue("id")

	payment, err := h.store.Get(ctx, id)
	if err != nil {
		logger.Error("failed to get payment", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to get payment", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("payment retrieved")

	if payment == nil {
		logger.Error("payment not found")
		span.SetStatus(codes.Error, "Payment not found")
		span.RecordError(err)
		span.AddEvent("payment not found")
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}
	logger.Info("payment found")

	err = json.NewEncoder(w).Encode(payment)
	if err != nil {
		logger.Error("failed to encode payment", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to encode payment", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("payment encoded")
}

func (h *PaymentHandler) Update(w http.ResponseWriter, r *http.Request) {
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("payment", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	logger := zap.New(core)

	ctx, span := otel.Tracer("payment").Start(r.Context(), "Create")

	defer span.End()

	payment := &model.Payment{}
	if err := json.NewDecoder(r.Body).Decode(payment); err != nil {
		logger.Error("failed to decode payment", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to decode payment", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	logger.Info("payment decoded")

	_, err := h.store.Update(ctx, payment)
	if err != nil {
		logger.Error("failed to update payment", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to update payment", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("payment updated")

	err = json.NewEncoder(w).Encode(payment)
	if err != nil {
		logger.Error("failed to encode payment", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to encode payment", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("payment encoded")
}

func (h *PaymentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("payment", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	logger := zap.New(core)

	ctx, span := otel.Tracer("payment").Start(r.Context(), "Create")

	defer span.End()

	id := r.PathValue("id")
	err := h.store.Delete(ctx, id)
	if err != nil {
		logger.Error("failed to delete payment", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to delete payment", trace.WithAttributes(attribute.String("error", err.Error())))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("payment deleted")
}

func (h *PaymentHandler) OnMessage(msg jetstream.Msg) {
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("payment", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	logger := zap.New(core)

	ctx, span := otel.Tracer("payment").Start(context.Background(), "OnMessage")

	defer span.End()

	payment := &model.Payment{}
	err := json.Unmarshal(msg.Data(), payment)
	if err != nil {
		logger.Error("failed to unmarshal payment", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to unmarshal payment", trace.WithAttributes(attribute.String("error", err.Error())))
		return
	}
	logger.Info("payment unmarshaled")

	_, err = h.store.Create(ctx, payment)
	if err != nil {
		logger.Error("failed to create payment", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.AddEvent("failed to create payment", trace.WithAttributes(attribute.String("error", err.Error())))
		return
	}
	logger.Info("payment created")

	_ = msg.Ack()
}

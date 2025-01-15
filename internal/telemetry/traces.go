package telemetry

import (
	"context"

	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

func InitTraces() {
	logger := log.NewLogeer()
	exp, err := otlptracegrpc.New(context.Background())
	if err != nil {
		logger.Error("err")	
	}
	logger.Info("Trace instrumentation carried out")
	otel.SetTracerProvider(trace.NewTracerProvider(trace.WithBatcher(exp)))
}
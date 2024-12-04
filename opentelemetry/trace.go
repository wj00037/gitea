package opentelemetry

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"code.gitea.io/gitea/modules/setting"
	"github.com/sirupsen/logrus"
)

// Initialize a gRPC connection to be used by both the tracer and meter
// providers.
func initConn(config *setting.OtelConfig) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(config.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	return conn, err
}

// Initializes an OTLP exporter, and configures the corresponding trace provider.
func initTracerProvider(
	ctx context.Context,
	res *resource.Resource,
	conn *grpc.ClientConn,
	fractions float64) (func(context.Context) error, error) {
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(fractions)),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}

func InitTrace(config *setting.OtelConfig) func() {
	logrus.Infof("InitTrace... ")

	if config == nil {
		logrus.Fatal("config is nil")
	}

	if !config.Enabled {
		logrus.Info("Tracing: disabled")
		return func() {}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	conn, err := initConn(&setting.Otel)
	if err != nil {
		logrus.Fatal(err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// The service name used to display traces in backends
			semconv.ServiceNameKey.String(config.Name),
		),
	)
	if err != nil {
		logrus.Fatal(err)
	}

	shutdownTracerProvider, err := initTracerProvider(ctx, res, conn, config.Fractions)
	if err != nil {
		logrus.Fatal(err)
	}

	shutdown := func() {
		if err := shutdownTracerProvider(ctx); err != nil {
			logrus.Fatalf("failed to shutdown TracerProvider: %s", err)
		}
	}

	logrus.Infof("Tracing: %v enabled successfully", config)

	return shutdown
}

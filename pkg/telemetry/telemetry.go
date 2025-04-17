// Package telemetry provides OpenTelemetry-based monitoring capabilities for AI applications.
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds configuration options for the telemetry client.
type Config struct {
	ServiceName    string
	Environment    string
	OTelEndpoint   string
	MetricsEnabled bool
	TracingEnabled bool
}

// Event represents a telemetry event to be recorded.
type Event struct {
	Timestamp    time.Time
	IP           string
	UserID       string
	Endpoint     string
	Method       string
	Destination  string
	StatusCode   int
	Duration     time.Duration
	Score        float64
	Reasons      []string
	Blocked      bool
	RequestData  map[string]interface{}
	ResponseData map[string]interface{}
}

// Client handles all telemetry operations.
type Client struct {
	config         Config
	meter          metric.Meter
	tracer         trace.Tracer
	metricExporter sdkmetric.Exporter
	traceExporter  sdktrace.SpanExporter
	meterProvider  *sdkmetric.MeterProvider
	tracerProvider *sdktrace.TracerProvider

	// Counters and gauges
	requestCounter    metric.Int64Counter
	blockedCounter    metric.Int64Counter
	latencyHistogram  metric.Float64Histogram
	requestsPerSecond metric.Float64UpDownCounter
}

// New creates a new telemetry client with OpenTelemetry instrumentation.
func New(config Config) (*Client, error) {
	client := &Client{
		config: config,
	}

	// Create a resource describing the service
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize OpenTelemetry if endpoint is provided
	if config.OTelEndpoint != "" {
		// Setup metrics if enabled
		if config.MetricsEnabled {
			if err := client.setupMetrics(res); err != nil {
				return nil, fmt.Errorf("failed to setup metrics: %w", err)
			}
		}

		// Setup tracing if enabled
		if config.TracingEnabled {
			if err := client.setupTracing(res); err != nil {
				return nil, fmt.Errorf("failed to setup tracing: %w", err)
			}
		}
	}

	return client, nil
}

// setupMetrics initializes the OpenTelemetry metrics pipeline
func (c *Client) setupMetrics(res *resource.Resource) error {
	ctx := context.Background()

	// Set up the connection to the OTLP endpoint
	conn, err := grpc.DialContext(ctx, c.config.OTelEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Create the OTLP exporter
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	// Create MeterProvider with the exporter
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
			// Default is 60s, reduce for faster metrics visibility during development
			sdkmetric.WithInterval(1*time.Second))),
	)

	// Set the global MeterProvider
	otel.SetMeterProvider(meterProvider)

	// Create a meter
	meter := meterProvider.Meter(
		"github.com/rohanadwankar/guardian",
		metric.WithInstrumentationVersion(c.Version()),
	)

	// Initialize the metrics with Prometheus-compatible naming (using underscores instead of dots)
	var err1, err2, err3, err4 error
	c.requestCounter, err1 = meter.Int64Counter(
		"guardian_requests_total",
		metric.WithDescription("Total number of AI requests processed"),
	)

	c.blockedCounter, err2 = meter.Int64Counter(
		"guardian_requests_blocked_total",
		metric.WithDescription("Total number of AI requests blocked"),
	)

	c.latencyHistogram, err3 = meter.Float64Histogram(
		"guardian_request_duration_seconds",
		metric.WithDescription("Latency of AI requests in seconds"),
		metric.WithUnit("s"),
	)

	c.requestsPerSecond, err4 = meter.Float64UpDownCounter(
		"guardian_requests_per_second",
		metric.WithDescription("Number of AI requests per second"),
		metric.WithUnit("{requests}"),
	)

	// Check for errors in creating instruments
	for _, err := range []error{err1, err2, err3, err4} {
		if err != nil {
			return fmt.Errorf("failed to create metric instruments: %w", err)
		}
	}

	// Store our instances
	c.meter = meter
	c.metricExporter = metricExporter
	c.meterProvider = meterProvider

	fmt.Printf("OpenTelemetry metrics initialized, sending to %s\n", c.config.OTelEndpoint)
	return nil
}

// setupTracing initializes the OpenTelemetry tracing pipeline
func (c *Client) setupTracing(res *resource.Resource) error {
	ctx := context.Background()

	// Set up the connection to the OTLP endpoint (reuse if metrics already established)
	conn, err := grpc.DialContext(ctx, c.config.OTelEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Create the trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create TracerProvider with batch span processor and exporter
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set the global TracerProvider and Propagator
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Create a tracer
	tracer := tracerProvider.Tracer(
		"github.com/rohanadwankar/guardian",
		trace.WithInstrumentationVersion(c.Version()),
	)

	// Store our instances
	c.tracer = tracer
	c.traceExporter = traceExporter
	c.tracerProvider = tracerProvider

	fmt.Printf("OpenTelemetry tracing initialized, sending to %s\n", c.config.OTelEndpoint)
	return nil
}

// Version returns the current version of the telemetry package.
func (c *Client) Version() string {
	return "0.1.0"
}

// RecordEvent records a telemetry event.
func (c *Client) RecordEvent(ctx context.Context, event Event) error {
	if !c.config.MetricsEnabled && !c.config.TracingEnabled {
		return nil
	}

	// Common attributes for the event
	attrs := []attribute.KeyValue{
		attribute.String("endpoint", event.Endpoint),
		attribute.String("user_id", event.UserID),
		attribute.String("ip", event.IP),
	}

	// Add UserID if available
	if event.UserID != "" {
		attrs = append(attrs, semconv.EnduserIDKey.String(event.UserID))
	}

	// Record metrics if enabled
	if c.config.MetricsEnabled && c.meter != nil {
		// Record the request count
		c.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))

		// Record request latency in seconds (convert from milliseconds)
		durationSeconds := float64(event.Duration.Milliseconds()) / 1000.0
		c.latencyHistogram.Record(ctx, durationSeconds, metric.WithAttributes(attrs...))

		// If this is a blocked request, increment the counter
		if event.Score >= 0.85 {
			blockedAttrs := append(attrs, attribute.Float64("score", event.Score))
			c.blockedCounter.Add(ctx, 1, metric.WithAttributes(blockedAttrs...))
		}
	}

	// Record span if tracing is enabled
	if c.config.TracingEnabled && c.tracer != nil {
		_, span := c.tracer.Start(ctx,
			fmt.Sprintf("AI Request: %s", event.Endpoint),
			trace.WithAttributes(attrs...))
		span.SetAttributes(attribute.Float64("score", event.Score))
		span.SetAttributes(attribute.Int64("duration_ms", event.Duration.Milliseconds()))

		// Add reasons as attributes if available
		for i, reason := range event.Reasons {
			span.SetAttributes(attribute.String(fmt.Sprintf("reason.%d", i), reason))
		}

		span.End()
	}

	return nil
}

// UpdateMessagesPerSecond updates the messages per second metric.
func (c *Client) UpdateMessagesPerSecond(ctx context.Context, mps float64) {
	if !c.config.MetricsEnabled || c.meter == nil {
		return
	}

	// Record the current rate
	c.requestsPerSecond.Add(ctx, mps,
		metric.WithAttributes(
			attribute.String("service", c.config.ServiceName),
			attribute.String("environment", c.config.Environment),
		))
}

// StartSpan starts a new tracing span.
func (c *Client) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if !c.config.TracingEnabled || c.tracer == nil {
		// Return a no-op span if tracing is not enabled
		return ctx, trace.SpanFromContext(ctx)
	}

	return c.tracer.Start(ctx, name)
}

// EndSpan ends a tracing span.
func (c *Client) EndSpan(span trace.Span) {
	if span != nil {
		span.End()
	}
}

// Shutdown gracefully shuts down the telemetry client.
func (c *Client) Shutdown() error {
	ctx := context.Background()

	// Shutdown meter provider if it exists
	if c.meterProvider != nil {
		if err := c.meterProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown meter provider: %w", err)
		}
	}

	// Shutdown tracer provider if it exists
	if c.tracerProvider != nil {
		if err := c.tracerProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown tracer provider: %w", err)
		}
	}

	return nil
}

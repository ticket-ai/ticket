// Package telemetry provides OpenTelemetry-based monitoring capabilities for AI applications.
package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/rohanadwankar/guardian/pkg/analyzer" // Import analyzer for Rule type

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

// --- Placeholder Costs (per 1000 tokens) ---
// These should ideally be configurable or fetched dynamically based on the model used.
const (
	costPer1kInputTokens  = 0.001 // Example: $0.001 / 1k input tokens
	costPer1kOutputTokens = 0.002 // Example: $0.002 / 1k output tokens
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
	Timestamp     time.Time
	IP            string
	UserID        string
	Endpoint      string
	Method        string
	Destination   string
	StatusCode    int
	Duration      time.Duration
	Score         float64
	Reasons       []string
	Blocked       bool
	InputTokens   int
	OutputTokens  int
	EstimatedCost float64
	RequestData   map[string]interface{}
	ResponseData  map[string]interface{}
	MatchedRules  []analyzer.Rule // Added field to store matched rules
}

// Client handles all telemetry operations.
type Client struct {
	config               Config
	meter                metric.Meter
	tracer               trace.Tracer
	metricExporter       sdkmetric.Exporter
	traceExporter        sdktrace.SpanExporter
	meterProvider        *sdkmetric.MeterProvider
	tracerProvider       *sdktrace.TracerProvider
	requestCounter       metric.Int64Counter
	blockedCounter       metric.Int64Counter
	latencyHistogram     metric.Float64Histogram
	inputTokensCounter   metric.Int64Counter
	outputTokensCounter  metric.Int64Counter
	estimatedCostCounter metric.Float64Counter
	flaggedCounter       metric.Int64Counter // Added counter for flagged requests
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
	var err1, err2, err3, err4, err5, err6, err7 error
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

	c.inputTokensCounter, err4 = meter.Int64Counter(
		"guardian_input_tokens_total",
		metric.WithDescription("Total number of estimated input tokens processed"),
		metric.WithUnit("{token}"),
	)

	c.outputTokensCounter, err5 = meter.Int64Counter(
		"guardian_output_tokens_total",
		metric.WithDescription("Total number of estimated output tokens generated"),
		metric.WithUnit("{token}"),
	)

	c.estimatedCostCounter, err6 = meter.Float64Counter(
		"guardian_estimated_cost_total",
		metric.WithDescription("Estimated cost of AI requests based on token usage"),
		metric.WithUnit("USD"),
	)

	c.flaggedCounter, err7 = meter.Int64Counter(
		"guardian_flagged_requests_total",
		metric.WithDescription("Total number of AI requests flagged by rules"),
	)

	// Check for errors in creating instruments
	for _, err := range []error{err1, err2, err3, err4, err5, err6, err7} {
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

	// --- Calculate Estimated Cost ---
	event.EstimatedCost = (float64(event.InputTokens)/1000.0)*costPer1kInputTokens +
		(float64(event.OutputTokens)/1000.0)*costPer1kOutputTokens

	// Common attributes for the event
	attrs := []attribute.KeyValue{
		attribute.String("endpoint", event.Endpoint),
		attribute.String("user_id", event.UserID),
		attribute.String("ip", event.IP),
		attribute.Int("guardian.tokens.input", event.InputTokens),
		attribute.Int("guardian.tokens.output", event.OutputTokens),
		attribute.Float64("guardian.estimated_cost", event.EstimatedCost),
		attribute.Bool("guardian.blocked", event.Blocked),         // Added blocked status attribute
		attribute.Float64("guardian.analysis_score", event.Score), // Added analysis score attribute
	}

	// Add attributes for matched rules
	ruleAttrs := []attribute.KeyValue{}
	for i, rule := range event.MatchedRules {
		prefix := fmt.Sprintf("guardian.rule.%d.", i)
		ruleAttrs = append(ruleAttrs,
			attribute.String(prefix+"name", rule.Name),
			attribute.String(prefix+"severity", rule.Severity),
		)
	}
	attrs = append(attrs, ruleAttrs...)

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

		// Record token counts and cost
		c.inputTokensCounter.Add(ctx, int64(event.InputTokens), metric.WithAttributes(attrs...))
		c.outputTokensCounter.Add(ctx, int64(event.OutputTokens), metric.WithAttributes(attrs...))
		c.estimatedCostCounter.Add(ctx, event.EstimatedCost, metric.WithAttributes(attrs...))

		// If this is a blocked request, increment the counter
		if event.Blocked {
			// Blocked counter uses the base attributes + score
			blockedAttrs := append(attrs, attribute.Float64("guardian.analysis_score", event.Score))
			c.blockedCounter.Add(ctx, 1, metric.WithAttributes(blockedAttrs...))
		}

		// If the request was flagged (matched any rules), increment the flagged counter
		if len(event.MatchedRules) > 0 {
			// Flagged counter uses base attributes + rule attributes
			flaggedAttrs := attrs // Already contains rule attributes
			c.flaggedCounter.Add(ctx, 1, metric.WithAttributes(flaggedAttrs...))
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

		if event.Blocked {
			span.SetStatus(codes.Error, "Request blocked by Guardian")
		} else if event.StatusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP Error: %d", event.StatusCode))
		} else {
			span.SetStatus(codes.Ok, "Success")
		}

		span.End()
	}

	return nil
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

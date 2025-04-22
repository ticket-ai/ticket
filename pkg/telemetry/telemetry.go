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
	costPer1kInputTokens  = 0.0025 // Example: $0.0025 / 1k input tokens
	costPer1kOutputTokens = 0.01   // Example: $0.01 / 1k output tokens
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
	MatchedRules  []analyzer.Rule // Store matched rules

	// NLP Analysis Metrics
	NLPMetrics analyzer.NLPMetrics // All NLP metrics from analyzer
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

	// Existing metrics
	requestCounter       metric.Int64Counter
	blockedCounter       metric.Int64Counter
	latencyHistogram     metric.Float64Histogram
	inputTokensCounter   metric.Int64Counter
	outputTokensCounter  metric.Int64Counter
	estimatedCostCounter metric.Float64Counter
	flaggedCounter       metric.Int64Counter

	// NLP metrics
	sentimentGauge        metric.Float64UpDownCounter // Can be negative
	toxicityHistogram     metric.Float64Histogram
	piiHistogram          metric.Float64Histogram
	profanityHistogram    metric.Float64Histogram
	biasHistogram         metric.Float64Histogram
	emotionalHistogram    metric.Float64Histogram
	manipulativeHistogram metric.Float64Histogram
	jailbreakHistogram    metric.Float64Histogram
	keywordCounter        metric.Int64Counter // For counting keyword occurrences
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

	fmt.Printf("Successfully created gRPC connection to %s\n", c.config.OTelEndpoint)

	// Create the OTLP exporter
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	fmt.Printf("Successfully created OTLP metric exporter\n")

	// Create MeterProvider with the exporter
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
			// Default is 60s, reduce for faster metrics visibility during development
			sdkmetric.WithInterval(1*time.Second))),
	)

	fmt.Printf("Successfully created MeterProvider\n")

	// Set the global MeterProvider
	otel.SetMeterProvider(meterProvider)

	// Create a meter
	meter := meterProvider.Meter(
		"github.com/rohanadwankar/guardian",
		metric.WithInstrumentationVersion(c.Version()),
	)

	fmt.Printf("Successfully created Meter\n")

	// Initialize the metrics with Prometheus-compatible naming (using underscores instead of dots)
	var err1, err2, err3, err4, err5, err6, err7 error

	// Basic metrics
	c.requestCounter, err1 = meter.Int64Counter(
		"guardian_requests_total",
		metric.WithDescription("Total number of AI requests processed"),
	)
	if err1 != nil {
		return fmt.Errorf("failed to create requestCounter: %w", err1)
	}

	c.blockedCounter, err2 = meter.Int64Counter(
		"guardian_requests_blocked_total",
		metric.WithDescription("Total number of AI requests blocked"),
	)
	if err2 != nil {
		return fmt.Errorf("failed to create blockedCounter: %w", err2)
	}

	c.latencyHistogram, err3 = meter.Float64Histogram(
		"guardian_request_duration_seconds",
		metric.WithDescription("Latency of AI requests in seconds"),
		metric.WithUnit("s"),
	)
	if err3 != nil {
		return fmt.Errorf("failed to create latencyHistogram: %w", err3)
	}

	c.inputTokensCounter, err4 = meter.Int64Counter(
		"guardian_input_tokens_total",
		metric.WithDescription("Total number of estimated input tokens processed"),
		metric.WithUnit("{token}"),
	)
	if err4 != nil {
		return fmt.Errorf("failed to create inputTokensCounter: %w", err4)
	}

	c.outputTokensCounter, err5 = meter.Int64Counter(
		"guardian_output_tokens_total",
		metric.WithDescription("Total number of estimated output tokens generated"),
		metric.WithUnit("{token}"),
	)
	if err5 != nil {
		return fmt.Errorf("failed to create outputTokensCounter: %w", err5)
	}

	c.estimatedCostCounter, err6 = meter.Float64Counter(
		"guardian_estimated_cost_USD_total",
		metric.WithDescription("Estimated cost of AI requests based on token usage"),
		metric.WithUnit("USD"),
	)
	if err6 != nil {
		return fmt.Errorf("failed to create estimatedCostCounter: %w", err6)
	}

	c.flaggedCounter, err7 = meter.Int64Counter(
		"guardian_flagged_requests_total",
		metric.WithDescription("Total number of AI requests flagged by rules"),
	)
	if err7 != nil {
		return fmt.Errorf("failed to create flaggedCounter: %w", err7)
	}

	fmt.Printf("Successfully created basic metrics\n")

	// Initialize NLP metrics
	fmt.Printf("Starting NLP metrics initialization\n")
	var nlpErr1, nlpErr2, nlpErr3, nlpErr4, nlpErr5, nlpErr6, nlpErr7, nlpErr8 error

	// Sentiment can be negative, so we use an UpDownCounter
	c.sentimentGauge, nlpErr1 = meter.Float64UpDownCounter(
		"guardian_nlp_sentiment",
		metric.WithDescription("Sentiment analysis score from -1.0 (negative) to 1.0 (positive)"),
	)
	if nlpErr1 != nil {
		return fmt.Errorf("failed to create sentimentGauge: %w", nlpErr1)
	}
	fmt.Printf("Created sentiment gauge\n")

	c.toxicityHistogram, nlpErr2 = meter.Float64Histogram(
		"guardian_nlp_toxicity",
		metric.WithDescription("Distribution of toxicity scores from 0.0 (not toxic) to 1.0 (very toxic)"),
	)
	if nlpErr2 != nil {
		return fmt.Errorf("failed to create toxicityHistogram: %w", nlpErr2)
	}
	fmt.Printf("Created toxicity histogram\n")

	c.piiHistogram, nlpErr3 = meter.Float64Histogram(
		"guardian_nlp_pii_detection",
		metric.WithDescription("Distribution of PII detection scores from 0.0 (no PII) to 1.0 (definite PII)"),
	)
	if nlpErr3 != nil {
		return fmt.Errorf("failed to create piiHistogram: %w", nlpErr3)
	}
	fmt.Printf("Created PII histogram\n")

	c.profanityHistogram, nlpErr4 = meter.Float64Histogram(
		"guardian_nlp_profanity",
		metric.WithDescription("Distribution of profanity scores from 0.0 (no profanity) to 1.0 (high profanity)"),
	)
	if nlpErr4 != nil {
		return fmt.Errorf("failed to create profanityHistogram: %w", nlpErr4)
	}
	fmt.Printf("Created profanity histogram\n")

	c.biasHistogram, nlpErr5 = meter.Float64Histogram(
		"guardian_nlp_bias",
		metric.WithDescription("Distribution of bias scores from 0.0 (unbiased) to 1.0 (highly biased)"),
	)
	if nlpErr5 != nil {
		return fmt.Errorf("failed to create biasHistogram: %w", nlpErr5)
	}
	fmt.Printf("Created bias histogram\n")

	c.emotionalHistogram, nlpErr6 = meter.Float64Histogram(
		"guardian_nlp_emotional",
		metric.WithDescription("Distribution of emotional content scores from 0.0 (not emotional) to 1.0 (highly emotional)"),
	)
	if nlpErr6 != nil {
		return fmt.Errorf("failed to create emotionalHistogram: %w", nlpErr6)
	}
	fmt.Printf("Created emotional histogram\n")

	c.manipulativeHistogram, nlpErr7 = meter.Float64Histogram(
		"guardian_nlp_manipulative",
		metric.WithDescription("Distribution of manipulative content scores from 0.0 (not manipulative) to 1.0 (highly manipulative)"),
	)
	if nlpErr7 != nil {
		return fmt.Errorf("failed to create manipulativeHistogram: %w", nlpErr7)
	}
	fmt.Printf("Created manipulative histogram\n")

	c.jailbreakHistogram, nlpErr8 = meter.Float64Histogram(
		"guardian_nlp_jailbreak",
		metric.WithDescription("Distribution of jailbreak intent scores from 0.0 (no jailbreak intent) to 1.0 (definite jailbreak)"),
	)
	if nlpErr8 != nil {
		return fmt.Errorf("failed to create jailbreakHistogram: %w", nlpErr8)
	}
	fmt.Printf("Created jailbreak histogram\n")

	c.keywordCounter, err = meter.Int64Counter(
		"guardian_nlp_keywords_total",
		metric.WithDescription("Count of specific keywords detected in content"),
	)
	if err != nil {
		return fmt.Errorf("failed to create keywordCounter: %w", err)
	}
	fmt.Printf("Created keyword counter\n")

	fmt.Printf("All NLP metrics initialized successfully\n")

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
		attribute.Bool("guardian.blocked", event.Blocked),
		attribute.Float64("guardian.analysis_score", event.Score),
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

	// Add NLP metrics as attributes for better filtering in dashboards
	nlpAttrs := []attribute.KeyValue{
		attribute.Float64("guardian.nlp.sentiment", event.NLPMetrics.Sentiment),
		attribute.Float64("guardian.nlp.toxicity", event.NLPMetrics.Toxicity),
		attribute.Float64("guardian.nlp.pii", event.NLPMetrics.PII),
		attribute.Float64("guardian.nlp.profanity", event.NLPMetrics.Profanity),
		attribute.Float64("guardian.nlp.bias", event.NLPMetrics.Bias),
		attribute.Float64("guardian.nlp.emotional", event.NLPMetrics.Emotional),
		attribute.Float64("guardian.nlp.manipulative", event.NLPMetrics.Manipulative),
		attribute.Float64("guardian.nlp.jailbreak_intent", event.NLPMetrics.JailbreakIntent),
	}
	attrs = append(attrs, nlpAttrs...)

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

		// Record NLP metrics
		// For sentiment we record the actual value (-1.0 to 1.0)
		c.sentimentGauge.Add(ctx, event.NLPMetrics.Sentiment, metric.WithAttributes(attrs...))

		// For all other metrics, we record the distribution
		c.toxicityHistogram.Record(ctx, event.NLPMetrics.Toxicity, metric.WithAttributes(attrs...))
		c.piiHistogram.Record(ctx, event.NLPMetrics.PII, metric.WithAttributes(attrs...))
		c.profanityHistogram.Record(ctx, event.NLPMetrics.Profanity, metric.WithAttributes(attrs...))
		c.biasHistogram.Record(ctx, event.NLPMetrics.Bias, metric.WithAttributes(attrs...))
		c.emotionalHistogram.Record(ctx, event.NLPMetrics.Emotional, metric.WithAttributes(attrs...))
		c.manipulativeHistogram.Record(ctx, event.NLPMetrics.Manipulative, metric.WithAttributes(attrs...))
		c.jailbreakHistogram.Record(ctx, event.NLPMetrics.JailbreakIntent, metric.WithAttributes(attrs...))

		// Record detected keywords
		for keyword, confidence := range event.NLPMetrics.Keywords {
			keywordAttrs := append(attrs,
				attribute.String("guardian.nlp.keyword", keyword),
				attribute.Float64("guardian.nlp.confidence", confidence),
			)
			c.keywordCounter.Add(ctx, 1, metric.WithAttributes(keywordAttrs...))
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

		// Add NLP metrics to span
		span.SetAttributes(attribute.Float64("guardian.nlp.sentiment", event.NLPMetrics.Sentiment))
		span.SetAttributes(attribute.Float64("guardian.nlp.toxicity", event.NLPMetrics.Toxicity))
		span.SetAttributes(attribute.Float64("guardian.nlp.pii", event.NLPMetrics.PII))
		span.SetAttributes(attribute.Float64("guardian.nlp.jailbreak_intent", event.NLPMetrics.JailbreakIntent))

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

// GetTotalRequests returns the total number of requests processed
func (c *Client) GetTotalRequests(ctx context.Context) int {
	// For a proper implementation, you'd maintain a counter or fetch from your metrics system
	return 200 // Placeholder value
}

// GetFlaggedRequests returns the number of flagged requests
func (c *Client) GetFlaggedRequests(ctx context.Context) int {
	// For a proper implementation, you'd maintain a counter or fetch from your metrics system
	return 15 // Placeholder value
}

// GetEstimatedCost returns the total estimated cost
func (c *Client) GetEstimatedCost(ctx context.Context) float64 {
	// For a proper implementation, you'd maintain a running sum or fetch from your metrics system
	return 12.50 // Placeholder value
}

// GetRecentToxicityScores returns recent toxicity scores
func (c *Client) GetRecentToxicityScores(ctx context.Context, limit int) []float64 {
	// In a real implementation, you'd maintain a circular buffer or other data structure
	return []float64{0.1, 0.2, 0.3, 0.4, 0.5} // Placeholder
}

// Add these functions after the existing GetRecentToxicityScores method

// GetBlockedRequests returns the number of blocked requests
func (c *Client) GetBlockedRequests(ctx context.Context) int {
	// For a proper implementation, you'd maintain a counter or fetch from your metrics system
	return 5 // Placeholder value
}

// GetAverageScore returns the average analysis score
func (c *Client) GetAverageScore(ctx context.Context) float64 {
	// For a proper implementation, you'd calculate this from accumulated scores
	return 0.42 // Placeholder value
}

// GetRecentProfanityScores returns recent profanity scores
func (c *Client) GetRecentProfanityScores(ctx context.Context, limit int) []float64 {
	// In a real implementation, you'd maintain a circular buffer or other data structure
	return []float64{0.05, 0.15, 0.0, 0.2, 0.1} // Placeholder
}

// GetRecentPIIScores returns recent PII detection scores
func (c *Client) GetRecentPIIScores(ctx context.Context, limit int) []float64 {
	// In a real implementation, you'd maintain a circular buffer or other data structure
	return []float64{0.0, 0.1, 0.3, 0.05, 0.0} // Placeholder
}

// GetRecentBiasScores returns recent bias scores
func (c *Client) GetRecentBiasScores(ctx context.Context, limit int) []float64 {
	// In a real implementation, you'd maintain a circular buffer or other data structure
	return []float64{0.1, 0.15, 0.2, 0.05, 0.1} // Placeholder
}

// GetRequestsPerModel returns the count of requests per model
func (c *Client) GetRequestsPerModel(ctx context.Context) map[string]int {
	// In a real implementation, you'd track this per model
	return map[string]int{
		"gpt-3.5-turbo": 65,
		"gpt-4":         25,
		"claude-2":      10,
	} // Placeholder
}

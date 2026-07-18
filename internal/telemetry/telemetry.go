// Package telemetry sends minimal, anonymous usage and error data so gitle's
// author can spot slowdowns and crashes across releases. It never sends repo
// paths, branch names, commit messages, or anything else typed by the user —
// only the subcommand name, how long it took, and whether it failed.
//
// It is safe to call every function here with a zero-value or missing API
// key: telemetry silently no-ops instead of erroring, so a dev build (or an
// offline machine) never breaks the actual git command being run.
package telemetry

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// grafanaOTLPEndpoint is the full OTLP/HTTP traces URL for the Grafana
// Cloud stack's gateway. It's account-specific but not a secret on its own
// (it's useless without the Authorization header below) — WithEndpointURL
// takes the path verbatim, so /v1/traces must be spelled out here.
const grafanaOTLPEndpoint = "https://otlp-gateway-prod-gb-south-1.grafana.net/otlp/v1/traces"

// exportTimeout bounds every network call telemetry makes. gitle wraps git
// and must stay snappy even when the machine is offline, so this is the
// hard cap on how much latency telemetry can ever add to a command.
const exportTimeout = 2 * time.Second

var tracer = otel.Tracer("gitle")

// Start wires up the OTel SDK against Grafana Cloud using authHeader (the
// literal "Basic <token>" value of a Grafana Cloud access policy token,
// baked in at build time — see .goreleaser.yaml). It returns a shutdown
// func that flushes the invocation's span; callers must invoke it before
// any os.Exit, since os.Exit skips deferred calls.
//
// Telemetry is disabled (shutdown is a no-op) when authHeader is empty,
// when GITLE_TELEMETRY=0 is set, or when DO_NOT_TRACK is set (the
// community convention: https://consoledonottrack.com).
func Start(ctx context.Context, authHeader, version string) (shutdown func()) {
	if authHeader == "" || disabled() {
		return func() {}
	}

	// The SDK's default error handler logs export failures (offline, bad
	// token, Grafana down, ...) straight to stderr, which would spam a
	// user's terminal on every command. Telemetry is best-effort; failures
	// are silently swallowed instead.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {}))

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(grafanaOTLPEndpoint),
		otlptracehttp.WithHeaders(map[string]string{"Authorization": authHeader}),
		otlptracehttp.WithTimeout(exportTimeout),
	)
	if err != nil {
		return func() {}
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("gitle"),
			semconv.ServiceVersion(version),
			attribute.String("gitle.install_id", installID()),
			attribute.String("os", runtime.GOOS),
			attribute.String("arch", runtime.GOARCH),
		),
	)
	if err != nil {
		res = resource.Default()
	}

	// A single span per invocation: SimpleSpanProcessor exports it inline at
	// span.End() rather than batching in a background goroutine that a
	// short-lived CLI process might not live long enough to flush.
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	tracer = provider.Tracer("gitle")

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), exportTimeout)
		defer cancel()
		_ = provider.Shutdown(shutdownCtx)
	}
}

// RecordInvocation records one command run: its full path ("gitle save"),
// how long it took, and its outcome. errCategory is a small fixed label
// ("git_missing", "not_a_repo", "command_error", ...) chosen by the caller —
// empty means success. Raw error text is never sent, since it may embed
// repo paths or branch names.
func RecordInvocation(ctx context.Context, commandPath string, duration time.Duration, errCategory string) {
	_, span := tracer.Start(ctx, commandPath,
		trace.WithTimestamp(time.Now().Add(-duration)),
		trace.WithAttributes(attribute.Int64("gitle.duration_ms", duration.Milliseconds())),
	)
	if errCategory != "" {
		span.SetStatus(codes.Error, errCategory)
		span.SetAttributes(attribute.String("gitle.error_category", errCategory))
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End(trace.WithTimestamp(time.Now()))
}

func disabled() bool {
	if v := os.Getenv("GITLE_TELEMETRY"); v == "0" || strings.EqualFold(v, "false") {
		return true
	}
	return os.Getenv("DO_NOT_TRACK") != ""
}

// installID returns a random, anonymous per-install identifier, persisted
// under the user's config dir so repeat runs count as the same install. It
// carries no personal information — just a UUID with nothing tied to it.
func installID() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return uuid.NewString()
	}
	path := filepath.Join(dir, "gitle", "telemetry_id")

	if b, err := os.ReadFile(path); err == nil {
		if id := strings.TrimSpace(string(b)); id != "" {
			return id
		}
	}

	id := uuid.NewString()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
		_ = os.WriteFile(path, []byte(id), 0o644)
	}
	return id
}

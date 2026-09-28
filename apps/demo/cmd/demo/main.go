package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/opspilot/opspilot/apps/demo/internal/fault"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	name := env("SERVICE_NAME", "storefront")
	if name == "traffic" {
		runTraffic(logger)
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	shutdown, err := setupTracing(ctx, name)
	if err != nil {
		logger.Error("tracing", "error", err)
	}
	defer func() {
		if shutdown != nil {
			_ = shutdown(context.Background())
		}
	}()

	app := newApp(name)
	server := &http.Server{
		Addr:              env("LISTEN", ":8080"),
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("demo listening", "service", name, "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server", "error", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shut)
}

type app struct {
	name      string
	version   string
	namespace string
	token     string
	client    *http.Client
	requests  *prometheus.CounterVec
	duration  *prometheus.HistogramVec
	mu        sync.Mutex
	fault     fault.State
}

func newApp(name string) *app {
	labels := []string{"service", "namespace", "version", "route", "status_code"}
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "http_requests_total", Help: "HTTP requests."}, labels)
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
	}, []string{"service", "namespace", "version", "route"})
	prometheus.MustRegister(requests, duration)
	return &app{
		name:      name,
		version:   env("SERVICE_VERSION", "0.0.0"),
		namespace: env("NAMESPACE", "demo-shop"),
		token:     os.Getenv("FAULT_TOKEN"),
		client:    &http.Client{Timeout: 3 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)},
		requests:  requests,
		duration:  duration,
	}
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("POST /internal/fault", a.setFault)
	mux.Handle("GET /buy", otelhttp.NewHandler(http.HandlerFunc(a.buy), "GET /buy"))
	mux.Handle("GET /checkout", otelhttp.NewHandler(http.HandlerFunc(a.checkout), "GET /checkout"))
	mux.Handle("GET /pay", otelhttp.NewHandler(http.HandlerFunc(a.pay), "GET /pay"))
	mux.Handle("GET /orders", otelhttp.NewHandler(http.HandlerFunc(a.orders), "GET /orders"))
	mux.Handle("GET /stock", otelhttp.NewHandler(http.HandlerFunc(a.stock), "GET /stock"))
	mux.Handle("GET /", otelhttp.NewHandler(http.HandlerFunc(a.root), "GET /"))
	return mux
}

func (a *app) root(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	a.observe(r.URL.Path, http.StatusOK, time.Since(start))
	writeJSON(w, http.StatusOK, map[string]string{"service": a.name, "version": a.version})
}

func (a *app) buy(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	status, err := a.call(r.Context(), env("CHECKOUT_URL", "http://checkout-api/checkout"))
	if err != nil {
		a.finish(r, "/buy", http.StatusBadGateway, start)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "checkout unavailable"})
		return
	}
	a.finish(r, "/buy", status, start)
	writeJSON(w, status, map[string]string{"service": a.name, "downstream": strconv.Itoa(status)})
}

func (a *app) checkout(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	payStatus, err := a.call(r.Context(), env("PAYMENT_URL", "http://payment-api/pay"))
	if err != nil {
		a.finish(r, "/checkout", http.StatusBadGateway, start)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "payment unavailable"})
		return
	}
	_, _ = a.call(r.Context(), env("ORDERS_URL", "http://orders-api/orders"))
	code := payStatus
	if code < 400 {
		code = http.StatusOK
	}
	a.finish(r, "/checkout", code, start)
	writeJSON(w, code, map[string]string{"service": a.name, "payment": strconv.Itoa(payStatus)})
}

func (a *app) pay(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, span := otel.Tracer(a.name).Start(r.Context(), "db.query")
	span.SetAttributes(attribute.String("db.system", "postgresql"), attribute.String("db.operation", "insert_payment"))
	defer span.End()

	a.mu.Lock()
	active := a.fault.Active(time.Now())
	latency := a.fault.Latency
	percent := a.fault.ErrorPercent
	a.mu.Unlock()

	if active && latency > 0 {
		timer := time.NewTimer(latency)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	status := http.StatusOK
	if active && percent > 0 && percentDraw(percent) {
		status = http.StatusInternalServerError
		span.SetStatus(codes.Error, "injected payment failure")
	} else {
		span.SetStatus(codes.Ok, "")
	}
	if status >= 500 {
		span.RecordError(fmt.Errorf("injected http %d", status))
	}
	a.finish(r, "/pay", status, start)
	writeJSON(w, status, map[string]string{"service": a.name, "status": strconv.Itoa(status)})
}

func (a *app) orders(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	_, _ = a.call(r.Context(), env("INVENTORY_URL", "http://inventory-api/stock"))
	a.finish(r, "/orders", http.StatusOK, start)
	writeJSON(w, http.StatusOK, map[string]string{"service": a.name})
}

func (a *app) stock(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	a.finish(r, "/stock", http.StatusOK, start)
	writeJSON(w, http.StatusOK, map[string]string{"service": a.name, "stock": "ok"})
}

func (a *app) setFault(w http.ResponseWriter, r *http.Request) {
	if a.name != "payment-api" {
		http.Error(w, "fault injection is only available on payment-api", http.StatusForbidden)
		return
	}
	if a.token == "" || r.Header.Get("X-OpsPilot-Fault-Token") != a.token {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body struct {
		Mode  string    `json:"mode"`
		Until time.Time `json:"until"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<10)).Decode(&body); err != nil {
		http.Error(w, "invalid", http.StatusBadRequest)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	switch body.Mode {
	case "off":
		a.fault.Disable()
	case "degraded":
		if body.Until.IsZero() || time.Until(body.Until) <= 0 || time.Until(body.Until) > 5*time.Minute {
			http.Error(w, "until must be within 5 minutes", http.StatusBadRequest)
			return
		}
		a.fault.Enable(body.Until, 40, 500*time.Millisecond)
	default:
		http.Error(w, "unknown mode", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mode": body.Mode, "active": a.fault.Active(time.Now())})
}

func (a *app) call(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	res, err := a.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
	return res.StatusCode, nil
}

func (a *app) finish(r *http.Request, route string, status int, start time.Time) {
	elapsed := time.Since(start)
	if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() && status >= 500 {
		span.SetStatus(codes.Error, http.StatusText(status))
	}
	a.observe(route, status, elapsed)
}

func (a *app) observe(route string, status int, elapsed time.Duration) {
	code := strconv.Itoa(status)
	a.requests.WithLabelValues(a.name, a.namespace, a.version, route, code).Inc()
	a.duration.WithLabelValues(a.name, a.namespace, a.version, route).Observe(elapsed.Seconds())
}

func setupTracing(ctx context.Context, name string) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://otel-collector.opspilot-system.svc:4318"
	}
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(name),
			semconv.ServiceVersion(env("SERVICE_VERSION", "0.0.0")),
			attribute.String("k8s.namespace.name", env("NAMESPACE", "demo-shop")),
			attribute.String("deployment.environment", "local"),
		),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp.Shutdown, nil
}

func runTraffic(logger *slog.Logger) {
	target := env("TARGET_URL", "http://storefront/buy")
	every := 200 * time.Millisecond
	if raw := os.Getenv("INTERVAL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			every = parsed
		}
	}
	client := &http.Client{Timeout: 3 * time.Second}
	logger.Info("traffic generator", "target", target, "interval", every.String())
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for range ticker.C {
		res, err := client.Get(target)
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
		res.Body.Close()
	}
}

func percentDraw(percent int) bool {
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return false
	}
	return int(n.Int64()) < percent
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

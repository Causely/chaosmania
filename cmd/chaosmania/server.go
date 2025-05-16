package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/actions"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var LOGGER *zap.Logger

var processedTransactionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "chaosmania_processed_transactions_duration",
	Help: "The processed transactions duration",
})

var httpRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests by status code, method and path",
	},
	[]string{"status", "method", "path"},
)

func handleRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		start := time.Now()

		// Parse the JSON data from the request body
		var workload actions.Workload
		err := json.NewDecoder(r.Body).Decode(&workload)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			httpRequestsTotal.WithLabelValues("400", r.Method, r.URL.Path).Inc()
			httpRequestsTotal.WithLabelValues("4xx", r.Method, r.URL.Path).Inc()
			fmt.Fprintf(w, "error: %s", err)
			return
		}

		err = workload.Verify()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			httpRequestsTotal.WithLabelValues("400", r.Method, r.URL.Path).Inc()
			httpRequestsTotal.WithLabelValues("4xx", r.Method, r.URL.Path).Inc()
			fmt.Fprintf(w, "error: %s", err)
			return
		}

		// Create a response writer wrapper only for the workload execution
		rw := &responseWriter{ResponseWriter: w}
		ctx := context.WithValue(r.Context(), actions.ResponseWriterKey, rw)
		ctx = logger.NewContext(ctx, LOGGER)

		err = workload.Execute(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			httpRequestsTotal.WithLabelValues("500", r.Method, r.URL.Path).Inc()
			httpRequestsTotal.WithLabelValues("5xx", r.Method, r.URL.Path).Inc()
			fmt.Fprintf(w, "workload error: %s", err)
			return
		}

		// If no status code was set by the workload, default to 200
		if rw.statusCode == 0 {
			w.WriteHeader(http.StatusOK)
			httpRequestsTotal.WithLabelValues("200", r.Method, r.URL.Path).Inc()
			httpRequestsTotal.WithLabelValues("2xx", r.Method, r.URL.Path).Inc()
		} else {
			httpRequestsTotal.WithLabelValues(strconv.Itoa(rw.statusCode), r.Method, r.URL.Path).Inc()
			httpRequestsTotal.WithLabelValues(fmt.Sprintf("%dxx", rw.statusCode/100), r.Method, r.URL.Path).Inc()
		}

		fmt.Fprint(w, " ")
		processedTransactionDuration.Observe(float64(time.Since(start).Seconds()))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		httpRequestsTotal.WithLabelValues("405", r.Method, r.URL.Path).Inc()
		httpRequestsTotal.WithLabelValues("4xx", r.Method, r.URL.Path).Inc()
		fmt.Fprintf(w, "Error: Method not allowed")
	}
}

// responseWriter is a wrapper around http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	httpRequestsTotal.WithLabelValues("200", r.Method, r.URL.Path).Inc()
	httpRequestsTotal.WithLabelValues("2xx", r.Method, r.URL.Path).Inc()
}

func run(log *zap.Logger, port int64) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRequests)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleHealth)

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	mux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	server := &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: mux}
	go func() {
		log.Info(fmt.Sprintf("listening at %v", port))
		err := server.ListenAndServe()

		if err != nil {
			log.Warn("webserver error", zap.Error(err))
		}
	}()
}

func runWithOTEL(log *zap.Logger, port int64) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRequests)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	mux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	handler := otelhttp.NewHandler(mux, "server",
		otelhttp.WithSpanNameFormatter(func(operation string, req *http.Request) string {
			return req.URL.Path
		}),
		otelhttp.WithFilter(func(req *http.Request) bool {
			return req.URL.Path != "/health" &&
				req.URL.Path != "/metrics" &&
				req.URL.Path != "/debug/pprof/" &&
				req.URL.Path != "/debug/pprof/profile" &&
				req.URL.Path != "/debug/pprof/symbol" &&
				req.URL.Path != "/debug/pprof/trace"
		}),
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
	)

	server := &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: handler}
	go func() {
		log.Info(fmt.Sprintf("listening at %v", port))
		err := server.ListenAndServe()

		if err != nil {
			log.Warn("webserver error", zap.Error(err))
		}
	}()
}

func runWithDatadog(log *zap.Logger, port int64) {
	mux := httptrace.NewServeMux()
	mux.HandleFunc("/", handleRequests)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	mux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	server := &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: mux}
	go func() {
		log.Info(fmt.Sprintf("listening at %v", port))
		err := server.ListenAndServe()

		if err != nil {
			log.Warn("webserver error", zap.Error(err))
		}
	}()
}

func fileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	if err == nil {
		return true // File exists
	}
	if os.IsNotExist(err) {
		return false // File does not exist
	}
	return false // Error occurred while checking file existence
}

func command_server(log *zap.Logger, ctx *cli.Context) error {
	LOGGER = log

	// Signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	stop := make(chan struct{})

	go func() {
		sig := <-sigs
		log.Info("received signal", zap.String("signal", sig.String()))
		close(stop)
	}()

	// Load services if any
	if fileExists("/etc/chaosmania/services.yaml") {
		err := actions.Manager.LoadFromFile("/etc/chaosmania/services.yaml")
		if err != nil {
			log.Warn("failed to load services", zap.Error(err))
			return err
		}
	} else {
		log.Warn("no services.yaml found, not loading any services")
	}

	// Load background services if any
	if fileExists("/etc/chaosmania/background_services.yaml") {
		err := actions.BackgroundManager.LoadFromFile("/etc/chaosmania/background_services.yaml")
		if err != nil {
			log.Warn("failed to load services", zap.Error(err))
			return err
		}
	} else {
		log.Warn("no background_services.yaml found, not loading any services")
	}

	ctx2 := logger.NewContext(context.Background(), LOGGER)
	ctx2, cancel := context.WithCancel(ctx2)
	defer cancel()
	actions.BackgroundManager.Run(ctx2)

	// Add Go module build info.
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{Matcher: regexp.MustCompile("/.*")}),
	))

	port := ctx.Int64("port")

	if pkg.IsDatadogEnabled() {
		tracer.Start()
		defer tracer.Stop()

		runWithDatadog(log, port)
	} else if pkg.IsOpenTelemetryEnabled() {
		shutdown := InitOTLPProvider(log)
		defer shutdown()

		runWithOTEL(log, port)
	} else {
		run(log, port)
	}

	<-stop

	return nil
}

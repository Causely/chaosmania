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
	"syscall"
	"time"

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
)

type contextKey string

const responseWriterKey contextKey = "http.ResponseWriter"

var processedTransactionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "chaosmania_processed_transactions_duration",
	Help: "The processed transactions duration",
})

func command_server(log *zap.Logger, ctx *cli.Context) error {
	// Signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	stop := make(chan struct{})

	shutdown := InitOTLPProvider(log)
	defer shutdown()

	go func() {
		sig := <-sigs
		log.Info("received signal", zap.String("signal", sig.String()))
		close(stop)
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			start := time.Now()

			// Parse the JSON data from the request body
			var workload actions.Workload
			err := json.NewDecoder(r.Body).Decode(&workload)

			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "error: %s", err)
				return
			}

			err = workload.Verify()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "error: %s", err)
				return
			}

			ctx := context.WithValue(r.Context(), responseWriterKey, w)
			ctx = logger.NewContext(ctx, log)

			err = workload.Execute(ctx)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "workload error: %s", err)
				return
			}

			fmt.Fprint(w, " ")
			processedTransactionDuration.Observe(float64(time.Since(start).Seconds()))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Error: Method not allowed")
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Add Go module build info.
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{Matcher: regexp.MustCompile("/.*")}),
	))

	mux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	handler := otelhttp.NewHandler(mux, "",
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return r.URL.Path
		}),
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
	)

	port := ctx.Int64("port")
	server := &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: handler}
	go func() {
		log.Info(fmt.Sprintf("listening at %v", port))
		err := server.ListenAndServe()

		if err != nil {
			log.Warn("webserver error", zap.Error(err))
		}
	}()

	<-stop

	return nil
}

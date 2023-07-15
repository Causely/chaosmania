package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Causely/chaosmania/pkg/actions"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type contextKey string

const responseWriterKey contextKey = "http.ResponseWriter"

func command_server(log *zap.Logger, ctx *cli.Context) error {
	// Signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	stop := make(chan struct{})

	go func() {
		sig := <-sigs
		log.Info("received signal", zap.String("signal", sig.String()))
		close(stop)
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:

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

			err = workload.Execute(ctx)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "workload error: %s", err)
				return
			}

			fmt.Fprint(w, " ")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Error: Method not allowed")
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := ctx.Int64("port")
	server := &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: mux}
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

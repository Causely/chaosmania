package actions

import (
	"context"
	"fmt"
	"time"

	"database/sql"

	"github.com/Causely/chaosmania/pkg/logger"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var PQDBQueryHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "postgres_queries",
	Help: "The number of Postgres SQL queries",
})

type PostgresqlQuery struct{}

type PostgresqlQueryConfig struct {
	Query        string   `json:"query"`
	Repeat       int      `json:"repeat"`
	Host         string   `json:"host"`
	Port         int      `json:"port"`
	MaxOpen      int      `json:"maxopen"`
	MaxIdle      int      `json:"maxidle"`
	DBname       string   `json:"dbname"`
	User         string   `json:"user"`
	Password     string   `json:"password"`
	SSLMode      string   `json:"sslmode"`
	AppName      string   `json:"appname"`
	BurnDuration Duration `json:"burn_duration"`
}

var PGDRIVER = "postgres"

func (a *PostgresqlQuery) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[PostgresqlQueryConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	repeat := 1
	if config.Repeat != 0 {
		repeat = config.Repeat
	}
	host := "postgres"
	if config.Host != "" {
		host = config.Host
	}
	port := 5432
	if config.Port != 0 {
		port = config.Port
	}
	maxopen := 0
	if config.MaxOpen != 0 {
		maxopen = config.MaxOpen
	}
	maxidle := 0
	if config.MaxIdle != 0 {
		maxidle = config.MaxIdle
	}
	dbname := "postgres"
	if config.DBname != "" {
		dbname = config.DBname
	}
	user := "postgres"
	if config.User != "" {
		user = config.User
	}
	password := "postgres"
	if config.Password != "" {
		password = config.Password
	}
	sslmode := "disable"
	if config.SSLMode != "" {
		sslmode = config.SSLMode
	}
	appname := "chaosmania"
	if config.AppName != "" {
		appname = config.AppName
	}
	connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=%s application_name=%s", host, port, dbname, user, password, sslmode, appname)
	db, err := sql.Open(PGDRIVER, connStr)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to connect to DB", zap.Error(err))
		return err
	}

	db.SetMaxOpenConns(maxopen)
	db.SetMaxIdleConns(maxidle)

	defer db.Close()

	for i := 0; i < repeat; i++ {
		now := time.Now()
		rows, looperr := db.QueryContext(ctx, config.Query)
		if looperr != nil {
			logger.FromContext(ctx).Warn("failed to execute query", zap.Error(err))
			err = looperr
			continue
		}

		cols, err := rows.Columns()
		if err != nil {
			logger.FromContext(ctx).Error("failed to list columns", zap.Error(err))
		}

		// Create a slice to hold interface{} values for each column
		values := make([]interface{}, len(cols))
		for i := range values {
			values[i] = new(interface{})
		}

		for rows.Next() {
			// Scan the values of each column in the current row
			err := rows.Scan(values...)
			if err != nil {
				logger.FromContext(ctx).Error("failed to scan row", zap.Error(err))
			}

			// Print the values of each column
			for i, col := range cols {
				val := *(values[i].(*interface{}))
				logger.FromContext(ctx).Debug(fmt.Sprintf("%s: %v\n", col, val))
			}
		}

		PQDBQueryHistogram.Observe(float64(time.Since(now).Seconds()))
	}

	// Now burn cpu, if configured to do so
	if config.BurnDuration.Milliseconds() != 0 {
		logger.FromContext(ctx).Info("Running burn for: " + config.BurnDuration.String())
		end := time.Now().Add(config.BurnDuration.Duration)
		for time.Now().Before(end) {
		}
	}
	return err
}

func (a *PostgresqlQuery) ParseConfig(data map[string]any) (any, error) {
	return ParseConfig[PostgresqlQueryConfig](data)
}

func init() {
	ACTIONS["PostgresqlQuery"] = &PostgresqlQuery{}
}

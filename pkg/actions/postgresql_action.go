package actions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"database/sql"

	_ "github.com/lib/pq"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.nhat.io/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.uber.org/zap"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
)

var PQDBQueryHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "postgres_queries",
	Help: "The number of Postgres SQL queries",
})

var pgDbRegisterOnce sync.Once
var pgDriverName string

type PostgresqlQuery struct{}

type PostgresqlQueryConfig struct {
	Query         string `json:"query"`
	Repeat        int    `json:"repeat"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	MaxOpen       int    `json:"maxopen"`
	MaxIdle       int    `json:"maxidle"`
	DBname        string `json:"dbname"`
	User          string `json:"user"`
	Password      string `json:"password"`
	SSLMode       string `json:"sslmode"`
	AppName       string `json:"appname"`
	PeerService   string `json:"peer_service"`
	PeerNamespace string `json:"peer_namespace"`
}

func openPostgres(dsn string, host string, port int, dbname string) (*sql.DB, error) {
	var dn string
	var err error
	pgDbRegisterOnce.Do(func() {
		// Register the otelsql wrapper for the provided postgres driver.
		dn, err = otelsql.Register("postgres",
			otelsql.AllowRoot(),
			otelsql.TraceQueryWithoutArgs(),
			otelsql.TraceRowsClose(),
			otelsql.TraceRowsAffected(),
			otelsql.WithDatabaseName(dbname),               // Optional.
			otelsql.WithSystem(semconv.DBSystemPostgreSQL), // Optional.
			otelsql.WithDefaultAttributes(
				semconv.ServerAddress(host),
				semconv.ServerPort(port),
			),
		)
		if err == nil {
			pgDriverName = dn
		}
	})
	if pgDriverName == "" || err != nil {
		return nil, err
	}

	// Connect to a Postgres database using the postgres driver wrapper.
	return sql.Open(pgDriverName, dsn)
}

func (postgres *PostgresqlQuery) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[PostgresqlQueryConfig](cfg)
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

	var db *sql.DB
	if pkg.IsDatadogEnabled() {
		db, err = sqltrace.Open("postgres", connStr, sqltrace.WithServiceName(config.PeerService))
	} else {
		db, err = openPostgres(connStr, host, port, dbname)
	}

	if err != nil {
		logger.FromContext(ctx).Warn("failed to connect to DB", zap.Error(err))
		return err
	}

	db.SetMaxOpenConns(maxopen)
	db.SetMaxIdleConns(maxidle)

	defer db.Close()

	for i := 0; i < repeat; i++ {
		now := time.Now()
		rows, err := db.QueryContext(ctx, config.Query)
		if err != nil {
			logger.FromContext(ctx).Warn("failed to execute query", zap.Error(err))
			return err
		}

		cols, err := rows.Columns()
		if err != nil {
			logger.FromContext(ctx).Error("failed to list columns", zap.Error(err))
			return err
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
				return err
			}

			// Print the values of each column
			for i, col := range cols {
				val := *(values[i].(*interface{}))
				logger.FromContext(ctx).Debug(fmt.Sprintf("%s: %v\n", col, val))
			}
		}

		PQDBQueryHistogram.Observe(float64(time.Since(now).Seconds()))
	}

	return nil
}

func (postgres *PostgresqlQuery) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[PostgresqlQueryConfig](data)
}

func init() {
	ACTIONS["PostgresqlQuery"] = &PostgresqlQuery{}
}

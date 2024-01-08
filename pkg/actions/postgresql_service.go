package actions

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
)

type PostgresqlService struct {
	name   ServiceName
	config PostgresqlServiceConfig
	db     *sql.DB
}

type PostgresqlServiceConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	MaxOpen  int    `json:"maxopen"`
	MaxIdle  int    `json:"maxidle"`
	DBname   string `json:"dbname"`
	User     string `json:"user"`
	Password string `json:"password"`
	SSLMode  string `json:"sslmode"`
	AppName  string `json:"appname"`
}

func (s *PostgresqlService) Name() ServiceName {
	return s.name
}

func (s *PostgresqlService) Type() ServiceType {
	return "postgresql"
}

func (s *PostgresqlService) Query(ctx context.Context, query string) ([]map[string]any, error) {
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to execute query", zap.Error(err))
		return nil, err
	}

	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		logger.FromContext(ctx).Error("failed to list columns", zap.Error(err))
		return nil, err
	}

	result := make([]map[string]any, 0)

	for rows.Next() {
		values := make([]any, len(cols))
		for i := range values {
			values[i] = new(interface{})
		}

		// Scan the values of each column in the current row
		err := rows.Scan(values...)
		if err != nil {
			logger.FromContext(ctx).Error("failed to scan row", zap.Error(err))
			return nil, err
		}

		row := make(map[string]any)
		for i, col := range cols {
			row[col] = *(values[i].(*interface{}))
		}

		result = append(result, row)
	}

	return result, nil
}

func NewPostgresqlService(name ServiceName, config map[string]any) (Service, error) {
	cfg, err := pkg.ParseConfig[PostgresqlServiceConfig](config)
	if err != nil {
		return nil, err
	}

	postgresqlService := PostgresqlService{
		config: *cfg,
		name:   name,
	}

	host := "postgres"
	if cfg.Host != "" {
		host = cfg.Host
	}
	port := 5432
	if cfg.Port != 0 {
		port = cfg.Port
	}
	maxopen := 0
	if cfg.MaxOpen != 0 {
		maxopen = cfg.MaxOpen
	}
	maxidle := 0
	if cfg.MaxIdle != 0 {
		maxidle = cfg.MaxIdle
	}
	dbname := "postgres"
	if cfg.DBname != "" {
		dbname = cfg.DBname
	}
	user := "postgres"
	if cfg.User != "" {
		user = cfg.User
	}
	password := "postgres"
	if cfg.Password != "" {
		password = cfg.Password
	}
	sslmode := "disable"
	if cfg.SSLMode != "" {
		sslmode = cfg.SSLMode
	}
	appname := "chaosmania"
	if cfg.AppName != "" {
		appname = cfg.AppName
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=%s application_name=%s", host, port, dbname, user, password, sslmode, appname)

	var db *sql.DB
	if pkg.IsDatadogEnabled() {
		db, err = sqltrace.Open("postgres", connStr)
	} else {
		db, err = openPostgres(connStr, host, port, dbname)
	}

	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxopen)
	db.SetMaxIdleConns(maxidle)
	postgresqlService.db = db

	return &postgresqlService, nil
}

func init() {
	SERVICE_TPES["postgresql"] = func(name ServiceName, m map[string]any) Service {
		s, err := NewPostgresqlService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}

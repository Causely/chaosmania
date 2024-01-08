package actions

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"database/sql"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type MysqlService struct {
	name   ServiceName
	config MysqlServiceConfig
	db     *sql.DB
}

type MysqlServiceConfig struct {
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

func (s *MysqlService) Name() ServiceName {
	return s.name
}

func (s *MysqlService) Type() ServiceType {
	return "mysql"
}

func (s *MysqlService) Query(ctx context.Context, query string) ([]map[string]any, error) {
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

func NewMysqlService(name ServiceName, config map[string]any) (Service, error) {
	cfg, err := pkg.ParseConfig[MysqlServiceConfig](config)
	if err != nil {
		return nil, err
	}

	mysqlService := MysqlService{
		config: *cfg,
		name:   name,
	}

	host := "mysql"
	if cfg.Host != "" {
		host = cfg.Host
	}
	port := 3306
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
	dbname := "test"
	if cfg.DBname != "" {
		dbname = cfg.DBname
	}
	user := "admin"
	if cfg.User != "" {
		user = cfg.User
	}
	password := "password"
	if cfg.Password != "" {
		password = cfg.Password
	}
	sslmode := "preferred"
	if cfg.SSLMode != "" {
		sslmode = cfg.SSLMode
	}

	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?tls=%s", user, password, host, strconv.Itoa(port), dbname, sslmode)
	db, err := openMysql(connStr, host, port, dbname)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(maxopen)
	db.SetConnMaxIdleTime(time.Second * 10)
	db.SetMaxIdleConns(maxidle)

	mysqlService.db = db

	return &mysqlService, nil
}

func (a *MysqlService) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[MysqlServiceConfig](data)
}

func init() {
	SERVICE_TPES["mysql"] = func(name ServiceName, m map[string]any) Service {
		s, err := NewMysqlService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}

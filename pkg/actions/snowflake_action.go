package actions

import (
	"context"
	"fmt"

	"database/sql"

	"github.com/snowflakedb/gosnowflake"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type SnowflakeQuery struct{}

type SnowflakeQueryConfig struct {
	Query     string `json:"query"`
	Account   string `json:"account"`
	User      string `json:"user"`
	Password  string `json:"password"`
	Database  string `json:"database"`
	Warehouse string `json:"warehouse"`
	Schema    string `json:"schema"`
}

func (postgres *SnowflakeQuery) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[SnowflakeQueryConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	c := gosnowflake.Config{
		Account:   config.Account,
		User:      config.User,
		Password:  config.Password,
		Database:  config.Database,
		Warehouse: config.Warehouse,
	}

	connStr, err := gosnowflake.DSN(&c)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to create DSN", zap.Error(err))
		return err
	}

	db, err := sql.Open("snowflake", connStr)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to connect to DB", zap.Error(err))
		return err
	}

	defer db.Close()

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

	return nil
}

func (postgres *SnowflakeQuery) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[SnowflakeQueryConfig](data)
}

func init() {
	ACTIONS["SnowflakeQuery"] = &SnowflakeQuery{}
}

package actions

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"database/sql"

	"github.com/Causely/chaosmania/pkg/logger"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var MYDBQueryHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "mysql_queries",
	Help: "The number of MySQL queries",
})

type MysqlQuery struct{}

type MysqlQueryConfig struct {
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

var MYDRIVER = "mysql"

func (a *MysqlQuery) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[MysqlQueryConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	repeat := 1
	if config.Repeat != 0 {
		repeat = config.Repeat
	}
	host := "mysql"
	if config.Host != "" {
		host = config.Host
	}
	port := 3306
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
	dbname := "test"
	if config.DBname != "" {
		dbname = config.DBname
	}
	user := "admin"
	if config.User != "" {
		user = config.User
	}
	password := "password"
	if config.Password != "" {
		password = config.Password
	}
	sslmode := "preferred"
	if config.SSLMode != "" {
		sslmode = config.SSLMode
	}
	//appname := "chaosmania"
	//if config.AppName != "" {
	//	appname = config.AppName
	//}

	//mysqlConfig := &mysql.Config{
	//	User:      user,
	//	Passwd:    password,
	//	Net:       "tcp",
	//	Addr:      host + ":" + strconv.Itoa(port),
	//	DBName:    dbname,
	//	TLSConfig: sslmode,
	//	Params: map[string]string{
	//		"application_name": appname,
	//	},
	//}
	//db, err := sql.Open(MYDRIVER, mysqlConfig.FormatDSN())
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?tls=%s", user, password, host, strconv.Itoa(port), dbname, sslmode)
	db, err := sql.Open(MYDRIVER, connStr)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to connect to DB", zap.Error(err))
		return err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(maxopen)
	db.SetConnMaxIdleTime(time.Second * 10)
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

		MYDBQueryHistogram.Observe(float64(time.Since(now).Seconds()))
	}

	// Now burn cpu, if configured to do so
	if config.BurnDuration.Milliseconds() != 0 {
		logger.FromContext(ctx).Info("Running burn for: " + config.BurnDuration.String())
		end := time.Now().Add(config.BurnDuration.Duration)
		for time.Now().Before(end) {
		}
	}
	return nil
}

func (a *MysqlQuery) ParseConfig(data map[string]any) (any, error) {
	return ParseConfig[MysqlQueryConfig](data)
}

func init() {
	ACTIONS["MysqlQuery"] = &MysqlQuery{}
}

package actions

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"database/sql"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.nhat.io/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.uber.org/zap"
)

var MYDBQueryHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "mysql_queries",
	Help: "The number of MySQL queries",
})

var sqlDbRegisterOnce sync.Once
var sqlDriverName string

type MysqlQuery struct{}

type MysqlQueryConfig struct {
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

func openMysql(dsn string, host string, port int, dbname string) (*sql.DB, error) {
	var dn string
	var err error
	sqlDbRegisterOnce.Do(func() {
		// Register the otelsql wrapper for the provided mysql driver.
		dn, err = otelsql.Register("mysql",
			otelsql.AllowRoot(),
			otelsql.TraceQueryWithoutArgs(),
			otelsql.TraceRowsClose(),
			otelsql.TraceRowsAffected(),
			otelsql.WithDatabaseName(dbname),
			otelsql.WithSystem(semconv.DBSystemMySQL),
			otelsql.WithDefaultAttributes(
				semconv.ServerAddress(host),
				semconv.ServerPort(port),
			),
		)
		if err == nil {
			sqlDriverName = dn
		}
	})

	if sqlDriverName == "" || err != nil {
		return nil, err
	}

	// Connect to a Mysql database using the mysql driver wrapper.
	return sql.Open(sqlDriverName, dsn)
}

func (mysql *MysqlQuery) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[MysqlQueryConfig](cfg)
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
	db, err := openMysql(connStr, host, port, dbname)
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
			logger.FromContext(ctx).Warn("failed to execute query", zap.Error(looperr))
			return looperr
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

	return nil
}

func (mysql *MysqlQuery) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[MysqlQueryConfig](data)
}

func init() {
	ACTIONS["MysqlQuery"] = &MysqlQuery{}
}

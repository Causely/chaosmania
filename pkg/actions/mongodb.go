package actions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.uber.org/zap"

	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

var MongoDBQueryHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "mongodb_queries",
	Help: "The number of MongoDB queries",
})

type MongoDBQuery struct {
	mongoClient *mongo.Client
	config      *MongoDbQueryConfig
	clientOpts  *options.ClientOptions
}

type MongoDBConnection struct {
	ConnScheme          string                 `json:"conn_scheme"`
	Hosts               []string               `json:"hosts"`
	Port                int                    `json:"port"`
	User                string                 `json:"user"`
	Password            string                 `json:"password"`
	SSLMode             bool                   `json:"sslmode;default:false"`
	ReplicaSet          string                 `json:"replica_set"`
	OpTimeoutMS         *time.Duration         `json:"op_timeout_ms;default:nil"`
	ConnectingTimeoutMs *time.Duration         `json:"connecting_timeout_ms;default:nil"`
	UseClientOptsOnly   bool                   `json:"use_client_opts_only;default:false"`
	ClientOptions       *options.ClientOptions `json:"client_options:default:nil"`
}

type MongoDbQueryConfig struct {
	AppName    string            `json:"appname;default:chaosmania"`
	Query      map[string]any    `json:"query"`
	DBname     string            `json:"dbname"`
	Connection MongoDBConnection `json:"connection"`

	MaxOpen      int64         `json:"maxopen"`
	MaxIdle      time.Duration `json:"maxidle"`
	Repeat       int           `json:"repeat"`
	BurnDuration pkg.Duration  `json:"burn_duration"`
}

func buildClientOpts(ctx context.Context, config MongoDbQueryConfig) *options.ClientOptions {
	connectionCfg := config.Connection
	var clientOpts *options.ClientOptions

	if connectionCfg.UseClientOptsOnly {
		clientOpts = connectionCfg.ClientOptions
	} else {
		clientOpts = options.Client()
		var strBuilder strings.Builder

		// mongodb://myDatabaseUser:D1fficultP%40ssw0rd@mongodb0.example.com:27017/?authSource=admin
		// mongodb://myDatabaseUser:D1fficultP%40ssw0rd@db0.example.com,db1.example.com,db2.example.com/?replicaSet=myRepl&ssl=true
		// mongodb://myDatabaseUser:D1fficultP%40ssw0rd@db0.example.com:27017,db1.example.com:27017,db2.example.com:27017/?replicaSet=myRepl

		connScheme := connstring.SchemeMongoDB
		if connectionCfg.ConnScheme != "" {
			connScheme = connectionCfg.ConnScheme
		}
		strBuilder.WriteString(connScheme + "://")

		user := ""
		if connectionCfg.User != "" {
			user = connectionCfg.User
		}
		password := ""
		if connectionCfg.Password != "" {
			password = connectionCfg.Password
		}
		if user != "" && password != "" {
			strBuilder.WriteString(user + ":" + password + "@")
		}

		port := 27017
		if connectionCfg.Port != 0 {
			port = connectionCfg.Port
		}

		// in the case of replica-sets we may want to provide N-number of comma-separated `host:port` pairs
		// the uri options should also include `replicaSet=<myRepl>` key-value pair
		// e.g.: `mongodb://user:pwd@db0.example.com:27017,db1.example.com:27017,db2.example.com:27017/?replicaSet=myRepl`
		hosts := []string{"localhost"}
		if len(connectionCfg.Hosts) != 0 {
			hosts = connectionCfg.Hosts
		}
		lenHosts := len(hosts)
		for idx, host := range hosts {
			strBuilder.WriteString(host + ":" + strconv.Itoa(port) + "/")
			if idx < lenHosts-1 {
				strBuilder.WriteString(",")
			}
		}

		// Start URI Options builder (?key=value&key2=value2 format)
		uriOptions := make([]string, 0)
		if connectionCfg.SSLMode {
			uriOptions = append(uriOptions, "ssl=true")
		}
		if connectionCfg.ReplicaSet != "" {
			uriOptions = append(uriOptions, "replicaSet="+connectionCfg.ReplicaSet)
		}

		if len(uriOptions) > 0 {
			strBuilder.WriteString("?")
			strBuilder.WriteString(strings.Join(uriOptions, "&"))
		}

		if connectionCfg.OpTimeoutMS != nil {
			clientOpts.SetTimeout(*connectionCfg.OpTimeoutMS)
		}

		if connectionCfg.ConnectingTimeoutMs != nil {
			clientOpts.SetConnectTimeout(*connectionCfg.ConnectingTimeoutMs)
		}

		clientOpts.ApplyURI(strBuilder.String())

		clientOpts.SetAppName(config.AppName)

		maxopen := int64(0)
		if config.MaxOpen != 0 {
			maxopen = config.MaxOpen
		}
		clientOpts.SetMaxConnecting(uint64(maxopen))

		maxidle := time.Duration(0)
		if config.MaxIdle != 0 {
			maxidle = config.MaxIdle
		}
		clientOpts.SetMaxConnIdleTime(maxidle)
	}

	logger.FromContext(ctx).Debug("Connecting to MongoDB: ",
		zap.Bool("ClientOptsCfgOnly", connectionCfg.UseClientOptsOnly),
		zap.String("ClientOptions", clientOpts.GetURI()),
	)

	return clientOpts
}

func (a *MongoDBQuery) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[MongoDbQueryConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}
	a.config = config

	a.clientOpts = buildClientOpts(ctx, *config)

	a.clientOpts.Monitor = otelmongo.NewMonitor()

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for {
		client, err := mongo.Connect(ctx, a.clientOpts)
		if err != nil {
			logger.FromContext(ctx).Error("failed to connect to mongodb", zap.Error(err))
		}
		a.mongoClient = client
		break
	}

	dbname := "admin"
	if config.DBname != "" {
		dbname = config.DBname
	}

	db := a.mongoClient.Database(dbname)

	defer func(client *mongo.Client, ctx context.Context) {
		err := client.Disconnect(ctx)
		if err != nil {
			logger.FromContext(ctx).Warn("failed to disconnect from mongodb", zap.Error(err))
		}
	}(db.Client(), ctx)

	repeat := 1
	if config.Repeat != 0 {
		repeat = config.Repeat
	}

	var cursor *mongo.Cursor
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			logger.FromContext(ctx).Error("failed to close cursor", zap.Error(err))
		}
	}(cursor, context.TODO())
	for i := 0; i < repeat; i++ {
		now := time.Now()
		//command := bson.D{
		//	{"find", "orders"},   // The command to find documents in the collection
		//	{"filter", bson.D{}}, // The filter to apply (empty here for all documents)
		//}
		var query bson.D
		for key, val := range config.Query {
			query = append(query, bson.E{Key: key, Value: val})
		}
		command, err := bson.Marshal(query)
		if err != nil {
			logger.FromContext(ctx).Warn("failed to marshal query", zap.Error(err))
			return err
		}

		cursor, err = db.RunCommandCursor(ctx, command)
		if err != nil {
			logger.FromContext(ctx).Warn("failed to execute query", zap.Error(err))
			return err
		}

		var results []bson.M
		for cursor.Next(context.TODO()) {
			var result bson.M
			if err := cursor.Decode(&result); err != nil {
				logger.FromContext(ctx).Error("failed to decode cursor", zap.Error(err))
				return err
			}
			results = append(results, result)
		}

		// Check for errors during cursor iteration
		if err := cursor.Err(); err != nil {
			logger.FromContext(ctx).Error("failed to iterate cursor", zap.Error(err))
			return err
		}

		// Print the results
		for idx, element := range results {
			logger.FromContext(ctx).Debug(fmt.Sprintf("document %d ---> %v:", idx, element))
		}

		MongoDBQueryHistogram.Observe(time.Since(now).Seconds())
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

func (a *MongoDBQuery) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[MongoDbQueryConfig](data)
}

func init() {
	ACTIONS["MongodbQuery"] = &MongoDBQuery{}
}

package main

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/urfave/cli/v2"
)

func main() {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.InfoLevel)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	defer func() {
		_ = logger.Sync()
	}()

	app := &cli.App{
		Name:  "chaosmania",
		Usage: "chaosmania client|server",
		Commands: []*cli.Command{{
			Name: "client",
			Action: func(ctx *cli.Context) error {
				return command_client(logger, ctx)
			},
			Flags: []cli.Flag{
				&cli.PathFlag{
					Name:     "plan",
					Aliases:  []string{"p"},
					Usage:    "Path to the execution plan",
					FilePath: "./plan.json",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "host",
					Usage:    "Host",
					Required: true,
				},
				&cli.Int64Flag{
					Name:     "port",
					Usage:    "Pod",
					Required: true,
				},
				&cli.StringSliceFlag{
					Name:     "header",
					Usage:    "Headers to include in the request",
					Required: false,
				},
				&cli.StringFlag{
					Name:     "total-duration",
					Usage:    "Total duration for the test run (e.g. '1h', '30m', '2h30m').  Plans with multiple phases will be scaled to fit this duration.",
					Required: false,
				},
				&cli.IntFlag{
					Name:     "max-repeats",
					Usage:    "Maximum number of phase repeats to execute (0 for unlimited, max 500)",
					Required: false,
					Value:    0,
				},
			},
		}, {
			Name: "server",
			Action: func(ctx *cli.Context) error {
				return command_server(logger, ctx)
			},
			Flags: []cli.Flag{
				&cli.Int64Flag{
					Name:     "port",
					Usage:    "Pod",
					Required: true,
				},
			},
		}},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal("failed: ", zap.Error(err))
	}
}

package jklserver

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"strconv"
	"time"
)

type Config struct {
	IP						string

	Separator 				string
	LifeTimeConnection 		time.Duration
	TimeShutdown			time.Duration
	BufSize					uint16

	PathToFolderLogFiles	string
}

func Default() *Config {
	return &Config{
		IP: "0.0.0.0",
		Separator: "\n",
		BufSize: 1024,
		LifeTimeConnection: 5 * time.Second,
		TimeShutdown: 60 * time.Second,
	}
}

func New() *Config {
	return &Config{
	}
}

func newLogger(path string) *zap.Logger {
	cfg := zap.Config{}

	if path != "" {
		logStartTime := time.Now()
		logName := strconv.Itoa(logStartTime.Day()) + "-" + strconv.Itoa(int(logStartTime.Month())) + "-" + strconv.Itoa(logStartTime.Year()) + ".log"
		cfg = zap.Config {
			Encoding:         "console",
			Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
			OutputPaths:      []string{"stderr", path + logName},
			ErrorOutputPaths: []string{"stderr", path + "ERR-" + logName},
			EncoderConfig: zapcore.EncoderConfig{
				MessageKey: "msg",

				LevelKey:    "level",
				EncodeLevel: zapcore.CapitalLevelEncoder,

				TimeKey:    "time",
				EncodeTime: zapcore.RFC3339TimeEncoder,

				CallerKey:    "caller",
				EncodeCaller: zapcore.ShortCallerEncoder,
			},
		}
	} else {
		cfg = zap.Config{
			Encoding: "console",
			Level:    zap.NewAtomicLevelAt(zapcore.DebugLevel),
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
			EncoderConfig: zapcore.EncoderConfig{
				MessageKey: "msg",

				LevelKey:    "level",
				EncodeLevel: zapcore.CapitalLevelEncoder,

				TimeKey:    "time",
				EncodeTime: zapcore.RFC3339TimeEncoder,

				//CallerKey:    "caller",
				//EncodeCaller: zapcore.ShortCallerEncoder,
			},
		}
	}

	logger, err := cfg.Build()
	if err != nil {
		log.Fatal(err)
	}

	return logger
}

func (cfg *Config) Build() *server {
	srv := newServer()

	logger := newLogger(cfg.PathToFolderLogFiles)

	srv.logger = logger
	srv.Config = cfg

	srv.build()

	return srv
}

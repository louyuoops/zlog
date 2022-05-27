package zlog

import (
	"fmt"
	"io/ioutil"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	yaml "gopkg.in/yaml.v2"
)

var (
	Zlogger *zap.Logger
	Sugar   *zap.SugaredLogger
)

type ZlogConfig struct {
	LumerConfig LumerConfig `yaml:"LumerConfig"`
	ZapConfig   ZapConfig   `yaml:"ZapConfig"`
}

type LumerConfig struct {
	Filename     string `yaml:"Filename"`
	WarnFilename string `yaml:"WarnFilename"`
	MaxSize      int    `yaml:"MaxSize"`
	MaxBackups   int    `yaml:"MaxBackups"`
	MaxAge       int    `yaml:"MaxAge"`
	Compress     bool   `yaml:"Compress"`
}

type ZapConfig struct {
	TimeKey       string `yaml:"TimeKey"`
	LevelKey      string `yaml:"LevelKey"`
	NameKey       string `yaml:"NameKey"`
	CallerKey     string `yaml:"CallerKey"`
	MessageKey    string `yaml:"MessageKey"`
	StacktraceKey string `yaml:"StacktraceKey"`
	ServiceName   string `yaml:"ServiceName"`
}

func RegisterLogger(logger *zap.Logger, sugar *zap.SugaredLogger) {
	logger = __initLogger()
	sugar = logger.Sugar()
	Zlogger = logger
	Sugar = sugar
}

func __initLogger() *zap.Logger {
	config := __initConfig()

	lumberCommonHook := lumberjack.Logger{
		Filename:   config.LumerConfig.Filename,
		MaxSize:    config.LumerConfig.MaxSize,
		MaxBackups: config.LumerConfig.MaxBackups,
		MaxAge:     config.LumerConfig.MaxAge,
		Compress:   config.LumerConfig.Compress,
	}

	lumberErrorHook := lumberjack.Logger{
		Filename:   config.LumerConfig.WarnFilename,
		MaxSize:    config.LumerConfig.MaxSize,
		MaxBackups: config.LumerConfig.MaxBackups,
		MaxAge:     config.LumerConfig.MaxAge,
		Compress:   config.LumerConfig.Compress,
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     config.ZapConfig.MessageKey,
		LevelKey:       config.ZapConfig.LevelKey,
		TimeKey:        config.ZapConfig.TimeKey,
		NameKey:        config.ZapConfig.NameKey,
		CallerKey:      config.ZapConfig.CallerKey,
		StacktraceKey:  config.ZapConfig.StacktraceKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	commonLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level <= zapcore.InfoLevel
	})
	errorLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zapcore.WarnLevel
	})

	commonCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&lumberCommonHook)),
		commonLevel,
	)
	errorCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&lumberErrorHook)),
		errorLevel,
	)

	caller := zap.AddCaller()
	development := zap.Development()
	filed := zap.Fields(zap.String("serviceName", config.ZapConfig.ServiceName))
	logger := zap.New(zapcore.NewTee(commonCore, errorCore), caller, development, filed)
	return logger
}

func __initConfig() *ZlogConfig {
	config := new(ZlogConfig)
	yamlConfFile, err := ioutil.ReadFile("./conf/zlog_config.yaml")
	if err != nil {
		fmt.Println("init zlog_config file failed;", err)
		panic("read zlog config failed")
	}
	err = yaml.Unmarshal(yamlConfFile, config)
	if err != nil {
		fmt.Println("failed to decode yaml zlog config")
		panic("unmarshal yaml zlog config failed")
	}
	__initLogDir(config.LumerConfig.Filename)
	return config
}

func __initLogDir(filename string) {
	_, err := os.Stat("./logs")
	if err == nil {
		_, err = os.Stat(filename)
		if os.IsNotExist(err) {
			_, err = os.Create(filename)
			if err != nil {
				panic("create log file failed")
			}
		}
	}
	if os.IsNotExist(err) {
		os.Mkdir("./logs", 0777)
		_, err = os.Create(filename)
		if err != nil {
			panic("create log file failed")
		}
	}
}

// package method
func Info(args ...interface{}) {
	Sugar.Info(args)
}

func Infow(msg string, args ...interface{}) {
	Sugar.Infow(msg, args)
}

func Debug(args ...interface{}) {
	Sugar.Debug(args)
}

func Debugw(msg string, args ...interface{}) {
	Sugar.Debugw(msg, args)
}

func Warn(args ...interface{}) {
	Sugar.Warn(args)
}

func Warnw(msg string, args ...interface{}) {
	Sugar.Warnw(msg, args)
}
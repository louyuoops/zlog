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
	LumberConfig LumberConfig `yaml:"LumerConfig"`
	ZapConfig    ZapConfig    `yaml:"ZapConfig"`
}

type LumberConfig struct {
	FilePath     string `yaml:"FilePath"`
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

func InitZlog(cfgFile string) {
	if Zlogger != nil && Sugar != nil {
		return
	}

	var zlogConfig = &ZlogConfig{}
	if len(cfgFile) > 0 {
		zlogConfig = __initConfig(cfgFile)
	}

	lumberConfig, zapConfig := &zlogConfig.LumberConfig, &zlogConfig.ZapConfig
	if lumberConfig == nil {
		lumberConfig = &LumberConfig{}
	}
	if zapConfig == nil {
		zapConfig = &ZapConfig{}
	}

	lumberCommonHook := lumberjack.Logger{
		Filename:   lumberConfig.Filename,
		MaxSize:    lumberConfig.MaxSize,
		MaxBackups: lumberConfig.MaxBackups,
		MaxAge:     lumberConfig.MaxAge,
		Compress:   lumberConfig.Compress,
	}

	lumberErrorHook := lumberjack.Logger{
		Filename:   lumberConfig.WarnFilename,
		MaxSize:    lumberConfig.MaxSize,
		MaxBackups: lumberConfig.MaxBackups,
		MaxAge:     lumberConfig.MaxAge,
		Compress:   lumberConfig.Compress,
	}

	zap.RegisterEncoder("textEncoder", getTextEncoder)

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "tag",
		NameKey:        "zlog",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	infoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level <= zapcore.InfoLevel
	})
	errorLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zapcore.WarnLevel
	})

	var infoCore, errorCore zapcore.Core

	if len(lumberCommonHook.Filename) > 0 {
		infoCore = zapcore.NewCore(
			NewtextEncoder(encoderConfig),
			zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&lumberCommonHook)),
			infoLevel,
		)
	} else {
		infoCore = zapcore.NewCore(
			NewtextEncoder(encoderConfig),
			zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)),
			infoLevel,
		)
	}

	if len(lumberErrorHook.Filename) > 0 {
		errorCore = zapcore.NewCore(
			NewtextEncoder(encoderConfig),
			zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&lumberErrorHook)),
			errorLevel,
		)
	} else {
		errorCore = zapcore.NewCore(
			NewtextEncoder(encoderConfig),
			zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)),
			errorLevel,
		)
	}

	caller := zap.AddCaller()
	// development := zap.Development()
	field := zap.Fields(zap.String("G_SERV_NAME", zapConfig.ServiceName))
	callStep := zap.AddCallerSkip(1)
	logger := zap.New(zapcore.NewTee(infoCore, errorCore), caller, callStep, field)
	Zlogger = logger
	Sugar = logger.Sugar()
}

func __initConfig(cfg string) *ZlogConfig {
	config := new(ZlogConfig)
	yamlConfFile, err := ioutil.ReadFile(cfg)
	if err != nil {
		panic("read zlog config failed")
	}
	err = yaml.Unmarshal(yamlConfFile, config)
	if err != nil {
		panic("unmarshal yaml zlog config failed")
	}
	config.LumberConfig.Filename = fmt.Sprintf("%s/%s", config.LumberConfig.FilePath, config.LumberConfig.Filename)
	config.LumberConfig.WarnFilename = fmt.Sprintf("%s/%s", config.LumberConfig.FilePath, config.LumberConfig.WarnFilename)
	__initLogDir(config.LumberConfig.FilePath, config.LumberConfig.Filename, config.LumberConfig.WarnFilename)
	return config
}

func __initLogDir(path string, notice string, warn string) {
	if !__isExist(path) {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			panic("create log directory failed")
		}
	}
	if !__isExist(notice) {
		_, err := os.Create(notice)
		if err != nil {
			panic("create notice file failed")
		}
	}
	if !__isExist(warn) {
		_, err := os.Create(warn)
		if err != nil {
			panic("create warnning file failed")
		}
	}
}

func __isExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

type Arg = zapcore.Field

func Debug(tag string, args []Arg) {
	Zlogger.Debug(tag, args...)
}

func Info(tag string, args []Arg) {
	Zlogger.Info(tag, args...)
}

func Warn(tag string, args []Arg) {
	Zlogger.Warn(tag, args...)
}

func Panic(tag string, args []Arg) {
	Zlogger.Panic(tag, args...)
}

func Fatal(tag string, args []Arg) {
	Zlogger.Fatal(tag, args...)
}

// sugar
func Debugw(tag string, args ...interface{}) {
	Sugar.Debugw(tag, args...)
}

func Infow(tag string, args ...interface{}) {
	Sugar.Infow(tag, args...)
}

func Warnw(tag string, args ...interface{}) {
	Sugar.Warnw(tag, args...)
}

func Panicw(tag string, args ...interface{}) {
	Sugar.Panicw(tag, args...)
}

func Fatalw(tag string, args ...interface{}) {
	Sugar.Fatalw(tag, args...)
}

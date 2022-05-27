package gotest

import (
	"testing"

	zlog "gitee.com/lrtxpra/zlog/wrapper"
	"go.uber.org/zap"
)

type People struct {
	Name   string
	Gender int
	Age    int
	School string
	Addr   Addr
}

type Addr struct {
	Province string
	City     string
	County   string
}

var person = People{
	Name:   "mike",
	Gender: 1,
	Age:    30,
	School: "BeiJing University",
	Addr: Addr{
		Province: "BeiJing",
		City:     "BeiJing",
		County:   "HaiDian",
	},
}

// test zap logger
func TestZlogLogger(t *testing.T) {
	zlog.InitZlog("./zlog_config.yaml")
	t.Log("test case")
	zlog.Info("msg_tag", []zlog.Arg{
		zap.String("msg", "this is a info msg"),
		zap.Reflect("person", person),
	})
	zlog.Debug("msg_tag", []zlog.Arg{
		zap.String("msg", "this is a debug msg"),
		zap.Reflect("person", person),
	})
	zlog.Warn("msg_tag", []zlog.Arg{
		zap.String("msg", "this is a warn msg"),
		zap.Reflect("person", person),
	})
	// fatal error exit
	// zlog.Fatal("msg_tag", []zlog.Arg{
	// 	zap.String("msg", "this is a Fatal msg"),
	// 	zap.Reflect("person", person),
	// })
	// panic exit
	// zlog.Panic("msg_tag", []zlog.Arg{
	// 	zap.String("msg", "this is a panic msg"),
	// 	zap.Reflect("person", person),
	// })
}

func TestZlogSugar(t *testing.T) {
	zlog.InitZlog("./zlog_config.yaml")
	t.Log("test case")
	zlog.Infow("msg_tag", "msg", "this is a info msg", "person", person)
	zlog.Debugw("msg_tag", "msg", "this is a debug msg", "person", person)
	zlog.Warnw("msg_tag", "msg", "this is a warn msg", "person", person)
	// zlog.Fatalw("msg_tag", "msg", "this is a fatal msg", "person", person)
	// zlog.Panicw("msg_tag", "msg", "this is a panic msg", "person", person)
}

package gotest

import (
	"testing"

	"gitee.com/lrtxpra/zlog/wrapper/zlog"
	"go.uber.org/zap"
)

func TestZlogNotice(t *testing.T) {
	t.Log("test case")
	zlog.Info("asd", zap.String("123", "123"))
}

wrapper of uber zap logger

config file example：
```
LumerConfig:
    FilePath: "./logs"
    Filename: "sticker.access"
    WarnFilename: "sticker.warn"
    MaxSize: 500
    MaxBackups: 24
    MaxAge: 7
    Compress: true

ZapConfig:
    ServiceName: "sticker"
```

when import current package to your project, you need to init zlog first.
```
import (
    zlog "gitee.com/lrtxpra/zlog/wrapper" 
)

func xxx() {
    // init logger with config yaml file
    zlog.InitZlog("./zlog_config.yaml")
}
```

usage reference to zlog_test.go
package main

import (
	"fmt"
	"github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"time"
)

const (
	logDir    = "logs/"
	accessLog = "access.log"
	errorLog  = "error.log"
	pidFile   = "running.pid"
)

func main() {
	logger := SetupLogger(logDir, accessLog, errorLog)

	pid := PidSetup{
		PidFile: pidFile,
	}

	pid.CreatePidFile(logger)

	logger.Info("Info", zap.String("11", "23"))

	time.Sleep(time.Second * 2)
	pid.RemovePidFile()
}

func SetupLogger(logDir, accessLog, errorLog string) (logger *zap.Logger) {
	// 设置一些基本日志格式 具体含义还比较好理解，直接看zap源码也不难懂
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.CapitalLevelEncoder,
		TimeKey:     "ts",
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},
		CallerKey:    "file",
		EncodeCaller: zapcore.ShortCallerEncoder,
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},
	})

	// 实现两个判断日志等级的interface (其实 zapcore.*Level 自身就是 interface)
	infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.WarnLevel
	})

	warnLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.WarnLevel
	})

	// 获取 info、warn日志文件的io.Writer 抽象 getWriter() 在下方实现
	infoWriter := getWriter(logDir + accessLog)
	warnWriter := getWriter(logDir + errorLog)

	// 最后创建具体的Logger
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(infoWriter), infoLevel),
		zapcore.NewCore(encoder, zapcore.AddSync(warnWriter), warnLevel),
	)

	logger = zap.New(core, zap.AddCaller()) // 需要传入 zap.AddCaller() 才会显示打日志点的文件名和行数, 有点小坑
	return
}

func getWriter(filename string) io.Writer {
	// 生成rotatelogs的Logger 实际生成的文件名 demo.log.YYmmddHH
	// demo.log是指向最新日志的链接
	// 保存7天内的日志，每1小时(整点)分割一次日志
	hook, err := rotatelogs.New(
		filename+".%Y%m%d%H", // 没有使用go风格反人类的format格式
		rotatelogs.WithLinkName(filename),
		rotatelogs.WithMaxAge(time.Hour*24*7),
		rotatelogs.WithRotationTime(time.Hour),
	)

	if err != nil {
		panic(err)
	}
	return hook
}

type PidSetup struct {
	PidFile string
}

func (pid *PidSetup) CreatePidFile(logger *zap.Logger) {
	f, err := os.OpenFile(pid.PidFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("failed to create pid file")
	}
	io.WriteString(f, fmt.Sprintf("%d", os.Getpid()))
	f.Close()
	logger.Info("Server ", zap.Int("pid", os.Getpid()))
}

func (pid *PidSetup) RemovePidFile() {
	os.Remove(pid.PidFile)
}
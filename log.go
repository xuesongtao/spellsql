package spellsql

import (
	"context"
	"log"
	"os"
	"runtime"
	"strconv"
)

type defaultLogger struct {
	log *log.Logger
}

func NewLogger() *defaultLogger {
	return &defaultLogger{
		log: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (d *defaultLogger) Info(ctx context.Context, v ...interface{}) {
	d.log.Println(append([]interface{}{"[INFO] " + d.getPrefix(3)}, v...)...)
}

func (d *defaultLogger) Error(ctx context.Context, v ...interface{}) {
	d.log.Println(append([]interface{}{"[ERRO] " + d.getPrefix(3)}, v...)...)
}

func (d *defaultLogger) Warning(ctx context.Context, v ...interface{}) {
	d.log.Println(append([]interface{}{"[WARN] " + d.getPrefix(3)}, v...)...)
}

func (d *defaultLogger) getPrefix(skip int) string {
	file, line := d.callInfo(skip)
	return file + ":" + strconv.Itoa(line)
}

func (d *defaultLogger) callInfo(skip int) (string, int) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", 0
	}
	file = parseFileName(file)
	return file, line
}

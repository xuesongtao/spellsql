package spellsql

import (
	"context"
	logOs "os"
	"runtime"
	"strconv"
	
	"log"
	// slog "gitlab.cd.anpro/go/common/log"
	// "errors"
)

type defaultLogger struct {
	log *log.Logger
}

func NewLogger() *defaultLogger {
	return &defaultLogger{
		log: log.New(logOs.Stdout, "", log.LstdFlags),
	}
}

func (d *defaultLogger) Info(ctx context.Context, v ...interface{}) {
	d.log.Println(append([]interface{}{"[INFO] " + d.getPrefix(3)}, v...)...)
	// slog.InfofWithTrace(ctx, d.getFormat(v...), v...)
}

func (d *defaultLogger) Error(ctx context.Context, v ...interface{}) {
	d.log.Println(append([]interface{}{"[ERRO] " + d.getPrefix(3)}, v...)...)
	// slog.ErrorfWithTrace(ctx, errors.New("sql handle err"), d.getFormat(v...), v...)
}

func (d *defaultLogger) Warning(ctx context.Context, v ...interface{}) {
	d.log.Println(append([]interface{}{"[WARN] " + d.getPrefix(3)}, v...)...)
	// slog.WarnfWithTrace(ctx, d.getFormat(v...), v...)
}

func (d *defaultLogger) getFormat(v ...interface{}) (formatStr string) {
	l := len(v)
	for i := 0; i < l; i++ {
		if formatStr == "" {
			formatStr = "%v"
			continue
		}
		formatStr += " %v"
	}
	return
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

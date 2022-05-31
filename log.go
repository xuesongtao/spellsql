package spellsql

import (
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	cjLog Logger
	once  sync.Once
)

func init() {
	cjLog = NewCjLogger()
}

// SetLogger 设置 logger
func SetLogger(logger Logger) {
	once.Do(func() {
		cjLog = logger
	})
}

type defaultLogger struct {
	log *log.Logger
}

func NewCjLogger() *defaultLogger {
	return &defaultLogger{
		log: log.New(os.Stderr, "", log.LstdFlags),
	}
}

func (d *defaultLogger) Info(v ...interface{}) {
	d.log.Println(append([]interface{}{"[INFO]"}, v...)...)
}

func (d *defaultLogger) Infof(format string, v ...interface{}) {
	d.log.Printf("[INFO] "+format, v...)
}

func (d *defaultLogger) Error(v ...interface{}) {
	d.log.Println(append([]interface{}{"[ERRO]"}, v...)...)
}

func (d *defaultLogger) Errorf(format string, v ...interface{}) {
	d.log.Printf("[ERRO] "+format, v...)
}

func (d *defaultLogger) Warning(v ...interface{})  {
	d.log.Println(append([]interface{}{"[WARN]"}, v...)...)
}

func (d *defaultLogger) Warningf(format string, v ...interface{})  {
	d.log.Printf("[WARN] "+format, v...)
}

func (d *defaultLogger) Fatal(v ...interface{}) {
	d.Error(v...)
	os.Exit(1)
}

func (d *defaultLogger) Fatalf(format string, v ...interface{}) {
	d.Errorf(format, v...)
	os.Exit(1)
}

func (d *defaultLogger) Panic(v ...interface{}) {
	d.Error(v...)
	panic(fmt.Sprint(v...))
}

func (d *defaultLogger) Panicf(format string, v ...interface{}) {
	d.Errorf(format, v...)
	panic(fmt.Sprintf(format, v...))
}

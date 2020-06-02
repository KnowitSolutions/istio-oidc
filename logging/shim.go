package logging

import (
	"fmt"
	"github.com/apex/log"
	"github.com/go-logr/logr"
)

const loggerKey = "logger"

var (
	Log      = &logger{Interface: log.Log}
	LogLevel = 0
)

type logger struct {
	log.Interface
	name  string
	level int
}

func (l *logger) Info(msg string, kvs ...interface{}) {
	if l.Enabled() {
		fields := mkFields(kvs)
		l.WithFields(fields).WithField(loggerKey, l.name).Info(msg)
	}
}

func (l *logger) Enabled() bool {
	return l.level <= LogLevel
}

func (l *logger) Error(err error, msg string, kvs ...interface{}) {
	fields := mkFields(kvs)
	l.WithError(err).WithFields(fields).WithField(loggerKey, l.name).Error(msg)
}

func (l *logger) V(level int) logr.InfoLogger {
	dup := *l
	dup.level = level
	return &dup
}

func (l *logger) WithValues(kvs ...interface{}) logr.Logger {
	fields := mkFields(kvs)
	dup := *l
	dup.Interface = l.WithFields(fields)
	return &dup
}

func (l *logger) WithName(name string) logr.Logger {
	dup := *l
	if dup.name != "" {
		dup.name += "."
	}
	dup.name += name
	return &dup
}

func mkFields(kvs []interface{}) *log.Fields {
	if len(kvs)%2 != 0 {
		panic("inconsistent key-value pairs")
	}

	fields := make(log.Fields, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		switch k := kvs[i].(type) {
		case string:
			fields[k] = kvs[i+1]
		case fmt.Stringer:
			fields[k.String()] = kvs[i+1]
		default:
			panic("non-string key in key-value pair")
		}
	}

	return &fields
}

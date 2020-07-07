package log

import (
	"github.com/apex/log"
	"github.com/go-logr/logr"
)

const loggerKey = "logger"

var (
	Shim = &logger{Interface: log.Log}
)

type logger struct {
	log.Interface
	name  string
}

func (l *logger) Info(msg string, kvs ...interface{}) {
	fields := MakeValues(kvs...)
	l.WithFields(fields).WithField(loggerKey, l.name).Info(msg)
}

func (l *logger) Enabled() bool {
	return true
}

func (l *logger) Error(err error, msg string, kvs ...interface{}) {
	fields := MakeValues(kvs...)
	l.WithFields(fields).WithError(err).WithField(loggerKey, l.name).Error(msg)
}

func (l *logger) V(_ int) logr.InfoLogger {
	dup := *l
	return &dup
}

func (l *logger) WithValues(kvs ...interface{}) logr.Logger {
	fields := MakeValues(kvs...)
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

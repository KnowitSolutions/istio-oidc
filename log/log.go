package log

import (
	"context"
	"fmt"
	"github.com/apex/log"
)

func MakeValues(kvs ...interface{}) log.Fields {
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

	return fields
}

func WithValues(ctx context.Context, kvs ...interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	fields := MakeValues(kvs...)
	logger := log.FromContext(ctx).WithFields(fields)
	return log.NewContext(ctx, logger)
}

func Info(ctx context.Context, vals log.Fields, msg string)  {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := log.FromContext(ctx)
	if vals != nil {
		logger = logger.WithFields(vals)
	}

	logger.Info(msg)
}

func Error(ctx context.Context, err error, msg string)  {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := log.FromContext(ctx)
	if err != nil {
		logger = logger.WithError(err)
	}

	logger.Error(msg)
}
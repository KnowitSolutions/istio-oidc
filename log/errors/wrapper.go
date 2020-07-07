package errors

import (
	"github.com/apex/log"
)

type annotated struct {
	msg    string
	fields log.Fields
	cause  error
}

func mk(msg string, meta []interface{}) *annotated {
	if len(meta)%2 != 0 {
		panic("uneven number of metadata parameters")
	}

	fields := make(log.Fields, len(meta)/2)
	for i := 0; i < len(meta); i += 2 {
		k, v := meta[i], meta[i+1]

		if _, ok := k.(string); !ok {
			panic("received non string key metadata parameter")
		}

		fields[k.(string)] = v
	}

	return &annotated{msg, fields, nil}
}

func New(msg string, meta ...interface{}) error {
	return mk(msg, meta)
}

func Wrap(err error, msg string, meta ...interface{}) error {
	if err == nil {
		return nil
	}

	wrapped := mk(msg, meta)
	wrapped.cause = err
	return wrapped
}

func (err *annotated) Error() string {
	if err.cause != nil {
		if err.msg != "" {
			return err.msg + ": " + err.cause.Error()
		} else {
			return err.cause.Error()
		}
	} else {
		return err.msg
	}
}

func (err *annotated) Fields() log.Fields {
	var sub log.Fields
	if cause, ok := err.cause.(log.Fielder); ok {
		sub = cause.Fields()
	}

	fields := make(log.Fields, len(err.fields)+len(sub))
	for k, v := range sub {
		fields[k] = v
	}
	for k, v := range err.fields {
		fields[k] = v
	}

	return fields
}

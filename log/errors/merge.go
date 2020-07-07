package errors

import (
	"github.com/apex/log"
	"strings"
)

type merged []error

func Merge(errs ...error) error {
	m := make(merged, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			m = append(m, err)
		}
	}

	switch len(m) {
	case 0:
		return nil
	case 1:
		return m[0]
	}

	return m
}

func (errs merged) Error() string {
	var b strings.Builder
	b.WriteString("multiple errors: ")

	first := true
	for _, err := range errs {
		if first {
			first = false
		} else {
			b.WriteString(", ")
		}
		b.WriteString(err.Error())
	}

	return b.String()
}

func (errs merged) Fields() log.Fields {
	fields := log.Fields{}
	for _, err := range errs {
		if fielder, ok := err.(log.Fielder); ok {
			for k, v := range fielder.Fields() {
				fields[k] = v
			}
		}
	}
	return fields
}
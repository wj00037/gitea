package sqltrace

import (
	"fmt"
	"reflect"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.opentelemetry.io/otel/trace"
)

type AttrSetter interface {
	set(*traceConfig)
}
type AttrSetterImpl func(*traceConfig)

func (attrSetterImpl AttrSetterImpl) set(config *traceConfig) {
	attrSetterImpl(config)
}

type traceConfig struct {
	db     string
	tracer trace.Tracer
	attrs  []attribute.KeyValue
}

func SetDBName(name string) AttrSetter {
	return AttrSetterImpl(func(c *traceConfig) {
		c.attrs = append(c.attrs, semconv.DBName(name))
	})
}

func replaceSql(sql string, args []interface{}) string {
	for i, arg := range args {
		if arg == nil {
			continue
		}
		if reflect.TypeOf(arg).Kind() == reflect.Ptr {
			sql = strings.Replace(sql, fmt.Sprintf("$%d", i+1), fmt.Sprintf("'%v'", reflect.ValueOf(arg).Elem().Interface()), -1)
		} else {
			sql = strings.Replace(sql, fmt.Sprintf("$%d", i+1), fmt.Sprintf("'%v'", arg), -1)
		}
	}
	return sql
}

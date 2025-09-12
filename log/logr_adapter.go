package log

import (
	"fmt"

	"github.com/go-logr/logr"
)

// logrSink implements logr.LogSink and delegates logging calls to the
// package LoggerFacade. We build a logr.Logger via `logr.New(&logrSink{})`.
type logrSink struct {
	lf     LoggerFacade
	name   string
	values map[string]interface{}
}

// NewLogrAdapter cria um novo logr.Logger que delega para o LoggerFacade fornecido.
func NewLogrAdapter(l LoggerFacade) logr.Logger {
	return logr.New(&logrSink{lf: l})
}

// NewGlobalLogr cria um logr.Logger usando o logger global do pacote `log`.
func NewGlobalLogr() logr.Logger { return NewLogrAdapter(GetGlobal()) }

func (r *logrSink) Init(_ logr.RuntimeInfo) {}

func (r *logrSink) Enabled(_ int) bool { return true }

func (r *logrSink) Info(level int, msg string, keysAndValues ...any) {
	fields := r.cloneValues()
	mergeKV(fields, keysAndValues)
	if r.name != "" {
		fields["logger"] = r.name
	}
	if level > 0 {
		r.lf.WithFields(fields).Debug(msg)
		return
	}
	r.lf.WithFields(fields).Info(msg)
}

func (r *logrSink) Error(err error, msg string, keysAndValues ...any) {
	fields := r.cloneValues()
	mergeKV(fields, keysAndValues)
	if r.name != "" {
		fields["logger"] = r.name
	}
	if len(fields) == 0 {
		r.lf.Error(err, msg)
		return
	}
	r.lf.Errorw(msg, err, fields)
}

func (r *logrSink) WithValues(keysAndValues ...any) logr.LogSink {
	nv := r.cloneValues()
	mergeKV(nv, keysAndValues)
	return &logrSink{lf: r.lf, name: r.name, values: nv}
}

func (r *logrSink) WithName(name string) logr.LogSink {
	var newName string
	if r.name == "" {
		newName = name
	} else {
		newName = r.name + "/" + name
	}
	return &logrSink{lf: r.lf, name: newName, values: r.cloneValues()}
}

func (r *logrSink) cloneValues() map[string]interface{} {
	if r.values == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(r.values))
	for k, v := range r.values {
		out[k] = v
	}
	return out
}

// mergeKV merges a slice of key/value pairs into the destination map. Keys are
// stringified via fmt.Sprint when needed. If an odd number of elements is
// provided, the last key will have a nil value.
func mergeKV(dest map[string]interface{}, kv []any) {
	if dest == nil {
		return
	}
	for i := 0; i < len(kv); i += 2 {
		key := fmt.Sprint(kv[i])
		var val interface{}
		if i+1 < len(kv) {
			val = kv[i+1]
		}
		dest[key] = val
	}
}

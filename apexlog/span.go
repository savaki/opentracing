package apexlog

import (
	"time"

	"github.com/apex/log"
	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
)

type Span struct {
	tracer        *Tracer
	operationName string
	startedAt     time.Time
	logger        *log.Entry

	// mutex protects the following
	baggage map[string]string
	tags    map[string]interface{}
}

// ForeachBaggageItem grants access to all baggage items stored in the
// SpanContext.
// The handler function will be called for each baggage key/value pair.
// The ordering of items is not guaranteed.
//
// The bool return value indicates if the handler wants to continue iterating
// through the rest of the baggage items; for example if the handler is trying to
// find some baggage item by pattern matching the name, it can return false
// as soon as the item is found to stop further iterations.
func (s *Span) ForeachBaggageItem(handler func(k, v string) bool) {
	for k, v := range s.baggage {
		if ok := handler(k, v); ok {
			return
		}
	}
}

// Sets the end timestamp and finalizes Span state.
//
// With the exception of calls to Context() (which are always allowed),
// Finish() must be the last call made to any span instance, and to do
// otherwise leads to undefined behavior.
func (s *Span) Finish() {
	s.logger.Stop(nil)
}

// FinishWithOptions is like Finish() but with explicit control over
// timestamps and log data.
func (s *Span) FinishWithOptions(opts opentracing.FinishOptions) {
	for _, record := range opts.LogRecords {
		s.tracer.info(s.logger, s.baggage, s.tags, record.Fields...)
	}
	s.logger.Stop(nil)
}

// Context() yields the SpanContext for this Span. Note that the return
// value of Context() is still valid after a call to Span.Finish(), as is
// a call to Span.Context() after a call to Span.Finish().
func (s *Span) Context() opentracing.SpanContext {
	return s
}

// Sets or changes the operation name.
func (s *Span) SetOperationName(operationName string) opentracing.Span {
	s.operationName = operationName
	return s
}

// Adds a tag to the span.
//
// If there is a pre-existing tag set for `key`, it is overwritten.
//
// Tag values can be numeric types, strings, or bools. The behavior of
// other tag value types is undefined at the OpenTracing level. If a
// tracing system does not know how to handle a particular value type, it
// may ignore the tag, but shall not panic.
func (s *Span) SetTag(key string, value interface{}) opentracing.Span {
	if s.tags == nil {
		s.tags = map[string]interface{}{}
	}
	s.tags[key] = value
	return s
}

// LogFields is an efficient and type-checked way to record key:value
// logging data about a Span, though the programming interface is a little
// more verbose than LogKV(). Here's an example:
//
//    span.LogFields(
//        log.String("event", "soft error"),
//        log.String("type", "cache timeout"),
//        log.Int("waited.millis", 1500))
//
// Also see Span.FinishWithOptions() and FinishOptions.BulkLogData.
func (s *Span) LogFields(fields ...otlog.Field) {
	s.tracer.info(s.logger, s.baggage, s.tags, fields...)
}

// LogKV is a concise, readable way to record key:value logging data about
// a Span, though unfortunately this also makes it less efficient and less
// type-safe than LogFields(). Here's an example:
//
//    span.LogKV(
//        "event", "soft error",
//        "type", "cache timeout",
//        "waited.millis", 1500)
//
// For LogKV (as opposed to LogFields()), the parameters must appear as
// key-value pairs, like
//
//    span.LogKV(key1, val1, key2, val2, key3, val3, ...)
//
// The keys must all be strings. The values may be strings, numeric types,
// bools, Go error instances, or arbitrary structs.
//
// (Note to implementors: consider the log.InterleavedKVToFields() helper)
func (s *Span) LogKV(alternatingKeyValues ...interface{}) {
	var fields []otlog.Field
	for i := 0; i+1 < len(alternatingKeyValues); i += 2 {
		key, ok := alternatingKeyValues[i].(string)
		if !ok {
			continue
		}

		switch value := alternatingKeyValues[i+1].(type) {
		case string:
			fields = append(fields, otlog.String(key, value))
		case int:
			fields = append(fields, otlog.Int(key, value))
		case int64:
			fields = append(fields, otlog.Int64(key, value))
		case int32:
			fields = append(fields, otlog.Int32(key, value))
		case uint64:
			fields = append(fields, otlog.Uint64(key, value))
		case uint32:
			fields = append(fields, otlog.Uint32(key, value))
		case bool:
			fields = append(fields, otlog.Bool(key, value))
		case float32:
			fields = append(fields, otlog.Float32(key, value))
		case float64:
			fields = append(fields, otlog.Float64(key, value))
		case error:
			fields = append(fields, otlog.Error(value))
		default:
			fields = append(fields, otlog.Object(key, value))
		}
	}
}

// SetBaggageItem sets a key:value pair on this Span and its SpanContext
// that also propagates to descendants of this Span.
//
// SetBaggageItem() enables powerful functionality given a full-stack
// opentracing integration (e.g., arbitrary application data from a mobile
// app can make it, transparently, all the way into the depths of a storage
// system), and with it some powerful costs: use this feature with care.
//
// IMPORTANT NOTE #1: SetBaggageItem() will only propagate baggage items to
// *future* causal descendants of the associated Span.
//
// IMPORTANT NOTE #2: Use this thoughtfully and with care. Every key and
// value is copied into every local *and remote* child of the associated
// Span, and that can add up to a lot of network and cpu overhead.
//
// Returns a reference to this Span for chaining.
func (s *Span) SetBaggageItem(restrictedKey, value string) opentracing.Span {
	if s.baggage == nil {
		s.baggage = map[string]string{}
	}
	s.baggage[restrictedKey] = value
	return s
}

// Gets the value for a baggage item given its key. Returns the empty string
// if the value isn't found in this Span.
func (s *Span) BaggageItem(restrictedKey string) string {
	if s.baggage == nil {
		return ""
	}
	return s.baggage[restrictedKey]
}

// Provides access to the Tracer that created this Span.
func (s *Span) Tracer() opentracing.Tracer {
	return s.tracer
}

// Deprecated: use LogFields or LogKV
func (s *Span) LogEvent(event string) {}

// Deprecated: use LogFields or LogKV
func (s *Span) LogEventWithPayload(event string, payload interface{}) {
}

// Deprecated: use LogFields or LogKV
func (s *Span) Log(data opentracing.LogData) {

}

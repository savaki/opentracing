package apexlog

import (
	"time"

	"github.com/apex/log"
	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
)

type Tracer struct {
	logger log.Interface
	msgKey string
}

func (t *Tracer) makeFields(baggage map[string]string, tags map[string]interface{}, fields ...otlog.Field) (string, log.Fields) {
	var (
		f   = log.Fields{}
		msg string
	)

	for k, v := range baggage {
		f[k] = v
	}
	for k, v := range tags {
		f[k] = v
	}

	for _, field := range fields {
		var (
			key   = field.Key()
			value = field.Value()
		)

		if key == t.msgKey {
			msg, _ = value.(string)
		} else {
			f[key] = value
		}
	}

	return msg, f
}

func (t *Tracer) info(logger log.Interface, baggage map[string]string, tags map[string]interface{}, fields ...otlog.Field) {
	if msg, f := t.makeFields(baggage, tags, fields...); len(f) > 0 {
		logger.WithFields(f).Info(msg)
	} else {
		logger.Info(msg)
	}
}

// Create, start, and return a new Span with the given `operationName` and
// incorporate the given StartSpanOption `opts`. (Note that `opts` borrows
// from the "functional options" pattern, per
// http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis)
//
// A Span with no SpanReference options (e.g., opentracing.ChildOf() or
// opentracing.FollowsFrom()) becomes the root of its own trace.
//
// Examples:
//
//     var tracer opentracing.Tracer = ...
//
//     // The root-span case:
//     sp := tracer.StartSpan("GetFeed")
//
//     // The vanilla child span case:
//     sp := tracer.StartSpan(
//         "GetFeed",
//         opentracing.ChildOf(parentSpan.Context()))
//
//     // All the bells and whistles:
//     sp := tracer.StartSpan(
//         "GetFeed",
//         opentracing.ChildOf(parentSpan.Context()),
//         opentracing.Tag{"user_agent", loggedReq.UserAgent},
//         opentracing.StartTime(loggedReq.Timestamp),
//     )
//
func (t *Tracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	options := &opentracing.StartSpanOptions{}
	for _, opt := range opts {
		opt.Apply(options)
	}

	var (
		parent *Span
		tags   = map[string]interface{}{}
	)

	for _, ref := range options.References {
		if ref.Type == opentracing.ChildOfRef {
			if v, ok := ref.ReferencedContext.(*Span); ok {
				parent = v
			}
		}
	}

	span := &Span{
		tracer:        t,
		operationName: operationName,
		startedAt:     time.Now(),
	}

	if parent != nil {
		for k, v := range parent.baggage {
			span.SetBaggageItem(k, v)
		}
	}

	for k, v := range options.Tags {
		tags[k] = v
	}

	_, f := t.makeFields(span.baggage, tags)
	if parent == nil {
		span.logger = t.logger.WithFields(f).Trace(operationName)
	} else {
		span.logger = parent.logger.WithFields(f).Trace(operationName)
	}

	return span
}

// Inject() takes the `sm` SpanContext instance and injects it for
// propagation within `carrier`. The actual type of `carrier` depends on
// the value of `format`.
//
// OpenTracing defines a common set of `format` values (see BuiltinFormat),
// and each has an expected carrier type.
//
// Other packages may declare their own `format` values, much like the keys
// used by `context.Context` (see
// https://godoc.org/golang.org/x/net/context#WithValue).
//
// Example usage (sans error handling):
//
//     carrier := opentracing.HTTPHeadersCarrier(httpReq.Header)
//     err := tracer.Inject(
//         span.Context(),
//         opentracing.HTTPHeaders,
//         carrier)
//
// NOTE: All opentracing.Tracer implementations MUST support all
// BuiltinFormats.
//
// Implementations may return opentracing.ErrUnsupportedFormat if `format`
// is not supported by (or not known by) the implementation.
//
// Implementations may return opentracing.ErrInvalidCarrier or any other
// implementation-specific error if the format is supported but injection
// fails anyway.
//
// See Tracer.Extract().
func (t *Tracer) Inject(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
	return opentracing.ErrUnsupportedFormat
}

// Extract() returns a SpanContext instance given `format` and `carrier`.
//
// OpenTracing defines a common set of `format` values (see BuiltinFormat),
// and each has an expected carrier type.
//
// Other packages may declare their own `format` values, much like the keys
// used by `context.Context` (see
// https://godoc.org/golang.org/x/net/context#WithValue).
//
// Example usage (with StartSpan):
//
//
//     carrier := opentracing.HTTPHeadersCarrier(httpReq.Header)
//     clientContext, err := tracer.Extract(opentracing.HTTPHeaders, carrier)
//
//     // ... assuming the ultimate goal here is to resume the trace with a
//     // server-side Span:
//     var serverSpan opentracing.Span
//     if err == nil {
//         span = tracer.StartSpan(
//             rpcMethodName, ext.RPCServerOption(clientContext))
//     } else {
//         span = tracer.StartSpan(rpcMethodName)
//     }
//
//
// NOTE: All opentracing.Tracer implementations MUST support all
// BuiltinFormats.
//
// Return values:
//  - A successful Extract returns a SpanContext instance and a nil error
//  - If there was simply no SpanContext to extract in `carrier`, Extract()
//    returns (nil, opentracing.ErrSpanContextNotFound)
//  - If `format` is unsupported or unrecognized, Extract() returns (nil,
//    opentracing.ErrUnsupportedFormat)
//  - If there are more fundamental problems with the `carrier` object,
//    Extract() may return opentracing.ErrInvalidCarrier,
//    opentracing.ErrSpanContextCorrupted, or implementation-specific
//    errors.
//
// See Tracer.Inject().
func (t *Tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	return nil, opentracing.ErrUnsupportedFormat
}

func New(logger log.Interface, options ...Option) *Tracer {
	c := &config{
		msgKey: DefaultMsgKey,
	}
	for _, opt := range options {
		opt(c)
	}

	if logger == nil {
		logger = log.Log
	}

	return &Tracer{
		logger: logger,
		msgKey: c.msgKey,
	}
}

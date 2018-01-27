package apexlog

import (
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/tj/assert"
)

func TestImplementsTracer(t *testing.T) {
	var tracer opentracing.Tracer = New(nil)
	assert.NotNil(t, tracer)
}

func TestParentChild(t *testing.T) {
	tracer := New(nil)
	parent := tracer.StartSpan("parent", opentracing.Tags{
		"tk": "tv",
	})
	parent.SetBaggageItem("bk", "bv")
	child := tracer.StartSpan("child", opentracing.ChildOf(parent.Context()))
	child.Finish()
	parent.Finish()
}

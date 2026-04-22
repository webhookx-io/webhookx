package dao

import "context"

type Options struct {
	Table            string
	EntityName       string
	Workspace        bool
	CachePropagate   bool
	CacheName        string
	PropagateHandler func(ctx context.Context, opts *Options, id string, entity interface{})
	Instrumented     bool
}

type OptionFunc func(*Options)

func WithInstrumented() OptionFunc {
	return func(o *Options) {
		o.Instrumented = true
	}
}

func WithWorkspace(workspace bool) OptionFunc {
	return func(o *Options) {
		o.Workspace = workspace
	}
}

func WithPropagateHandler(fn func(ctx context.Context, opts *Options, id string, entity interface{})) OptionFunc {
	return func(o *Options) {
		o.PropagateHandler = fn
	}
}

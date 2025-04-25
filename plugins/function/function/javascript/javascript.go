package javascript

import (
	"errors"
	"fmt"
	"github.com/dop251/goja"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/webhookx-io/webhookx/plugins/function/sdk"
	"strings"
	"time"
)

type JavaScript struct {
	opts   Options
	vm     *goja.Runtime
	script string
}

type Options struct {
	Timeout time.Duration
}

func New(script string, opts Options) *JavaScript {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	return &JavaScript{
		opts:   opts,
		script: script,
		vm:     vm,
	}
}

var cache, _ = lru.New[string, *goja.Program](128)

func (m *JavaScript) Execute(ctx *sdk.ExecutionContext) (res sdk.ExecutionResult, err error) {
	vm := m.vm

	err = vm.GlobalObject().Set("webhookx", sdk.NewSDK(&sdk.Options{
		Context: ctx,
		Result:  &res,
	}))
	if err != nil {
		return
	}

	err = vm.Set("console", map[string]interface{}{
		"log": func(call goja.FunctionCall) goja.Value {
			sb := strings.Builder{}
			for i, arg := range call.Arguments {
				sb.WriteString(arg.String())
				if i != len(call.Arguments)-1 {
					sb.WriteString(" ")
				}
			}
			fmt.Println(sb.String())
			return goja.Undefined()
		},
	})
	if err != nil {
		return
	}

	if m.opts.Timeout > 0 {
		timer := time.AfterFunc(m.opts.Timeout, func() { vm.Interrupt(errors.New("timeout")) })
		defer timer.Stop()
	}

	program, ok := cache.Get(m.script)
	if !ok {
		program, err = goja.Compile("", m.script, false)
		if err != nil {
			return res, err
		}
		cache.Add(m.script, program)
	}

	_, err = vm.RunProgram(program)
	if err != nil {
		if e, ok := err.(*goja.InterruptedError); ok {
			err = e.Unwrap()
		}
		return
	}

	var handle func() (interface{}, error)
	handleFunction := vm.Get("handle")
	if handleFunction == nil {
		return res, errors.New("handle is not defined")
	}
	err = vm.ExportTo(handleFunction, &handle)
	if err != nil {
		return
	}

	output, err := handle()
	if err != nil {
		switch e := err.(type) {
		case *goja.InterruptedError:
			return res, e.Unwrap()
		case *goja.Exception:
			return res, errors.New(e.String()) // full stacktrace
		}
		return
	}

	res.ReturnValue = output

	return
}

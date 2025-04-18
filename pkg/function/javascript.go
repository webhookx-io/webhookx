package function

import (
	"errors"
	"fmt"
	"github.com/dop251/goja"
	lru "github.com/hashicorp/golang-lru/v2"
	"strings"
	"time"
)

type JavaScriptFunction struct {
	vm     *goja.Runtime
	script string
}

var cache, _ = lru.New[string, *goja.Program](128)

func NewJavaScript(script string) *JavaScriptFunction {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	return &JavaScriptFunction{
		script: script,
		vm:     vm,
	}
}

func (f *JavaScriptFunction) Execute(ctx *ExecutionContext) (res ExecutionResult, err error) {
	vm := f.vm

	api := NewAPI(ctx, &res)
	err = vm.GlobalObject().Set("webhookx", api)
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

	timer := time.AfterFunc(Timeout, func() { vm.Interrupt(errors.New("timeout")) })
	defer timer.Stop()

	program, ok := cache.Get(f.script)
	if !ok {
		program, err = goja.Compile("", f.script, false)
		if err != nil {
			return res, err
		}
		cache.Add(f.script, program)
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
		if e, ok := err.(*goja.InterruptedError); ok {
			err = e.Unwrap()
		}
		return
	}

	res.ReturnValue = output

	return
}

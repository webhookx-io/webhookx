package wasm

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func initLogger() *bytes.Buffer {
	var buf bytes.Buffer
	writer := zapcore.AddSync(&buf)

	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = ""

	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		writer,
		zapcore.DebugLevel,
	))
	zap.ReplaceGlobals(logger)
	return &buf
}

var _ = Describe("wasm", Ordered, func() {

	languages := map[string]string{
		"AssemblyScript": "./testdata/assemblyscript/index.wasm",
		"Rust":           "./testdata/rust/index.wasm",
		"TinyGo":         "./testdata/tinygo/index.wasm",
	}

	for language, filename := range languages {
		It(language, func() {
			buf := initLogger()
			p := new(WasmPlugin)
			p.Config.File = filename

			r, err := http.NewRequest("GET", "https://example.com", nil)
			assert.NoError(GinkgoT(), err)
			c := plugin.NewContext(context.TODO(), r, nil)
			err = p.ExecuteOutbound(c)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "https://httpbin.org/anything", c.Request.URL.String())
			assert.Equal(GinkgoT(), "POST", c.Request.Method)
			assert.EqualValues(GinkgoT(), map[string]string{"Foo": "bar"}, utils.HeaderMap(c.Request.Header))
			assert.Equal(GinkgoT(), "{}", string(c.GetRequestBody()))
			lines := strings.Split(buf.String(), "\n")
			assert.Equal(GinkgoT(), `DEBUG	[wasm]: {"url":"https://example.com","method":"GET","headers":{},"payload":""}`, lines[0])
			assert.Equal(GinkgoT(), `INFO	[wasm]: a info message`, lines[1])
			assert.Equal(GinkgoT(), `WARN	[wasm]: a warn message`, lines[2])
			assert.Equal(GinkgoT(), `ERROR	[wasm]: a error message`, lines[3])
		})
	}

	Context("errors", func() {
		It("file not found", func() {
			p := new(WasmPlugin)
			p.Config.File = "notfound.wasm"
			err := p.ExecuteOutbound(plugin.NewContext(context.TODO(), nil, nil))
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), "open notfound.wasm: no such file or directory", err.Error())
		})

		It("transform not defined", func() {
			p := new(WasmPlugin)
			p.Config.File = "./testdata/no_transform.wasm"
			err := p.ExecuteOutbound(plugin.NewContext(context.TODO(), nil, nil))
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), "exported function 'transform' is not defined in module", err.Error())
		})

		It("transform return does not return 0", func() {
			p := new(WasmPlugin)
			p.Config.File = "./testdata/transform_return_1.wasm"
			err := p.ExecuteOutbound(plugin.NewContext(context.TODO(), nil, nil))
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), "transform failed with value 0", err.Error())
		})
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wasm Suite")
}

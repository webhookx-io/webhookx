package wasm

import (
	"bytes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/pkg/plugin/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
	"testing"
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
			plugin := New()
			plugin.(*WasmPlugin).cfg.File = filename

			pluginReq := &types.Request{
				URL:     "https://example.com",
				Method:  "GET",
				Headers: make(map[string]string),
				Payload: "",
			}
			err := plugin.Execute(pluginReq, nil)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "https://httpbin.org/anything", pluginReq.URL)
			assert.Equal(GinkgoT(), "POST", pluginReq.Method)
			assert.EqualValues(GinkgoT(), map[string]string{"foo": "bar"}, pluginReq.Headers)
			assert.Equal(GinkgoT(), "{}", pluginReq.Payload)
			lines := strings.Split(buf.String(), "\n")
			assert.Equal(GinkgoT(), `DEBUG	[wasm]: {"url":"https://example.com","method":"GET","headers":{},"payload":""}`, lines[0])
			assert.Equal(GinkgoT(), `INFO	[wasm]: a info message`, lines[1])
			assert.Equal(GinkgoT(), `WARN	[wasm]: a warn message`, lines[2])
			assert.Equal(GinkgoT(), `ERROR	[wasm]: a error message`, lines[3])
		})
	}

	Context("errors", func() {
		It("file not found", func() {
			plugin := New()
			plugin.(*WasmPlugin).cfg.File = "notfound.wasm"
			err := plugin.Execute(nil, nil)
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), "open notfound.wasm: no such file or directory", err.Error())
		})

		It("transform not defined", func() {
			plugin := New()
			plugin.(*WasmPlugin).cfg.File = "./testdata/no_transform.wasm"
			err := plugin.Execute(nil, nil)
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), "the 'transform' function is not defined in the module", err.Error())
		})

		It("transform return does not return 0", func() {
			plugin := New()
			plugin.(*WasmPlugin).cfg.File = "./testdata/transform_return_1.wasm"
			err := plugin.Execute(nil, nil)
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), "transform failed with value 0", err.Error())
		})
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wasm Suite")
}

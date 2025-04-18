package function

import (
	"bytes"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Function", Ordered, func() {

	Context("API", func() {
		It("console.log", func() {
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			script := `function handle() { console.log(1, 'a', 'b', 'c') }`
			function := NewJavaScript(script)
			_, err := function.Execute(nil)

			w.Close()
			out, _ := io.ReadAll(r)
			os.Stdout = old

			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), string(out), "1 a b c\n")
		})

		Context("utils", func() {

			Context("hmac", func() {
				It("sanity", func() {
					script := `function handle() {
						return {
							'SHA-1': webhookx.utils.hmac('SHA-1', 'secret', 'string'),
							'SHA-256': webhookx.utils.hmac('SHA-256', 'secret', 'string'),
							'SHA-512': webhookx.utils.hmac('SHA-512', 'secret', 'string'),
							'MD5': webhookx.utils.hmac('MD5', 'secret', 'string'),
						}
					}`
					function := NewJavaScript(script)
					res, err := function.Execute(nil)
					assert.Nil(GinkgoT(), err)
					v := res.ReturnValue.(map[string]interface{})
					assert.Equal(GinkgoT(), "ee8ac90d2d72885a4247f86addea4c3c29a4cba4", hex.EncodeToString(v["SHA-1"].([]byte)))
					assert.Equal(GinkgoT(), "5c5253535592dc799dd2bf256445c45be30da83563a819fd84bdfec58eedfc39", hex.EncodeToString(v["SHA-256"].([]byte)))
					assert.Equal(GinkgoT(), "c13223b6f7331a608ec24bf3adfd8f599622b4d5b28cd27f9dcf92430a6263f435d07ff59809785ef04c5cb4aefd02357578efc8862e254a7505e26c76806194", hex.EncodeToString(v["SHA-512"].([]byte)))
					assert.Equal(GinkgoT(), "f191b78cc46ae59cfa7aeeb6a5f3fed4", hex.EncodeToString(v["MD5"].([]byte)))
				})
				It("should panic", func() {
					script := `function handle() { webhookx.utils.hmac('test', 'secret', 'string') }`
					function := NewJavaScript(script)
					assert.PanicsWithError(GinkgoT(), "unknown algorithm: test", func() { function.Execute(nil) })
				})
			})

			Context("encode", func() {
				It("sanity", func() {
					script := `function handle() {
						return {
							'hex': webhookx.utils.encode('hex', 'string'),
							'base64': webhookx.utils.encode('base64', 'strings=+/'),
							'base64url': webhookx.utils.encode('base64url', 'strings=+/'),
						}
					}`
					function := NewJavaScript(script)
					res, err := function.Execute(nil)
					assert.Nil(GinkgoT(), err)
					v := res.ReturnValue.(map[string]interface{})
					assert.Equal(GinkgoT(), "737472696e67", v["hex"])
					assert.Equal(GinkgoT(), "c3RyaW5ncz0rLw==", v["base64"])
					assert.Equal(GinkgoT(), "c3RyaW5ncz0rLw", v["base64url"])
				})
				It("should panic", func() {
					script := `function handle() { webhookx.utils.encode('test', 'string') }`
					function := NewJavaScript(script)
					assert.PanicsWithError(GinkgoT(), "unknown encode type: test", func() { function.Execute(nil) })
				})
			})

			Context("digestEqual", func() {
				It("should return true", func() {
					script := `function handle() { return webhookx.utils.digestEqual('a', 'a') }`
					function := NewJavaScript(script)
					res, err := function.Execute(nil)
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), true, res.ReturnValue)
				})
				It("should return false", func() {
					script := `function handle() { return webhookx.utils.digestEqual('a', 'b') }`
					function := NewJavaScript(script)
					res, err := function.Execute(nil)
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), false, res.ReturnValue)
				})
			})
		})

		Context("request", func() {
			It("sanity", func() {
				script := `function handle() {
					return {
						method: webhookx.request.getMethod(),
						headers: webhookx.request.getHeaders(),
						body: webhookx.request.getBody()
					}
                }`
				function := NewJavaScript(script)
				result, err := function.Execute(&ExecutionContext{
					HTTPRequest: HTTPRequest{
						Method: "GET",
						Headers: map[string]string{
							"Content-Type":        "application/json",
							"X-Hub-Signature-256": "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17",
						},
						Body: "payload",
					},
				})
				assert.Nil(GinkgoT(), err)
				v := result.ReturnValue.(map[string]interface{})
				assert.Equal(GinkgoT(), "GET", v["method"])
				assert.Equal(GinkgoT(), "payload", v["body"])
				headers := v["headers"].(map[string]string)
				assert.Equal(GinkgoT(), "application/json", headers["Content-Type"])
				assert.Equal(GinkgoT(), "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17", headers["X-Hub-Signature-256"])
			})
		})

		Context("response", func() {
			Context("exit", func() {
				It("sanity", func() {
					script := `function handle() { webhookx.response.exit(400, { 'Content-Type': 'application/json' }, 'test') }`
					function := NewJavaScript(script)
					res, err := function.Execute(nil)
					assert.Nil(GinkgoT(), err)
					assert.NotNil(GinkgoT(), res.HTTPResponse)
					assert.Equal(GinkgoT(), 400, res.HTTPResponse.Code)
					assert.EqualValues(GinkgoT(), map[string]string{"Content-Type": "application/json"}, res.HTTPResponse.Headers)
					assert.Equal(GinkgoT(), "test", res.HTTPResponse.Body)
				})
				It("body can be object", func() {
					script := `function handle() { webhookx.response.exit(400, null, { message: 'invalid signature' } ) }`
					function := NewJavaScript(script)
					res, err := function.Execute(nil)
					assert.Nil(GinkgoT(), err)
					assert.NotNil(GinkgoT(), res.HTTPResponse)
					assert.Equal(GinkgoT(), 400, res.HTTPResponse.Code)
					assert.Equal(GinkgoT(), `{"message":"invalid signature"}`, res.HTTPResponse.Body)
				})
			})
		})

		Context("log", func() {
			var log *zap.SugaredLogger
			var buf bytes.Buffer
			BeforeEach(func() {
				log = zap.S()
				writer := zapcore.AddSync(&buf)
				encoderConfig := zap.NewDevelopmentEncoderConfig()
				encoderConfig.TimeKey = ""
				logger := zap.New(zapcore.NewCore(
					zapcore.NewConsoleEncoder(encoderConfig),
					writer,
					zapcore.DebugLevel,
				))
				zap.ReplaceGlobals(logger)
			})
			AfterEach(func() {
				zap.ReplaceGlobals(log.Desugar())
			})
			It("sanity", func() {
				script := `function handle() { 
                    webhookx.log.debug('debug')
					webhookx.log.info('info')
					webhookx.log.warn('warn')
					webhookx.log.error('error')
                }`
				function := NewJavaScript(script)
				function.Execute(nil)
				assert.Equal(GinkgoT(), "DEBUG\tdebug\nINFO\tinfo\nWARN\twarn\nERROR\terror\n", buf.String())
			})
		})
	})

	Context("timeout", func() {
		It("timeout during load", func() {
			script := `
				// sleep 1500 ms
				var now = new Date().getTime(); 
				while(new Date().getTime() <= now + 1500) {}
			`
			function := NewJavaScript(script)
			_, err := function.Execute(nil)
			assert.Equal(GinkgoT(), "timeout", err.Error())
		})

		It("timeout during executing function", func() {
			script := `function handle() {
				// sleep 1500 ms
				var now = new Date().getTime(); 
				while(new Date().getTime() <= now + 1500) {}  
			}`
			function := NewJavaScript(script)
			_, err := function.Execute(nil)
			assert.Equal(GinkgoT(), "timeout", err.Error())
		})
	})

	Context("errors", func() {
		It("error during loading script", func() {
			script := `throw("js error");`
			function := NewJavaScript(script)
			_, err := function.Execute(nil)
			assert.Equal(GinkgoT(), "js error at <eval>:1:1(1)", err.Error())
		})
		It("error during executing function", func() {
			script := `function handle() { JSON.parse("invalid JSON") }`
			function := NewJavaScript(script)
			_, err := function.Execute(nil)
			assert.Equal(GinkgoT(), "SyntaxError: invalid character 'i' looking for beginning of value at parse (native)", err.Error())
		})
		It("should return error when handle function not defined", func() {
			script := ``
			function := NewJavaScript(script)
			_, err := function.Execute(nil)
			assert.Equal(GinkgoT(), "handle is not defined", err.Error())
		})

	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Function Suite")
}

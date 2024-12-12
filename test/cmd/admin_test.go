package cmd

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/cmd"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/pkg/plugin/webhookx_signature"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
	"os"
	"time"
)

func fullURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)
}

func startHTTP(handler http.HandlerFunc, addr string) *http.Server {
	s := &http.Server{
		Handler: handler,
		Addr:    addr,
	}
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("failed to start HTTP server: %s\n", err.Error())
		}
	}()
	return s
}

var _ = Describe("admin", Ordered, func() {
	Context("sync", func() {
		Context("sanity", func() {
			var app *app.Application
			var db *db.DB

			BeforeAll(func() {
				db = helper.InitDB(true, nil)
				app = utils.Must(helper.Start(map[string]string{
					"WEBHOOKX_ADMIN_LISTEN":   "0.0.0.0:8080",
					"WEBHOOKX_PROXY_LISTEN":   "0.0.0.0:8081",
					"WEBHOOKX_WORKER_ENABLED": "true",
				}))
			})

			AfterAll(func() {
				app.Stop()
			})

			It("sanity", func() {
				output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yml")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "", output)

				endpoint, err := db.Endpoints.Select(context.TODO(), "name", "default-endpoint")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), "default-endpoint", *endpoint.Name)
				assert.Equal(GinkgoT(), true, endpoint.Enabled)
				assert.Equal(GinkgoT(), []string{"charge.succeeded"}, []string(endpoint.Events))
				assert.EqualValues(GinkgoT(), 10000, endpoint.Request.Timeout)
				assert.Equal(GinkgoT(), "https://httpbin.org/anything", endpoint.Request.URL)
				assert.Equal(GinkgoT(), "POST", endpoint.Request.Method)
				assert.Equal(GinkgoT(), "secret", endpoint.Request.Headers["x-apikey"])
				assert.EqualValues(GinkgoT(), "fixed", endpoint.Retry.Strategy)
				assert.EqualValues(GinkgoT(), []int64{0, 3600, 3600}, endpoint.Retry.Config.Attempts)

				source, err := db.Sources.Select(context.TODO(), "name", "default-source")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), "default-source", *source.Name)
				assert.Equal(GinkgoT(), true, source.Enabled)
				assert.Equal(GinkgoT(), "/", source.Path)
				assert.Equal(GinkgoT(), []string{"POST"}, []string(source.Methods))
				assert.Equal(GinkgoT(), 200, source.Response.Code)
				assert.Equal(GinkgoT(), "application/json", source.Response.ContentType)
				assert.Equal(GinkgoT(), `{"message": "OK"}`, source.Response.Body)

				plugins, err := db.Plugins.ListEndpointPlugin(context.TODO(), endpoint.ID)
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 1, len(plugins))
				assert.Equal(GinkgoT(), "webhookx-signature", plugins[0].Name)
				assert.Equal(GinkgoT(), true, plugins[0].Enabled)
				assert.Equal(GinkgoT(), `{"signing_secret": "foo"}`, string(plugins[0].Config))
			})

			It("entities not defined in the declarative configuration should be deleted", func() {
				ws, err := db.Workspaces.GetDefault(context.TODO())
				assert.NoError(GinkgoT(), err)

				endpoint := factory.EndpointWS(ws.ID)
				err = db.Endpoints.Insert(context.TODO(), &endpoint)
				assert.NoError(GinkgoT(), err)

				source := factory.SourceWS(ws.ID)
				err = db.Sources.Insert(context.TODO(), &source)
				assert.NoError(GinkgoT(), err)

				_, err = executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yml")
				assert.Nil(GinkgoT(), err)

				dbEndpoint, err := db.Endpoints.Get(context.TODO(), endpoint.ID)
				assert.NoError(GinkgoT(), err)
				assert.Nil(GinkgoT(), dbEndpoint)

				dbSource, err := db.Sources.Get(context.TODO(), source.ID)
				assert.NoError(GinkgoT(), err)
				assert.Nil(GinkgoT(), dbSource)
			})

			It("entities id should not be changed after multiple syncs", func() {
				_, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yml")
				assert.Nil(GinkgoT(), err)

				endpoint1, err := db.Endpoints.Select(context.TODO(), "name", "default-endpoint")
				assert.NoError(GinkgoT(), err)
				assert.NotNil(GinkgoT(), endpoint1)

				_, err = executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yml")
				assert.Nil(GinkgoT(), err)

				endpoint2, err := db.Endpoints.Select(context.TODO(), "name", "default-endpoint")
				assert.NoError(GinkgoT(), err)
				assert.NotNil(GinkgoT(), endpoint2)

				assert.Equal(GinkgoT(), endpoint1.ID, endpoint2.ID)
			})

			Context("errors", func() {
				It("missing filename", func() {
					output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync")
					assert.NotNil(GinkgoT(), err)
					assert.Equal(GinkgoT(), "Error: accepts 1 arg(s), received 0\n", output)
				})
				It("invalid filename", func() {
					output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "unknown.yaml")
					assert.NotNil(GinkgoT(), err)
					assert.Equal(GinkgoT(), "Error: open unknown.yaml: no such file or directory\n", output)
				})
				It("invalid yaml", func() {
					output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/invalid_webhookx.yml")
					assert.NotNil(GinkgoT(), err)
					assert.Equal(GinkgoT(), "Error: invalid status code: 400 {\"message\":\"malformed yaml content: yaml: unmarshal errors:\\n  line 1: cannot unmarshal !!str `webhook...` into map[string]interface {}\"}\n", output)
				})
			})
		})

		Context("flags", Ordered, func() {
			It("--timeout", func() {
				server := startHTTP(func(writer http.ResponseWriter, r *http.Request) {
					time.Sleep(time.Second * 2)
				}, ":8080")
				output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yml", "--timeout", "1")
				assert.NotNil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "Error: Post \"http://localhost:8080/workspaces/default/sync\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)\n", output)
				assert.Nil(GinkgoT(), server.Shutdown(context.TODO()))
			})

			It("--workspace", func() {
				var url string
				server := startHTTP(func(writer http.ResponseWriter, r *http.Request) {
					url = fullURL(r)
				}, "127.0.0.1:8080")
				output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yml", "--workspace", "foo")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "", output)
				assert.Equal(GinkgoT(), "http://localhost:8080/workspaces/foo/sync", url)
				assert.Nil(GinkgoT(), server.Shutdown(context.TODO()))
			})

			It("--addr", func() {
				time.Sleep(time.Second * 5)
				var url string
				server := startHTTP(func(writer http.ResponseWriter, r *http.Request) {
					url = fullURL(r)
				}, "127.0.0.1:8888")
				output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yml", "--addr", "http://localhost:8888")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "", output)
				assert.Equal(GinkgoT(), "http://localhost:8888/workspaces/default/sync", url)
				assert.Nil(GinkgoT(), server.Shutdown(context.TODO()))
			})
		})
	})

	Context("dump", func() {
		Context("sanity", func() {
			var app *app.Application
			var db *db.DB

			BeforeAll(func() {
				db = helper.InitDB(true, nil)
				app = utils.Must(helper.Start(map[string]string{
					"WEBHOOKX_ADMIN_LISTEN":   "0.0.0.0:8080",
					"WEBHOOKX_PROXY_LISTEN":   "0.0.0.0:8081",
					"WEBHOOKX_WORKER_ENABLED": "true",
				}))
			})

			AfterAll(func() {
				app.Stop()
			})

			It("sanity", func() {
				ws, err := helper.GetDeafultWorkspace()
				assert.Nil(GinkgoT(), err)

				endpoint := factory.EndpointWS(ws.ID,
					factory.WithEndpointID("2q6ItdkHcFz8jQaXxrGp35xsShS"),
				)
				assert.NoError(GinkgoT(), db.Endpoints.Insert(context.TODO(), &endpoint))

				source := factory.SourceWS(ws.ID,
					factory.WithSourceID("2q6ItgNdNEIvoJ2wffn5G5j8HYC"),
				)
				assert.NoError(GinkgoT(), db.Sources.Insert(context.TODO(), &source))

				plugin := factory.PluginWS(
					ws.ID,
					factory.WithPluginID("2q6ItZRVNB0EyVr6j8Pxa7VTohU"),
					factory.WithPluginEndpointID(endpoint.ID),
					factory.WithPluginName("webhookx-signature"),
					factory.WithPluginConfig(&webhookx_signature.Config{SigningSecret: "test"}))
				assert.NoError(GinkgoT(), db.Plugins.Insert(context.TODO(), &plugin))

				output, err := executeCommand(cmd.NewRootCmd(), "admin", "dump")
				assert.Nil(GinkgoT(), err)
				expected, err := os.ReadFile("testdata/dump.yml")
				require.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), string(expected), output)
			})
		})
	})
})

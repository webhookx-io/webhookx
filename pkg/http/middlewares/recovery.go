package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/types"
	"go.uber.org/zap"
)

type Recovery struct {
	CustomizeError func(err error, w http.ResponseWriter) (customized bool)
}

func NewRecovery(customizeError func(err error, w http.ResponseWriter) (customized bool)) *Recovery {
	return &Recovery{CustomizeError: customizeError}
}

func (m *Recovery) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				var err error
				switch v := e.(type) {
				case error:
					err = v
				default:
					err = errors.New(fmt.Sprint(e))
				}

				if m.CustomizeError != nil && m.CustomizeError(err, w) {
					return
				}

				buf := make([]byte, 2048)
				n := runtime.Stack(buf, false)
				buf = buf[:n]

				zap.S().Errorf("panic recovered: %v\n %s", err, buf)
				response.JSON(w, 500, types.ErrorResponse{Message: "internal error"})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/webhookx-io/webhookx/db/errs"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/types"
	"go.uber.org/zap"
)

func PanicRecovery(h http.Handler) http.Handler {
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

				if e, ok := err.(*errs.DBError); ok {
					response.JSON(w, 400, types.ErrorResponse{Message: e.Error()})
					return
				}

				buf := make([]byte, 2048)
				n := runtime.Stack(buf, false)
				buf = buf[:n]

				zap.S().Errorf("panic recovered: %v\n %s", err, buf)
				response.JSON(w, 500, types.ErrorResponse{Message: "internal error"})
			}
		}()

		h.ServeHTTP(w, r)
	})
}

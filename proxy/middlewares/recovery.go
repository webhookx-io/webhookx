package middlewares

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/errs"
	"github.com/webhookx-io/webhookx/pkg/types"
	"go.uber.org/zap"
	"net/http"
	"runtime"
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

				if errors.Is(err, dao.ErrConstraintViolation) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(400)
					bytes, _ := json.Marshal(types.ErrorResponse{Message: err.Error()})
					w.Write(bytes)
					return
				}

				if e, ok := err.(*errs.DBError); ok {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(400)
					bytes, _ := json.Marshal(types.ErrorResponse{Message: e.Error()})
					_, _ = w.Write(bytes)
					return
				}

				buf := make([]byte, 2048)
				n := runtime.Stack(buf, false)
				buf = buf[:n]

				zap.S().Errorf("panic recovered: %v\n %s", err, buf)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(500)
				w.Write([]byte(`{"message": "internal error"}`))
			}
		}()

		h.ServeHTTP(w, r)
	})
}

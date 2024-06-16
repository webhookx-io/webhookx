package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/webhookx-io/webhookx/db/dao"
	"go.uber.org/zap"
	"net/http"
	"runtime"
)

func panicRecovery(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				//var e *Error
				//if errors.As(err.(error), &e) {
				//	w.Header().Set("Content-Type", ApplicationJsonType)
				//	//w.Header()["Content-Type"] = []string{}
				//	w.WriteHeader(http.StatusInternalServerError)
				//	w.Write([]byte("{\"message\": \"internal error\"}"))
				//	return
				//}
				var err error
				switch v := e.(type) {
				case error:
					err = v
				default:
					err = errors.New(fmt.Sprint(e))
				}

				if errors.Is(err, dao.ErrConstraintViolation) {
					w.Header().Set("Content-Type", ApplicationJsonType)
					w.WriteHeader(400)
					bytes, _ := json.Marshal(ErrorResponse{Message: err.Error()})
					w.Write(bytes)
					return
				}

				buf := make([]byte, 2048)
				n := runtime.Stack(buf, false)
				buf = buf[:n]

				zap.S().Errorf("panic recovered: %v\n %s", err, buf)
				w.Header().Set("Content-Type", ApplicationJsonType)
				w.WriteHeader(500)
				w.Write([]byte(`{"message": "internal error"}`))
			}
		}()

		h.ServeHTTP(w, r)
	})
}

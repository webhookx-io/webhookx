package helper

import (
	"errors"
	"fmt"
	"net/http"
)

func StartHttpServer(handler http.HandlerFunc, addr string) *http.Server {
	s := &http.Server{
		Handler: handler,
		Addr:    addr,
	}
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(fmt.Errorf("failed to start HTTP server: %s", err.Error()))
		}
	}()
	return s
}

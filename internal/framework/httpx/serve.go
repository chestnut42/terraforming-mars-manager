package httpx

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
)

func ServeContext(ctx context.Context, handler http.Handler, addr string) error {
	srv := &http.Server{
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return context.WithoutCancel(ctx)
		},
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer cancel()
		defer wg.Done()

		err = srv.Serve(l)
	}()

	<-ctx.Done()
	// nolint:errcheck
	_ = srv.Shutdown(context.Background())
	wg.Wait()
	return err
}

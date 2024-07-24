package signalx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
)

var ErrSignal = errors.New("received signal")

func ListenContext(ctx context.Context, signals ...os.Signal) error {
	ch := make(chan os.Signal, 1)

	signal.Notify(ch, signals...)

	select {
	case <-ctx.Done():
		return nil
	case sig := <-ch:
		return fmt.Errorf("%w (%v)", ErrSignal, sig)
	}
}

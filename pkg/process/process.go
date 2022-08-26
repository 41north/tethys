package process

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func Run(app func(ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		log.Debug("listening for termination signals")
		c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		cancel()
	}()

	return app(ctx)
}

package proxy

import (
	"context"
	"net/http"
	"time"

	"github.com/41north/tethys/pkg/jsonrpc"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var (
	upgrader = websocket.Upgrader{}

	errNoClientsAvailable = jsonrpc.Error{
		Code:    -3200,
		Message: "no client available",
	}

	httpErrGroup = new(errgroup.Group)
	wsErrorGroup = new(errgroup.Group)
)

func listenAndServe(ctx context.Context, options Options) error {
	srv := &http.Server{Addr: options.Address}
	http.HandleFunc("/", requestHandler)

	httpErrGroup.Go(func() error {
		<-ctx.Done()
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(timeoutCtx)
	})

	httpErrGroup.Go(func() error {
		err := srv.ListenAndServe()
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	})

	return httpErrGroup.Wait()
}

func requestHandler(writer http.ResponseWriter, request *http.Request) {
	c, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	handler := newWsHandler(c, wsErrorGroup)
	handler.handle(context.Background())
}

package proxy

import (
	"context"
	"encoding/json"
	"time"

	"github.com/41north/tethys/pkg/jsonrpc"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type wsHandler struct {
	conn   *websocket.Conn
	group  *errgroup.Group
	respCh chan *jsonrpc.Response
}

func newWsHandler(conn *websocket.Conn, group *errgroup.Group) wsHandler {
	return wsHandler{
		conn:   conn,
		group:  group,
		respCh: make(chan *jsonrpc.Response, 256),
	}
}

func (h *wsHandler) handle(ctx context.Context) {
	h.group.Go(h.socketWrite)
	h.group.Go(func() error {
		return h.socketRead(ctx)
	})
}

func (h *wsHandler) socketWrite() error {
	for resp := range h.respCh {
		if err := h.conn.WriteJSON(resp); err != nil {
			switch err.(type) {

			case *websocket.CloseError:
				return err

			default:
				log.WithError(err).Error("failed to write json to websocket")
			}
		}
	}
	return nil
}

func (h *wsHandler) socketRead(ctx context.Context) error {
	for {
		select {

		case <-ctx.Done():
			close(h.respCh)
			return nil

		default:

			_, bytes, err := h.conn.ReadMessage()
			if err != nil {
				close(h.respCh)
				return err
			}

			var req jsonrpc.Request
			if err = json.Unmarshal(bytes, &req); err != nil {
				h.respCh <- &jsonrpc.Response{
					Error: &jsonrpc.ErrParse,
				}
				continue
			}

			h.group.Go(func() error {
				resp, err := invokeWithCache(req, 10*time.Second)
				if err == nil {
					h.respCh <- resp
				}
				return err
			})
		}
	}
}

package web3

import (
	"context"
	"sync"

	"github.com/creachadair/jrpc2"

	log "github.com/sirupsen/logrus"
)

type subManager struct {
	subscriptions sync.Map
}

func (sm *subManager) notify(notification *SubscriptionNotification) {
	id := notification.SubscriptionId
	entry, loaded := sm.subscriptions.Load(id)
	if !loaded {
		log.WithFields(log.Fields{"subscriptionId": id}).
			Warnf("subscription notification received but no handler found")
		return
	}
	ch := entry.(chan *SubscriptionNotification)
	ch <- notification
}

func (sm *subManager) add(id string) chan *SubscriptionNotification {
	ch := make(chan *SubscriptionNotification, 256)
	sm.subscriptions.Store(id, ch)
	return ch
}

func (sm *subManager) remove(id string) {
	entry, loaded := sm.subscriptions.LoadAndDelete(id)
	if !loaded {
		// do nothing
		return
	}
	ch := entry.(chan *SubscriptionNotification)
	close(ch)
}

func (sm *subManager) close() {
	sm.subscriptions.Range(func(id, entry any) bool {
		sm.subscriptions.Delete(id)
		ch := entry.(chan *SubscriptionNotification)
		close(ch)
		return true
	})
}

func (c *Client) Subscribe(ctx context.Context, params []interface{}) (string, error) {
	var result string
	err := c.CallResult(ctx, "eth_subscribe", params, &result)
	return result, err
}

func (c *Client) Unsubscribe(ctx context.Context, subscriptionId string) (bool, error) {
	var result bool
	err := c.CallResult(ctx, "eth_unsubscribe", []any{subscriptionId}, &result)
	return result, err
}

func (c *Client) SubscribeToSyncStatus(context context.Context) (string, error) {
	return c.Subscribe(context, []any{"syncing"})
}

func (c *Client) SubscribeToNewHeads(context context.Context) (string, error) {
	return c.Subscribe(context, []any{"newHeads"})
}

func (c *Client) handleRequest(req *jrpc2.Request) {
	if req.Method() != "eth_subscription" {
		log.Errorf("unexpected request received: %v", req)
		return
	}

	var notification SubscriptionNotification
	err := req.UnmarshalParams(notification)
	if err != nil {
		log.WithError(err).
			WithField("request", req).
			Warn("malformed notification received")
		return
	}

	c.sm.notify(&notification)
}

func (c *Client) HandleSubscription(subscriptionId string) chan *SubscriptionNotification {
	return c.sm.add(subscriptionId)
}

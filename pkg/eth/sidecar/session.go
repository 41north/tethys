package sidecar

import (
	"context"
	"fmt"
	natseth "github.com/41north/web3/pkg/eth/nats"
	"strconv"
	"time"

	"github.com/41north/web3/pkg/eth"
	"github.com/41north/web3/pkg/eth/web3"
	natsutil "github.com/41north/web3/pkg/nats"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type clientSession struct {
	url string
	log *log.Entry

	maxRetryDelay time.Duration

	client *web3.Client
	ctx    context.Context

	stateManager *natseth.StateManager

	clientProfile *eth.ClientProfile
	clientStatus  *eth.ClientStatus

	newHeadsPublisher *natsutil.Publisher[web3.NewHead]

	subscriptionIds []string

	group *errgroup.Group
}

func newClientSession(opts Options) clientSession {
	return clientSession{
		url: opts.ClientUrl,
		log: log.WithFields(log.Fields{
			"component": "ClientSession",
			"url":       opts.ClientUrl,
		}),
		maxRetryDelay: 60 * time.Second,
		group:         new(errgroup.Group),
	}
}

func (cs *clientSession) connect(ctx context.Context) error {

	client, err := web3.NewClient(cs.url)
	if err != nil {
		return errors.Annotate(err, "failed to create eth client")
	}

	cs.ctx = ctx

	closeCh := make(chan interface{}, 1)
	closeHandler := func(code int, message string) {
		cs.log.WithFields(log.Fields{
			"code":    code,
			"message": message,
		}).Debug("client connection has been closed")
		closeCh <- true
	}

	retryDelay := 1 * time.Second

	isConnected := false

	for !isConnected {

		select {
		case <-ctx.Done():
			return nil

		default:
			err = client.Connect(closeHandler)
			if err != nil {
				log.WithError(err).Errorf("failed to connect to web3 client, retrying in %v", retryDelay)
				<-time.After(retryDelay)
				retryDelay = retryDelay * 2
				if retryDelay > cs.maxRetryDelay {
					retryDelay = cs.maxRetryDelay
				}
			}
			isConnected = err == nil
		}

	}

	cs.client = client

	sessionCtx, sessionCancel := context.WithCancel(ctx)
	defer sessionCancel()

	// connect to client and determine it's properties
	clientProfile, err := cs.buildClientProfile()
	if err != nil {
		return errors.Annotate(err, "failed to build client profile")
	}

	// init the state stores based on the client's network and chain id
	stateManager, err := natseth.NewStateManager(natsJs, natseth.NetworkAndChainId(clientProfile.NetworkId, clientProfile.ChainId))
	if err != nil {
		return errors.Annotate(err, "failed to initialise state manager")
	}

	// store the client profile in NATS
	_, err = stateManager.Profiles.Put(clientProfile.Id(), *clientProfile)
	if err != nil {
		return errors.Annotate(err, "failed to put client profile in NATS")
	}

	initialStatus, err := cs.buildInitialClientStatus(clientProfile)
	if err != nil {
		return errors.Annotate(err, "failed to build initial client status")
	}

	// store the client status in NATS
	_, err = stateManager.Status.Put(initialStatus.Id, *initialStatus)
	if err != nil {
		return errors.Annotate(err, "failed to put initial client status into NATS")
	}

	// cache the values
	cs.stateManager = stateManager
	cs.clientProfile = clientProfile
	cs.clientStatus = initialStatus

	// subscribe and publish new heads
	if err = cs.buildNewHeadsPublisher(); err != nil {
		return errors.Annotate(err, "failed to build new heads publisher")
	}

	if err = cs.subscribeToNewHeads(sessionCtx); err != nil {
		return errors.Annotate(err, "failed to subscribe to new heads")
	}

	// start listening for rpc requests from NATS
	cs.group.Go(func() error {
		return cs.listenForRpcRequests(sessionCtx)
	})

	// wait for cancellation or downstream disconnection
	select {
	case <-ctx.Done():
	case <-closeCh:
	}

	cs.log.Debug("stopping")

	sessionCancel()
	if err = cs.group.Wait(); err != nil {
		cs.log.WithError(err).Error("failed to cleanly stop listening for rpc requests")
	}

	client.Close()

	return nil
}

func (cs *clientSession) buildClientProfile() (*eth.ClientProfile, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nodeInfo, err := cs.client.NodeInfo(ctx).Await(ctx)
	if err != nil {
		return nil, errors.New("Could not retrieve client node info")
	}

	clientVersion, err := nodeInfo.ParseClientVersion()
	if err != nil {
		return nil, errors.New("Could not determine client client version")
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	networkIdStr, err := cs.client.NetVersion(ctx).Await(ctx)
	if err != nil {
		return nil, errors.New("failed to retrieve network version")
	}

	networkId, err := strconv.ParseUint(*networkIdStr, 10, 64)
	if err != nil {
		return nil, errors.New("failed to parse network version")
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	chainIdHex, err := cs.client.ChainId(ctx).Await(ctx)
	if err != nil {
		return nil, errors.New("failed to retrieve chain id")
	}

	chainId, err := hexutil.DecodeUint64(*chainIdHex)
	if err != nil {
		return nil, errors.New("failed to parse chain id")
	}

	profile := eth.ClientProfile{
		NetworkId:     networkId,
		ChainId:       chainId,
		NodeInfo:      *nodeInfo,
		ClientVersion: clientVersion,
	}

	return &profile, nil
}

func (cs *clientSession) buildInitialClientStatus(profile *eth.ClientProfile) (*eth.ClientStatus, error) {

	// get sync progress
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	syncing, err := cs.client.SyncProgress(ctx).Await(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get sync progress")
	}

	syncStatus := &web3.SyncStatus{
		Syncing: *syncing,
	}

	// get latest block
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	latestBlock, err := cs.client.LatestBlock(ctx).Await(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get latest block")
	}

	head := &web3.Head{
		BlockNumber:     latestBlock.Number,
		BlockHash:       latestBlock.Hash,
		ParentHash:      latestBlock.ParentHash,
		Difficulty:      latestBlock.Difficulty,
		TotalDifficulty: latestBlock.TotalDifficulty,
	}

	return &eth.ClientStatus{
		Id:         profile.Id(),
		Head:       head,
		SyncStatus: syncStatus,
	}, nil
}

func (cs *clientSession) listenForRpcRequests(ctx context.Context) error {

	clientId := cs.clientProfile.Id()
	networkId := strconv.FormatUint(cs.clientProfile.NetworkId, 10)
	chainId := strconv.FormatUint(cs.clientProfile.ChainId, 10)

	srv, err := natsutil.NewRpcServer(cs.clientProfile.Id(), natsConn, cs.client)
	if err != nil {
		return errors.Annotate(err, "failed to create nats rpc server")
	}

	subFn := func(conn *nats.Conn, msgs chan *nats.Msg) ([]*nats.Subscription, error) {

		// uniquely identifies this client
		subject := eth.SubjectName("eth", "rpc", networkId, chainId, clientId)

		sub, err := conn.ChanSubscribe(subject, msgs)
		if err != nil {
			return nil, errors.Annotatef(err, "failed to subscribe to subject: %s", subject)
		}

		return []*nats.Subscription{sub}, nil
	}

	return srv.ListenAndServe(ctx, subFn)
}

func (cs *clientSession) subscribeToNewHeads(ctx context.Context) error {

	// subscribe for new heads
	requestCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	subId, err := cs.client.SubscribeToNewHeads(requestCtx).Await(requestCtx)
	if err != nil {
		return errors.Annotate(err, "failed to subscribe to new heads")
	}

	cs.subscriptionIds = append(cs.subscriptionIds, *subId)

	//
	newHeads := cs.client.HandleSubscription(*subId)

	cs.group.Go(func() error {

		running := true
		for running {
			select {
			case <-ctx.Done():
				running = false
			case notification, ok := <-newHeads:
				running = ok
				if notification != nil {
					cs.onNewHead(notification)
				}
			}
		}

		cs.log.Debug("newHeads subscription closed")

		// delete status entry in kv store
		if err := cs.stateManager.Status.Delete(cs.clientProfile.Id()); err != nil {
			log.WithError(err).Warn("failed to remove client status from kv store")
		}

		return nil
	})

	return nil
}

func (cs *clientSession) onNewHead(notification *web3.SubscriptionNotification) {

	var newHead web3.NewHead
	err := notification.UnmarshalResult(&newHead)

	if err != nil {
		cs.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("failed to unmarshal new head result")
		return
	}

	cv := cs.clientProfile.ClientVersion
	msgId := fmt.Sprintf("%s:%s:%s", cv.Name, cv.Version, newHead.Hash[2:10])

	_, err = cs.newHeadsPublisher.PublishRaw(notification.Result, nats.MsgId(msgId))
	if err != nil {
		cs.log.WithError(err).Error("failed to publish new head")
	}

	// update client status
	totalDifficulty := newHead.TotalDifficulty

	if totalDifficulty == "" {
		// we fetch the block again as total difficulty is not always available in the newHeads object depending on client impl.
		// total difficulty is useful for working out re-orgs

		requestCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		block, err := cs.client.LatestBlock(context.Background()).Await(requestCtx)
		if err != nil {
			cs.log.WithError(err).Error("failed to fetch block")
			return
		}
		totalDifficulty = block.TotalDifficulty
	}

	// update client status
	statusUpdate := eth.ClientStatus{
		Head: &web3.Head{
			BlockNumber:     newHead.Number,
			BlockHash:       newHead.Hash,
			ParentHash:      newHead.ParentHash,
			Difficulty:      newHead.Difficulty,
			TotalDifficulty: totalDifficulty,
		},
	}

	mergedStatus, err := cs.clientStatus.Merge(&statusUpdate)
	if err != nil {
		log.WithError(err).Error("failed to merge client status")
		return
	}

	if _, err = cs.stateManager.Status.Put(mergedStatus.Id, *mergedStatus); err != nil {
		log.WithError(err).Error("failed to put client status")
	}
}

func (cs *clientSession) buildNewHeadsPublisher() error {

	cp := cs.clientProfile
	cv := cp.ClientVersion
	version := eth.SanitizeVersion(cv.Version)

	networkId := strconv.FormatUint(cp.NetworkId, 10)
	chainId := strconv.FormatUint(cp.ChainId, 10)

	subject := eth.SubjectName("eth", "newHeads", networkId, chainId, cv.Name, version, cp.Id())

	publisher, err := natsutil.NewPublisher[web3.NewHead](
		natsJs, subject,
		func(js nats.JetStreamContext) error {

			streamConfig := &nats.StreamConfig{
				Name:              fmt.Sprintf("eth_%s_%s_newHeads", networkId, chainId),
				Description:       fmt.Sprintf("ETH newHeads for networkId %s and chainId %s", networkId, chainId),
				Subjects:          []string{eth.SubjectName("eth", "newHeads", networkId, chainId, "*", "*", "*")},
				MaxMsgsPerSubject: 128,
			}

			_, err := js.AddStream(streamConfig)
			if err != nil {
				return errors.Annotate(err, "failed to create new heads stream")
			}

			return nil
		},
	)

	cs.newHeadsPublisher = publisher
	return err
}

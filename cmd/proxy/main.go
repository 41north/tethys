package main

import (
	"net/url"

	"github.com/alecthomas/kong"
	log "github.com/sirupsen/logrus"
)

type proxyCmd struct {
	Address   string   `name:"" env:"PROXY_SERVER_ADDRESS" default:":8080" help:"Address to bind the websocket server to."`
	NetworkId uint64   `name:"" env:"ETH_NETWORK_ID" default:"1" help:"Ethereum network id"`
	ChainId   uint64   `name:"" env:"ETH_CHAIN_ID" default:"1" help:"Ethereum chain id"`
	NatsURL   *url.URL `name:"" env:"NATS_URL" default:"ns://127.0.0.1:4222" help:"NATS server url"`
}

var cli struct {
	Debug bool        `short:"d" default:"0" help:"Enable debug logging."`
	Eth   ethProxyCmd `cmd:"" help:"Run an Ethereum proxy"`
}

func main() {
	ctx := kong.Parse(&cli)

	// set debug for now
	if cli.Debug {
		log.SetLevel(log.DebugLevel)
	}

	// configure logging
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

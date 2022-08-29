package main

import (
	"net/url"

	"github.com/alecthomas/kong"
	log "github.com/sirupsen/logrus"
)

type proxyCmd struct {
	Address   string `name:"" env:"PROXY_SERVER_ADDRESS" default:":8080" help:"Address to bind the websocket server to."`
	NetworkId uint64 `name:"" env:"ETH_NETWORK_ID" default:"1" help:"Ethereum network id."`
	ChainId   uint64 `name:"" env:"ETH_CHAIN_ID" default:"1" help:"Ethereum chain id."`
	Nats      struct {
		URL      *url.URL `name:"" env:"URL" default:"ns://127.0.0.1:4222" help:"NATS server url."`
		Embedded struct {
			Enable           bool   `name:"" env:"ENABLE" default:"0" required:"" help:"Starts the proxy with an embedded NATS server."`
			UseDefaultConfig bool   `name:"" env:"USE_DEFAULT_CONFIG" required:"" xor:"nats-config" help:"NATS configuration PATH with default options to be used when NATS embedded mode is enabled."`
			ConfigPath       string `name:"" env:"CONFIG_PATH" required:"" xor:"nats-config" type:"existingfile" help:"NATS configuration PATH options to be used when NATS embedded mode is enabled."`
		} `embed:"" prefix:"embedded." envprefix:"EMBEDDED_"`
	} `embed:"" prefix:"nats." envprefix:"NATS_"`
}

var cli struct {
	Log struct {
		Level string `enum:"debug,info,warn,error" env:"LOG_LEVEL" default:"info" help:"Configure logging level."`
	} `embed:"" prefix:"log."`
	Eth ethProxyCmd `cmd help:"Run an Ethereum proxy."`
}

func main() {
	ctx := kong.Parse(&cli)

	// configure logging
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	// set log level
	switch {
	case cli.Log.Level == "debug":
		log.SetLevel(log.DebugLevel)
	case cli.Log.Level == "info":
		log.SetLevel(log.InfoLevel)
	case cli.Log.Level == "warn":
		log.SetLevel(log.WarnLevel)
	case cli.Log.Level == "error":
		log.SetLevel(log.ErrorLevel)
	}

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

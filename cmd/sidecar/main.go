package main

import (
	"github.com/alecthomas/kong"
	log "github.com/sirupsen/logrus"
)

type sidecarCmd struct {
	ClientUrl            string `name:"client-url" env:"WEB3_URL" default:"ws://127.0.0.1:8546" help:"Websocket url for connecting to a eth client"`
	ClientConnectionType string `name:"client-connection-type" env:"WEB3_CONNECTION_TYPE" default:"ConnectionTypeDirect" help:"Indicates how the sidecar is connecting to the web3 client"`
	// todo make client id required only if connection type is managed
	ClientId string `name:"client-id" env:"WEB3_CLIENT_ID" help:"Allows for manually specifying the client id when the connection type is managed."`
	NatsUrl  string `name:"nats-url" env:"NATS_URL" default:"ns://127.0.0.1:4222" help:"NATS server url"`
}

var cli struct {
	Log struct {
		Level string `enum:"debug,info,warn,error" env:"LOG_LEVEL" default:"info" help:"Configure logging level."`
	} `embed:"" prefix:"log-"`
	Eth ethSidecarCmd `cmd:"" help:"Run an Ethereum sidecar"`
}

func main() {
	ctx := kong.Parse(&cli)

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

	// configure logging
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

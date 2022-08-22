package main

import (
	"github.com/alecthomas/kong"
	log "github.com/sirupsen/logrus"
)

type sidecarCmd struct {
	ClientUrl string `name:"client-url" env:"WEB3_URL" default:"ws://127.0.0.1:8546" help:"Websocket url for connecting to a eth client"`
	NatsUrl   string `name:"nats-url" env:"NATS_URL" default:"ns://127.0.0.1:4222" help:"NATS server url"`
}

var cli struct {
	Debug bool          `short:"d" default:"0" help:"Enable debug logging."`
	Eth   ethSidecarCmd `cmd:"" help:"Run an Ethereum sidecar"`
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

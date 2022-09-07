terraform {
  required_providers {
    jetstream = {
      source = "nats-io/jetstream"
    }
  }
}

provider "jetstream" {
  servers = "localhost:4222"
}
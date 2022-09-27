# Tethys

![Build](https://github.com/41north/tethys/actions/workflows/ci.yml/badge.svg)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

Status: _VERY EXPERIMENTAL_, we are gonna be re-writing this thing several times over in the next few months.

Tethys is intended to take the pain out of accessing chain data, whether it be from a local pool of homogenous client
implementations, via a managed service such as [Alchemy](https://www.alchemy.com/) or [Infura](https://infura.io/), or
a mixture of both.

Cost effective, performant and resilient access to chain data is not easy, and it's a problem we keep coming up against.
Tethys is our attempt to solve that problem once and for all, allowing us to focus on what we want to build rather than
getting dragged back down into the mud.

## Roadmap

Our initial focus will be on Ethereum. Eventually we want to expand to other chains.

The following is an initial high level list of things, in no particular order, that we want to achieve:

- [ ] Execution Layer (EL) tracking
  - [x] JSON-RPC websocket
  - [ ] IPC
  - [ ] Protobuf (Erigon)
- [ ] Consensus Layer (CL) tracking
  - [ ] JSON-RPC websocket
  - [ ] IPC
  - [ ] Protobuf (Erigon)
- [ ] Request Proxying across EL and CL clients
  - [ ] Direct addressing
  - [x] Smart routing based on client state and implementation, chain/network and so on
  - [ ] Preferential routing based on client location and type e.g. managed services versus local client
- [ ] EL load balancing for CL clients
- [ ] Mempool tracking
- [ ] Resilient logical subscriptions backed by one or more clients
- [ ] Multi data center and multi regional deployment
- [ ] Kubernetes operator
- [ ] Aggressive caching where-ever it makes sense.

## Documentation

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/41north/tethys)

Full `go doc` style documentation for the project can be viewed online without
installing this package by using the excellent GoDoc site here:
http://godoc.org/github.com/41north/tethys

You can also view the documentation locally once the package is installed with
the `godoc` tool by running `godoc -http=":6060"` and pointing your browser to
http://localhost:6060/pkg/github.com/41north/tethys

## Installation

TBD

## Quick Start

TBD

## License

Tethys is licensed under the [AGPL v3 License](LICENSE)

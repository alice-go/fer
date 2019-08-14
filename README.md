# fer

[![GitHub release](https://img.shields.io/github/release/alice-go/fer.svg)](https://github.com/alice-go/fer/releases)
[![GoDoc](https://godoc.org/github.com/alice-go/fer?status.svg)](https://godoc.org/github.com/alice-go/fer)
[![Build Status](https://travis-ci.org/alice-go/fer.svg?branch=master)](https://travis-ci.org/alice-go/fer)
[![codecov](https://codecov.io/gh/alice-go/fer/branch/master/graph/badge.svg)](https://codecov.io/gh/alice-go/fer)
[![DOI](https://zenodo.org/badge/73269900.svg)](https://zenodo.org/badge/latestdoi/73269900)

`fer` is a simple reimplementation of [FairMQ](https://github.com/FairRootGroup/FairMQ) in [Go](https://golang.org).

## License

`fer` is released under the `BSD-3` license.

## Installation

`fer` is installable _via_ `go` `get`:

```sh
$> go get github.com/alice-go/fer/...
```

*NOTE:* you need at least `go1.7`.

## Documentation

Documentation is available on [godoc](https://godoc.org/github.com/alice-go/fer).

## Examples

### Testing example-2 from FairMQ tutorial

```sh
## terminal 1
$> fer-ex-sink --id sink1 --mq-config ./_example/cmd/testdata/ex2-sampler-processor-sink.json

## terminal 2
$> fer-ex-processor --id processor --mq-config ./_example/cmd/testdata/ex2-sampler-processor-sink.json

## terminal 3
$> fer-ex-sampler --id sampler1 --mq-config ./_example/cmd/testdata/ex2-sampler-processor-sink.json
```

This will run 3 devices, using the `ZeroMQ` transport.

To run with `nanomsg` as a transport layer, add `--transport nanomsg` to the invocations.

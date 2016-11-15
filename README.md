# fer

[![GoDoc](https://godoc.org/github.com/sbinet-alice/fer?status.svg)](https://godoc.org/github.com/sbinet-alice/fer)

`fer` is a simple reimplementation of [FairMQ](https://github.com/FairRootGroup/FairRoot/tree/master/fairmq) in [Go](https://golang.org).

## License

`fer` is released under the `BSD-3` license.

## Installation

`fer` is installable _via_ `go` `get`:

```sh
$> go get github.com/sbinet-alice/fer/...
```

*NOTE:* you need at least `go1.7`.

## Documentation

Documentation is available on [godoc](https://godoc.org/github.com/sbinet-alice/fer).

## Examples

### Testing example-2 from FairMQ tutorial

```sh
## terminal 1
$> fer-ex-sink --id sink1 --mq-config ./example/cmd/testdata/ex2-sampler-processor-sink.json

## terminal 2
$> fer-ex-processor --id processor --mq-config ./example/cmd/testdata/ex2-sampler-processor-sink.json

## terminal 3
$> fer-ex-sampler --id sampler1 --mq-config ./example/cmd/testdata/ex2-sampler-processor-sink.json
```

This will run 3 devices, using the `ZeroMQ` transport.

To run with `nanomsg` as a transport layer, add `--transport nanomsg` to the invocations.

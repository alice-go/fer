// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package nanomsg implements the mq.Driver interface and allows
// to use mq.Sockets via nanomsg sockets.
package nanomsg // import "github.com/alice-go/fer/mq/nanomsg"

import (
	"fmt"

	"github.com/alice-go/fer/mq"
	"nanomsg.org/go-mangos"
	"nanomsg.org/go-mangos/protocol/bus"
	"nanomsg.org/go-mangos/protocol/pair"
	"nanomsg.org/go-mangos/protocol/pub"
	"nanomsg.org/go-mangos/protocol/pull"
	"nanomsg.org/go-mangos/protocol/push"
	"nanomsg.org/go-mangos/protocol/rep"
	"nanomsg.org/go-mangos/protocol/req"
	"nanomsg.org/go-mangos/protocol/sub"
	"nanomsg.org/go-mangos/transport/inproc"
	"nanomsg.org/go-mangos/transport/ipc"
	"nanomsg.org/go-mangos/transport/tcp"
)

type socket struct {
	mangos.Socket
	typ mq.SocketType
}

func (s socket) Type() mq.SocketType {
	return s.typ
}

type driver struct{}

func (driver) Name() string {
	return "nanomsg"
}

func (driver) NewSocket(typ mq.SocketType) (mq.Socket, error) {
	var sck mangos.Socket
	var err error

	switch typ {
	case mq.Sub, mq.XSub:
		sck, err = sub.NewSocket()
		if err == nil {
			err = sck.SetOption(mangos.OptionSubscribe, []byte(""))
		}
	case mq.Pub, mq.XPub:
		sck, err = pub.NewSocket()
	case mq.Push:
		sck, err = push.NewSocket()
	case mq.Pull:
		sck, err = pull.NewSocket()
	case mq.Req, mq.Dealer:
		sck, err = req.NewSocket()
	case mq.Rep, mq.Router:
		sck, err = rep.NewSocket()
	case mq.Pair:
		sck, err = pair.NewSocket()
	case mq.Bus:
		sck, err = bus.NewSocket()
	default:
		return nil, fmt.Errorf("fer/nanomsg: invalid socket type %v (%d)", typ, int(typ))
	}

	if err != nil {
		return nil, err
	}

	sck.AddTransport(ipc.NewTransport())
	sck.AddTransport(tcp.NewTransport())
	sck.AddTransport(inproc.NewTransport())
	return socket{Socket: sck, typ: typ}, err
}

func init() {
	mq.Register("nanomsg", driver{})
}

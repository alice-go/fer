// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package nanomsg implements the mq.Driver interface and allows
// to use mq.Sockets via nanomsg sockets.
package nanomsg

import (
	"fmt"

	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/bus"
	"github.com/go-mangos/mangos/protocol/pair"
	"github.com/go-mangos/mangos/protocol/pub"
	"github.com/go-mangos/mangos/protocol/pull"
	"github.com/go-mangos/mangos/protocol/push"
	"github.com/go-mangos/mangos/protocol/rep"
	"github.com/go-mangos/mangos/protocol/req"
	"github.com/go-mangos/mangos/protocol/sub"
	"github.com/go-mangos/mangos/transport/ipc"
	"github.com/go-mangos/mangos/transport/tcp"
	"github.com/sbinet-alice/fer/mq"
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
	return socket{Socket: sck, typ: typ}, err
}

func init() {
	mq.Register("nanomsg", driver{})
}

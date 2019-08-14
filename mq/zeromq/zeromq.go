// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package zeromq implements the mq.Driver interface and allows
// to use mq.Sockets via ZeroMQ sockets.
package zeromq // import "github.com/alice-go/fer/mq/zeromq"

import (
	"context"
	"fmt"
	"strings"

	"github.com/alice-go/fer/mq"
	"github.com/go-zeromq/zmq4"
)

type socket struct {
	zmq zmq4.Socket
	typ mq.SocketType
}

func (s *socket) Close() error {
	if s.zmq != nil {
		s.zmq.Close()
		s.zmq = nil
	}
	return nil
}

func (s *socket) Send(data []byte) error {
	return s.zmq.Send(zmq4.NewMsg(data))
}

func (s *socket) Recv() ([]byte, error) {
	msg, err := s.zmq.Recv()
	return msg.Bytes(), err
}

func (s *socket) Listen(addr string) error {
	addr = globAddr(addr)
	return s.zmq.Listen(addr)
}

func (s *socket) Dial(addr string) error {
	addr = globAddr(addr)
	return s.zmq.Dial(addr)
}

func (s *socket) Type() mq.SocketType {
	return s.typ
}

func globAddr(addr string) string {
	addr = strings.Replace(addr, "//*:", "//0.0.0.0:", 1)
	addr = strings.Replace(addr, ":*", ":0", 1)
	return addr
}

type driver struct{}

func (driver) Name() string {
	return "zeromq"
}

func (drv driver) NewSocket(typ mq.SocketType) (mq.Socket, error) {
	var (
		sck = socket{typ: typ}
		err error
		ctx = context.Background()
	)

	switch typ {
	case mq.Sub:
		sck.zmq = zmq4.NewSub(ctx)

	case mq.XSub:
		return nil, fmt.Errorf("mq/zeromq: mq.XSub not implemented")

	case mq.Pub:
		sck.zmq = zmq4.NewPub(ctx)

	case mq.XPub:
		return nil, fmt.Errorf("mq/zeromq: mq.XPub not implemented")

	case mq.Push:
		sck.zmq = zmq4.NewPush(ctx)

	case mq.Pull:
		sck.zmq = zmq4.NewPull(ctx)

	case mq.Req:
		sck.zmq = zmq4.NewReq(ctx)

	case mq.Dealer:
		return nil, fmt.Errorf("mq/zeromq: mq.Dealer not implemented")

	case mq.Rep:
		sck.zmq = zmq4.NewRep(ctx)

	case mq.Router:
		return nil, fmt.Errorf("mq/zeromq: mq.Router not implemented")

	case mq.Pair:
		return nil, fmt.Errorf("mq/zeromq: mq.Pair not implemented")

	case mq.Bus:
		return nil, fmt.Errorf("mq/zeromq: mq.Bus not implemented")

	default:
		return nil, fmt.Errorf("mq/zeromq: invalid socket type %v (%d)", typ, int(typ))
	}

	switch typ {
	case mq.Sub, mq.XSub:
		err = sck.zmq.SetOption(zmq4.OptionSubscribe, "")
		if err != nil {
			return nil, err
		}
	}

	return &sck, err
}

func init() {
	var drv driver
	mq.Register("zeromq", drv)
}

// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zeromq

// #cgo pkg-config: libzmq
// #include "zmq.h"
// #include <stdlib.h>
// #include <string.h>
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/sbinet-alice/fer/mq"
)

func getError(v C.int) error {
	if v == 0 {
		return nil
	}
	id := C.zmq_errno()
	msg := C.zmq_strerror(id)
	return fmt.Errorf(C.GoString(msg))
}

type socket struct {
	c unsafe.Pointer
}

func (s *socket) Close() error {
	return getError(C.zmq_close(s.c))
}

func (s *socket) Send(data []byte) error {
	cbuf := unsafe.Pointer(&data[0])
	clen := C.size_t(len(data))
	o := C.zmq_send(s.c, cbuf, clen, 0)
	if o > 0 {
		return nil
	}
	return getError(o)
}

func (s *socket) Recv() ([]byte, error) {
	var msg C.zmq_msg_t
	if i := C.zmq_msg_init(&msg); i != 0 {
		return nil, getError(i)
	}
	defer C.zmq_msg_close(&msg)

	size := C.zmq_msg_recv(&msg, s.c, 0)
	if size < 0 {
		return nil, getError(size)
	}
	if size == 0 {
		return []byte{}, nil
	}
	data := make([]byte, int(size))
	C.memcpy(unsafe.Pointer(&data[0]), C.zmq_msg_data(&msg), C.size_t(size))
	err := getError(C.zmq_msg_close(&msg))
	return data, err
}

func (s *socket) Listen(addr string) error {
	caddr := C.CString(addr)
	v := C.zmq_bind(s.c, caddr)
	C.free(unsafe.Pointer(caddr))
	return getError(v)
}

func (s *socket) Dial(addr string) error {
	caddr := C.CString(addr)
	v := C.zmq_connect(s.c, caddr)
	C.free(unsafe.Pointer(caddr))
	return getError(v)
}

type driver struct {
	ctx unsafe.Pointer
}

func (*driver) Name() string {
	return "zeromq"
}

func (drv *driver) NewSocket(typ mq.SocketType) (mq.Socket, error) {
	var (
		sck   socket
		err   error
		ctype C.int
	)

	switch typ {
	case mq.Sub:
		ctype = C.ZMQ_SUB

	case mq.XSub:
		ctype = C.ZMQ_XSUB

	case mq.Pub:
		ctype = C.ZMQ_PUB

	case mq.XPub:
		ctype = C.ZMQ_XPUB

	case mq.Push:
		ctype = C.ZMQ_PUSH

	case mq.Pull:
		ctype = C.ZMQ_PULL

	case mq.Req:
		ctype = C.ZMQ_REQ

	case mq.Dealer:
		ctype = C.ZMQ_DEALER

	case mq.Rep:
		ctype = C.ZMQ_REP

	case mq.Router:
		ctype = C.ZMQ_ROUTER

	case mq.Pair:
		ctype = C.ZMQ_PAIR

	case mq.Bus:
		return nil, fmt.Errorf("mq/zeromq: fer.Bus not implemented")

	default:
		return nil, fmt.Errorf("mq/zeromq: invalid socket type %v (%d)", typ, int(typ))
	}

	sck.c = C.zmq_socket(drv.ctx, ctype)

	if sck.c == nil {
		return nil, getError(1)
	}

	return &sck, err
}

func init() {
	var drv driver
	drv.ctx = C.zmq_ctx_new()
	mq.Register("zeromq", &drv)
}

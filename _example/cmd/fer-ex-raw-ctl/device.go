// Copyright 2017 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/alice-go/fer"
	"github.com/alice-go/fer/config"
)

type sampler struct {
	cfg   config.Device
	datac chan fer.Msg
	n     int
	quit  chan int
}

func (dev *sampler) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *sampler) Init(ctl fer.Controler) error {
	datac, err := ctl.Chan("data1", 0)
	if err != nil {
		return err
	}

	dev.datac = datac
	return nil
}

func (dev *sampler) Run(ctl fer.Controler) error {
	for {
		select {
		case dev.datac <- fer.Msg{Data: newToken("HELLO").Bytes()}:
			dev.n++
			time.Sleep(10 * time.Microsecond)
		case <-ctl.Done():
			return nil
		case <-dev.quit:
			return nil
		}
	}
}

type processor struct {
	cfg    config.Device
	idatac chan fer.Msg
	odatac chan fer.Msg
	n      int
	quit   chan int
}

func (dev *processor) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *processor) Init(ctl fer.Controler) error {
	idatac, err := ctl.Chan("data1", 0)
	if err != nil {
		return err
	}

	odatac, err := ctl.Chan("data2", 0)
	if err != nil {
		return err
	}

	dev.idatac = idatac
	dev.odatac = odatac
	return nil
}

func (dev *processor) Run(ctl fer.Controler) error {
	for {
		select {
		case data := <-dev.idatac:
			tok := tokenFrom(data.Data)
			// ctl.Printf("received: %q\n", string(data.Data))
			out := append([]byte(nil), tok.msg...)
			out = append(out, []byte(" (modified by "+dev.cfg.Name()+")")...)
			tok.msg = out
			dev.odatac <- fer.Msg{Data: tok.Bytes()}
			dev.n++
		case <-ctl.Done():
			return nil
		case <-dev.quit:
			return nil
		}
	}
}

type sink struct {
	cfg   config.Device
	datac chan fer.Msg
	n     int
	out   chan token
	quit  chan int
}

func (dev *sink) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *sink) Init(ctl fer.Controler) error {
	datac, err := ctl.Chan("data2", 0)
	if err != nil {
		return err
	}

	dev.datac = datac
	return nil
}

func (dev *sink) Run(ctl fer.Controler) error {
	for {
		select {
		case data := <-dev.datac:
			dev.n++
			tok := tokenFrom(data.Data)
			now := time.Now()
			select {
			case dev.out <- token{
				msg: []byte(fmt.Sprintf("%s: %q", now.Format("2006-01-02 15:04:05.9"), tok.msg)),
				beg: tok.beg,
				end: now,
			}:
			default:
			}
		case <-ctl.Done():
			return nil
		case <-dev.quit:
			return nil
		}
	}
}

// token is the data exchanged between devices.
type token struct {
	msg []byte
	beg time.Time
	end time.Time
}

func newToken(msg string) token {
	return token{
		msg: []byte(msg),
		beg: time.Now(),
	}
}

func (tok token) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, int64(len(tok.msg)))
	buf.Write(tok.msg)

	data, err := tok.beg.MarshalBinary()
	if err != nil {
		log.Printf("error marshaling token: %v", err)
		return buf.Bytes()
	}

	binary.Write(buf, binary.LittleEndian, int64(len(data)))
	buf.Write(data)

	return buf.Bytes()
}

func tokenFrom(data []byte) token {
	var tok token
	tok.msg = make([]byte, int(binary.LittleEndian.Uint64(data[:8])))
	data = data[8:]
	copy(tok.msg, data[:len(tok.msg)])
	data = data[len(tok.msg):]

	n := int(binary.LittleEndian.Uint64(data[:8]))
	data = data[8:]
	err := tok.beg.UnmarshalBinary(data[:n])
	if err != nil {
		log.Printf("error unmarshaling token: %v", err)
		return tok
	}
	data = data[n:]

	return tok
}

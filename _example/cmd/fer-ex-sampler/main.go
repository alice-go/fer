// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"time"

	"github.com/sbinet-alice/fer"
	"github.com/sbinet-alice/fer/config"
)

type sampler struct {
	cfg   config.Device
	datac chan fer.Msg
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
		case dev.datac <- fer.Msg{Data: []byte("HELLO")}:
			ctl.Printf("sent 'HELLO' (%v)\n", time.Now())
		case <-ctl.Done():
			return nil
		}
	}
}

func main() {
	err := fer.Main(&sampler{})
	if err != nil {
		log.Fatal(err)
	}
}

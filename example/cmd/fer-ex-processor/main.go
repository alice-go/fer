// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"

	"github.com/sbinet-alice/fer"
	"github.com/sbinet-alice/fer/config"
)

type Device struct {
	cfg    config.Device
	idatac chan fer.Msg
	odatac chan fer.Msg
}

func (dev *Device) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *Device) Init(ctl fer.Controler) error {
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

func (dev *Device) Run(ctl fer.Controler) error {
	for {
		select {
		case data := <-dev.idatac:
			log.Printf("received: %q\n", string(data.Data))
			out := append([]byte(nil), data.Data...)
			out = append(out, []byte(" (modified by "+dev.cfg.ID+")")...)
			dev.odatac <- fer.Msg{Data: out}
		case <-ctl.Done():
			return nil
		}
	}
	return nil
}

func (dev *Device) Pause(ctl fer.Controler) error {
	return nil
}

func (dev *Device) Reset(ctl fer.Controler) error {
	return nil
}

func main() {
	err := fer.Main(&Device{})
	if err != nil {
		log.Fatal(err)
	}
}

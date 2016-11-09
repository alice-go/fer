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
	cfg   config.Device
	datac chan fer.Msg
}

func (dev *Device) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *Device) Init(ctrl fer.Controler) error {
	datac, err := ctrl.Chan("data2", 0)
	if err != nil {
		return err
	}

	dev.datac = datac
	return nil
}

func (dev *Device) Run(ctrl fer.Controler) error {
	for {
		select {
		case data := <-dev.datac:
			log.Printf("received: %q\n", string(data.Data))
		case <-ctrl.Done():
			return nil
		}
	}
	return nil
}

func (dev *Device) Pause(ctrl fer.Controler) error {
	return nil
}

func (dev *Device) Reset(ctrl fer.Controler) error {
	return nil
}

func main() {
	err := fer.Main(&Device{})
	if err != nil {
		log.Fatal(err)
	}
}

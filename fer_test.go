// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fer

import (
	"log"
	"testing"

	"github.com/sbinet-alice/fer/config"
)

func TestSamplerProcessorSink(t *testing.T) {

}

type sampler struct {
	cfg   config.Device
	datac chan Msg
}

func (dev *sampler) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *sampler) Init(ctl Controler) error {
	datac, err := ctl.Chan("data1", 0)
	if err != nil {
		return err
	}

	dev.datac = datac
	return nil
}

func (dev *sampler) Run(ctl Controler) error {
	for {
		select {
		case dev.datac <- Msg{Data: []byte("HELLO")}:
		case <-ctl.Done():
			return nil
		}
	}
}

func (dev *sampler) Pause(ctl Controler) error {
	return nil
}

func (dev *sampler) Reset(ctl Controler) error {
	return nil
}

type processor struct {
	cfg    config.Device
	idatac chan Msg
	odatac chan Msg
}

func (dev *processor) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *processor) Init(ctl Controler) error {
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

func (dev *processor) Run(ctl Controler) error {
	for {
		select {
		case data := <-dev.idatac:
			log.Printf("received: %q\n", string(data.Data))
			out := append([]byte(nil), data.Data...)
			out = append(out, []byte(" (modified by "+dev.cfg.ID+")")...)
			dev.odatac <- Msg{Data: out}
		case <-ctl.Done():
			return nil
		}
	}
}

func (dev *processor) Pause(ctl Controler) error {
	return nil
}

func (dev *processor) Reset(ctl Controler) error {
	return nil
}

type sink struct {
	cfg   config.Device
	datac chan Msg
}

func (dev *sink) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *sink) Init(ctl Controler) error {
	datac, err := ctl.Chan("data2", 0)
	if err != nil {
		return err
	}

	dev.datac = datac
	return nil
}

func (dev *sink) Run(ctl Controler) error {
	for {
		select {
		case data := <-dev.datac:
			log.Printf("received: %q\n", string(data.Data))
		case <-ctl.Done():
			return nil
		}
	}
}

func (dev *sink) Pause(ctl Controler) error {
	return nil
}

func (dev *sink) Reset(ctl Controler) error {
	return nil
}

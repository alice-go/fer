// Copyright 2017 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image/color"
	"io"
	"log"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/pkg/profile"
	"github.com/sbinet-alice/fer"
	"go-hep.org/x/hep/hbook"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

var (
	timeout   = flag.Duration("timeout", 20*time.Second, "timeout for fer-pods")
	transport = flag.String("transport", "zeromq", "transport medium to use")
	protocol  = flag.String("protocol", "tcp", "protocol to use for transport")
	doprof    = flag.Bool("cpu-prof", false, "enable CPU profiling")
)

func main() {
	flag.Parse()

	if *doprof {
		defer profile.Start(profile.CPUProfile).Stop()
	}

	stdout := new(bytes.Buffer)
	datac := make(chan Data, 100)
	go runHelloWorld(stdout, os.Stdin, datac)

	h := hbook.NewH1D(100, 0, 500)
loop:
	for data := range datac {
		if data.quit {
			break loop
		}
		h.Fill(float64(data.delta)*1e-3, 1)
	}

	pl := hplot.New()
	pl.Title.Text = fmt.Sprintf("%s -- %s", *transport, *protocol)
	pl.X.Label.Text = "Time of Flight (us)"

	hh := hplot.NewH1D(h)
	hh.LineStyle.Color = color.RGBA{255, 0, 0, 255}
	hh.Infos.Style = hplot.HInfoSummary

	pl.Add(hh, plotter.NewGrid())

	oname := fmt.Sprintf("tof-%s-%s.png", *transport, *protocol)
	err := pl.Save(20*vg.Centimeter, -1, oname)
	if err != nil {
		log.Fatal(err)
	}
}

type Data struct {
	delta int64
	quit  bool
}

func runHelloWorld(w io.Writer, r io.Reader, datac chan Data) {
	cfg, err := getSPSConfig(*transport)
	if err != nil {
		log.Printf("error: %v\n", err)
		return
	}

	dev1 := &sink{out: make(chan token, 1), quit: make(chan int, 1)}
	dev2 := &processor{quit: make(chan int, 1)}
	dev3 := &sampler{quit: make(chan int, 1)}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	errc := make(chan error)
	go func() {
		cfg := cfg
		cfg.ID = "sink1"
		errc <- fer.RunDevice(ctx, cfg, dev1, r, w)
	}()

	go func() {
		cfg := cfg
		cfg.ID = "processor"
		errc <- fer.RunDevice(ctx, cfg, dev2, r, w)
	}()

	go func() {
		cfg := cfg
		cfg.ID = "sampler1"
		errc <- fer.RunDevice(ctx, cfg, dev3, r, w)
	}()

	i := 0
loop:
	for {
		select {
		case err := <-errc:
			log.Printf("error: %v", err)
			break loop
		case <-ctx.Done():
			log.Printf("time's up (%v)", ctx.Err())
			break loop
		case out := <-dev1.out:
			i++
			if i%500 == 0 {
				delta := out.end.Sub(out.beg)
				datac <- Data{
					delta: delta.Nanoseconds(),
				}
			}
		}
	}
	datac <- Data{quit: true}
}

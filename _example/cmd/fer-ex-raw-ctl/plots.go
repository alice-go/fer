// Copyright 2018 The fer Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"bufio"
	"fmt"
	"image/color"
	"log"
	"os"

	"go-hep.org/x/hep/hbook"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

func main() {
	for _, protocol := range []string{"tcp", "ipc", "inproc"} {
		doplot(protocol)
	}
}

var (
	colors = plotutil.SoftColors
)

func doplot(protocol string) {
	p := hplot.New()
	p.Title.Text = protocol
	p.X.Label.Text = "Time of Flight (us)"

	for i, trans := range []string{"zeromq", "nanomsg", "czmq"} {
		f, err := os.Open(fmt.Sprintf("tof-%s-%s.txt", trans, protocol))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		h := hbook.NewH1D(100, 0, 200)
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			var v float64
			fmt.Sscanf(sc.Text(), "%e", &v)
			h.Fill(v, 1)
		}

		hh := hplot.NewH1D(h)
		hh.LineStyle.Color = colors[i]
		hh.FillColor = color.Transparent
		hh.Infos.Style = hplot.HInfoNone

		p.Add(hh)
		p.Plot.Legend.Add(trans, hh)
	}

	p.Add(plotter.NewGrid())
	p.Plot.Legend.Top = true

	oname := fmt.Sprintf("result-%s.png", protocol)
	err := p.Save(20*vg.Centimeter, -1, oname)
	if err != nil {
		log.Fatal(err)
	}
}

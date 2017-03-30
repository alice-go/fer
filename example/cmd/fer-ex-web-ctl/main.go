// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"image/color"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-hep.org/x/hep/hbook"
	"go-hep.org/x/hep/hplot"

	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"github.com/gonum/plot/vg/vgsvg"
	"github.com/sbinet-alice/fer"
	"github.com/sbinet-alice/fer/config"
	"golang.org/x/net/websocket"
)

var (
	addr    = flag.String("addr", ":8080", "web server address")
	timeout = flag.Duration("timeout", 20*time.Second, "timeout for fer-pods")
	datac   = make(chan Data, 1024)
)

func main() {
	flag.Parse()

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/basic", basicHandler)
	http.Handle("/data", websocket.Handler(dataHandler))
	log.Panic(http.ListenAndServe(*addr, nil))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, homePage)
}

func basicHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("/basic...")
	//stdout := new(bytes.Buffer)
	stdin := new(bytes.Buffer)
	stdout := ioutil.Discard
	runHelloWorld(stdout, stdin, datac)
	fmt.Fprintf(w, "ok\n") //hello from /basic\n<code>%s</code>\n", string(stdout.Bytes()))
}

func dataHandler(ws *websocket.Conn) {
	defer ws.Close()

	lines := make([]string, 0, 30)
	h := hbook.NewH1D(100, 0, 5)
	state := Data{}
	for data := range datac {
		if data.quit {
			return
		}
		if n := len(lines); n == cap(lines) {
			copy(lines[2:n-1], lines[3:n])
			lines = lines[:n-1]
			lines[1] = "[...]"
		}
		lines = append(lines, data.Lines)
		h.Fill(float64(data.delta)*1e-9, 1)
		state.Lines = strings.Join(lines, "<br>")
		state.Plot = plotToF(h)

		err := websocket.JSON.Send(ws, state)
		if err != nil {
			log.Printf("error sending data: %v", err)
		}
	}
}

func plotToF(h *hbook.H1D) string {
	pl, err := hplot.New()
	if err != nil {
		log.Panic(err)
	}
	pl.X.Label.Text = "Time of Flight (s)"

	hh, err := hplot.NewH1D(h)
	if err != nil {
		log.Panic(err)
	}
	hh.LineStyle.Color = color.RGBA{255, 0, 0, 255}
	hh.Infos.Style = hplot.HInfoSummary

	pl.Add(hh, plotter.NewGrid())

	return renderSVG(pl)
}

func renderSVG(p *hplot.Plot) string {
	size := 20 * vg.Centimeter
	canvas := vgsvg.New(size, size/vg.Length(math.Phi))
	p.Draw(draw.New(canvas))
	out := new(bytes.Buffer)
	_, err := canvas.WriteTo(out)
	if err != nil {
		panic(err)
	}
	return string(out.Bytes())
}

type Data struct {
	Lines string `json:"lines"`
	Plot  string `json:"plot"`
	delta int64
	quit  bool
}

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

func runHelloWorld(w io.Writer, r io.Reader, datac chan Data) {
	cfg, err := getSPSConfig("nanomsg")
	if err != nil {
		log.Printf("error: %v\n", err)
		return
	}

	dev1 := &sink{out: make(chan token, 1), quit: make(chan int, 1)}
	dev2 := &processor{quit: make(chan int, 1)}
	dev3 := &sampler{quit: make(chan int, 1)}

	pr1 := new(bytes.Buffer)
	pr2 := new(bytes.Buffer)
	pr3 := new(bytes.Buffer)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	errc := make(chan error)
	go func() {
		cfg := cfg
		cfg.ID = "sink1"
		errc <- fer.RunDevice(ctx, cfg, dev1, pr1, w)
	}()

	go func() {
		cfg := cfg
		cfg.ID = "processor"
		errc <- fer.RunDevice(ctx, cfg, dev2, pr2, w)
	}()

	go func() {
		cfg := cfg
		cfg.ID = "sampler1"
		errc <- fer.RunDevice(ctx, cfg, dev3, pr3, w)
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
					Lines: string(out.msg),
					delta: delta.Nanoseconds(),
				}
			}
		}
	}
	datac <- Data{Lines: fmt.Sprintf("%8d msgs processed by %q", dev3.n, dev3.cfg.Name())}
	datac <- Data{Lines: fmt.Sprintf("%8d msgs processed by %q", dev2.n, dev2.cfg.Name())}
	datac <- Data{Lines: fmt.Sprintf("%8d msgs processed by %q", dev1.n, dev1.cfg.Name())}
	datac <- Data{Lines: "DONE"}
	datac <- Data{quit: true}
}

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

const homePage = `<html>
<head>
	<title>Web-based fer</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link rel="stylesheet" href="https://www.w3schools.com/w3css/3/w3.css">
	<script src="https://ajax.googleapis.com/ajax/libs/jquery/3.1.1/jquery.min.js"></script>
</head>

<script type="text/javascript">
	function update(data) {
		var node = document.getElementById("device-log");
		node.innerHTML = data["lines"];
		node.scrollTop = node.scrollHeight;

		var plot = document.getElementById("plot");
		plot.innerHTML = data["plot"];
	};

	function runBasic() {
		var sock = new WebSocket("ws://"+location.host+"/data");
		sock.onopen = function(){};
		sock.onclose = function(){console.log("closing...");};
		sock.onmessage = function(event) {
			var data = JSON.parse(event.data);
			update(data);
		};
		$.ajax({
			url: "/basic",
			method: "POST",
			processData: false,
			contentData: false,
			success: function() {},
			error: function(err) {
				alert("/basic failed: "+JSON.stringify(err));
			}
		});
	};
</script>
<body>

<header class="w3-container w3-black">
    <h1>Control panel</h1>
	<div class="w3-container">
		<button class="w3-button w3-blue" onclick="runBasic();">Launch</button>
	</div>
</header>

<div class="w3-cell-row">
  <div class="w3-container w3-light-gray w3-cell w3-mobile">
	<div class="w3-cell-row">
		<div id="device" class="w3-container w3-cell">
		<div id="device-log"></div>
		</div>
		<div class="w3-cell-row w3-display-center w3-panel w3-container">
			<div class="w3-container w-cell w3-center w3-display-center">
				<div id="plot" class="w3-container w3-card-4 w3-center w3-white">
				</div>
			</div>
		</div>
	</div>
  </div>
</div>

</body>
</html>
`

func getTCPPort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

func getSPSConfig(transport string) (config.Config, error) {
	var cfg config.Config

	port1, err := getTCPPort()
	if err != nil {
		return cfg, fmt.Errorf("error getting free TCP port: %v\n", err)
	}
	port2, err := getTCPPort()
	if err != nil {
		return cfg, fmt.Errorf("error getting free TCP port: %v\n", err)
	}

	cfg = config.Config{
		Control:   "interactive",
		Transport: transport,
		Options: config.Options{
			Devices: []config.Device{
				{
					ID: "sampler1",
					Channels: []config.Channel{
						{
							Name: "data1",
							Sockets: []config.Socket{
								{
									Type:    "push",
									Method:  "bind",
									Address: "tcp://*:" + port1,
								},
							},
						},
					},
				},
				{
					Key: "processor",
					Channels: []config.Channel{
						{
							Name: "data1",
							Sockets: []config.Socket{
								{
									Type:    "pull",
									Method:  "connect",
									Address: "tcp://localhost:" + port1,
								},
							},
						},
						{
							Name: "data2",
							Sockets: []config.Socket{
								{
									Type:    "push",
									Method:  "connect",
									Address: "tcp://localhost:" + port2,
								},
							},
						},
					},
				},
				{
					ID: "sink1",
					Channels: []config.Channel{
						{
							Name: "data2",
							Sockets: []config.Socket{
								{
									Type:    "pull",
									Method:  "bind",
									Address: "tcp://*:" + port2,
								},
							},
						},
					},
				},
			},
		},
	}

	return cfg, nil
}

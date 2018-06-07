// Copyright 2017 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"image/color"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"sync"
	"time"

	"github.com/sbinet-alice/fer"
	"go-hep.org/x/hep/hbook"
	"go-hep.org/x/hep/hplot"
	"golang.org/x/net/websocket"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgsvg"
)

const cookieName = "FER_WEB_SRV"

var (
	addr    = flag.String("addr", ":8080", "web server address")
	timeout = flag.Duration("timeout", 20*time.Second, "timeout for fer-pods")
)

type server struct {
	mu      sync.RWMutex
	cookies map[string]*http.Cookie
	datac   map[string]chan Data
}

func main() {
	flag.Parse()

	srv := newServer()

	http.HandleFunc("/", srv.wrap(srv.rootHandler))
	http.HandleFunc("/run", srv.wrap(srv.runHandler))
	http.Handle("/data", websocket.Handler(srv.dataHandler))
	log.Panic(http.ListenAndServe(*addr, nil))
}

func newServer() *server {
	srv := server{
		cookies: make(map[string]*http.Cookie),
		datac:   make(map[string]chan Data),
	}
	go srv.run()
	return &srv
}

func (srv *server) run() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		srv.gc()
	}
}

func (srv *server) gc() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	for name, cookie := range srv.cookies {
		now := time.Now()
		if now.After(cookie.Expires) {
			close(srv.datac[name])
			delete(srv.datac, name)
			delete(srv.cookies, name)
			cookie.MaxAge = -1
		}
	}
}

func (srv *server) wrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := srv.setCookie(w, r)
		if err != nil {
			log.Printf("error retrieving cookie: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fn(w, r)
	}
}

func (srv *server) setCookie(w http.ResponseWriter, r *http.Request) error {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	cookie, err := r.Cookie(cookieName)
	if err != nil && err != http.ErrNoCookie {
		return err
	}

	if cookie != nil {
		if v, ok := srv.datac[cookie.Value]; v == nil || !ok {
			srv.datac[cookie.Value] = make(chan Data, 1024)
			srv.cookies[cookie.Value] = cookie
		}
		return nil
	}

	secret := make([]byte, 256)
	_, err = rand.Read(secret)
	if err != nil {
		return err
	}

	cookie = &http.Cookie{
		Name:    cookieName,
		Value:   string(secret),
		Expires: time.Now().Add(24 * time.Hour),
	}
	srv.datac[cookie.Value] = make(chan Data, 1024)
	srv.cookies[cookie.Value] = cookie
	http.SetCookie(w, cookie)
	return nil
}

func (srv *server) cookie(r *http.Request) (*http.Cookie, error) {
	srv.mu.RLock()
	defer srv.mu.RUnlock()
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}

	if cookie == nil {
		return nil, http.ErrNoCookie
	}
	return srv.cookies[cookie.Value], nil
}

func (srv *server) data(r *http.Request) (chan Data, error) {
	srv.mu.RLock()
	defer srv.mu.RUnlock()
	cookie, err := srv.cookie(r)
	if err != nil {
		return nil, err
	}
	return srv.datac[cookie.Value], nil
}

func (srv *server) rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, homePage)
}

func (srv *server) runHandler(w http.ResponseWriter, r *http.Request) {
	stdin := new(bytes.Buffer)
	stdout := ioutil.Discard
	datac, err := srv.data(r)
	if err != nil {
		log.Printf("error retrieving data channel: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	runHelloWorld(stdout, stdin, datac)
	fmt.Fprintf(w, "ok\n")
}

func (srv *server) dataHandler(ws *websocket.Conn) {
	defer ws.Close()

	datac, err := srv.data(ws.Request())
	if err != nil {
		log.Printf("error retrieving data channel: %v\n", err)
		return
	}

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
	pl := hplot.New()
	pl.X.Label.Text = "Time of Flight (s)"

	hh := hplot.NewH1D(h)
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

func runHelloWorld(w io.Writer, r io.Reader, datac chan Data) {
	cfg, err := getSPSConfig("nanomsg")
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

	function run() {
		var sock = new WebSocket("ws://"+location.host+"/data");
		sock.onopen = function(){};
		sock.onclose = function(){console.log("closing...");};
		sock.onmessage = function(event) {
			var data = JSON.parse(event.data);
			update(data);
		};
		$.ajax({
			url: "/run",
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
		<button class="w3-button w3-blue" onclick="run();">Launch</button>
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

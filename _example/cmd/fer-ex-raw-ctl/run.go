// +build ignore

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
)

type stats struct {
	prot, trans     string
	real, user, sys float64
	cpu             float64
	maxrss          int64
}

type statSlice []stats

func (p statSlice) Len() int      { return len(p) }
func (p statSlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p statSlice) Less(i, j int) bool {
	pi := p[i]
	pj := p[j]
	if pi.prot == pj.prot {
		return pi.trans < pj.trans
	}
	return pi.prot < pj.prot
}

func main() {
	build()

	os.Setenv("GODEBUG", "cgocheck=0")
	var (
		wg  sync.WaitGroup
		out = make(chan stats)
		vs  []stats
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range out {
			vs = append(vs, v)
		}
	}()

	for _, trans := range []string{"zeromq", "nanomsg", "czmq"} {
		var wg sync.WaitGroup
		for _, prot := range []string{"tcp", "ipc", "inproc"} {
			wg.Add(1)
			run(&wg, prot, trans, out)
		}
		wg.Wait()
	}
	close(out)
	wg.Wait()

	sort.Sort(statSlice(vs))
	for _, v := range vs {
		fmt.Printf("%#v\n", v)
	}
}

func build() {
	cmd := exec.Command("go", "build", "-v", "-tags=czmq")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	err := cmd.Run()
	if err != nil {
		log.Panic(err)
	}
}

func cleanup() {
	err := exec.Command("/bin/rm", "-rf", "./raw-ctl-p*").Run()
	if err != nil {
		log.Panic(err)
	}
}

func run(wg *sync.WaitGroup, prot, trans string, ch chan stats) {
	defer wg.Done()
	stdout := new(bytes.Buffer)
	cmd := exec.Command(
		"/usr/bin/time", "-f", "real=%e user=%U sys=%S CPU=%P MaxRSS=%M I/O=%I/%O",
		"./fer-ex-raw-ctl", "-protocol", prot, "-transport", trans,
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = stdout
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	err := cmd.Run()
	if err != nil {
		log.Fatalf("%s-%s: %v", prot, trans, err)
	}

	st := stats{trans: trans, prot: prot}
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		if !strings.HasPrefix(sc.Text(), "real=") {
			continue
		}
		var i, o int
		_, err := fmt.Sscanf(
			sc.Text(),
			"real=%f user=%f sys=%f CPU=%f%% MaxRSS=%d I/O=%d/%d",
			&st.real, &st.user, &st.sys, &st.cpu, &st.maxrss, &i, &o,
		)
		if err != nil {
			log.Fatalf("%s-%s: %v (%s)", prot, trans, err, sc.Text())
		}
	}
	ch <- st
}

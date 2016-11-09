// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// fer-json-fmt format JSON configuration files following the one true style.
package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"strings"

	"github.com/sbinet-alice/fer/config"
)

func main() {
	inplace := flag.Bool("w", false, "write formatted JSON file in-place")

	flag.Parse()
	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(2)
	}

	log.SetFlags(0)
	log.SetPrefix("fer-json-fmt: ")

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var cfg config.Config
	err = json.NewDecoder(f).Decode(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}

	var w io.Writer = os.Stdout
	if *inplace {
		f, err = os.Create(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		w = f
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", strings.Repeat(" ", 4))
	err = enc.Encode(cfg)
	if err != nil {
		log.Fatal(err)
	}
}

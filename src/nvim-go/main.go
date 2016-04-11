// Copyright 2016 Koichi Shiraishi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command nvimgo is a Neovim remote plogin.
package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/neovim-go/vim"
	"github.com/garyburd/neovim-go/vim/plugin"

	_ "nvim-go/cgo"
	_ "nvim-go/commands"
	_ "nvim-go/nvim"
)

func init() {
	// logrus instead of stdlib log
	// neovim-go hijacked log format
	if lf := os.Getenv("NEOVIM_GO_LOG_FILE"); lf != "" {
		f, err := os.OpenFile(lf, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}

		log.SetOutput(f)
		log.SetFormatter(&RpluginFormatter{})
		log.SetLevel(log.DebugLevel)
	}

	plugin.HandleAutocmd("BufWinEnter", &plugin.AutocmdOptions{Pattern: "*.go", Eval: "g:go#debug#pprof"}, pprofDebug)
}

func main() {
	plugin.Main()
}

func pprofDebug(v *vim.Vim, flag int64) {
	if flag != int64(0) {
		fmt.Printf("Start pprof debug\n")
		go func() {
			log.Println(http.ListenAndServe("0.0.0.0:6060", http.DefaultServeMux))
		}()
	}
}

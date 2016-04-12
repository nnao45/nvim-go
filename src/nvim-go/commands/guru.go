// Copyright 2016 Koichi Shiraishi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// guru: a tool for answering questions about Go source code.
//
//    http://golang.org/s/oracle-design
//    http://golang.org/s/oracle-user-manual

package commands

import (
	"errors"
	"fmt"
	"go/build"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/neovim-go/vim"
	"github.com/garyburd/neovim-go/vim/plugin"
	"golang.org/x/tools/cmd/guru/serial"

	"nvim-go/gb"
	"nvim-go/guru"
	"nvim-go/nvim"
)

func init() {
	plugin.HandleCommand("GoGuru",
		&plugin.CommandOptions{
			NArgs: "+", Complete: "customlist,GuruCompletelist", Eval: "[expand('%:p:h'), expand('%:p')]"},
		cmdGuru)
	plugin.HandleFunction("GuruCompletelist", &plugin.FunctionOptions{}, onComplete)
}

var (
	guruReflection  = "go#guru#reflection"
	vGuruReflection interface{}
	guruKeepCursor  = "go#guru#keep_cursor"
	vGuruKeepCursor interface{}
	guruDebug       = "go#debug"
	vGuruDebug      interface{}
)

type onGuruEval struct {
	Cwd  string `msgpack:",array"`
	File string
}

func cmdGuru(v *vim.Vim, args []string, eval *onGuruEval) {
	go Guru(v, args, eval)
}

func Guru(v *vim.Vim, args []string, eval *onGuruEval) error {
	defer gb.WithGoBuildForPath(eval.Cwd)()

	useReflection := false
	v.Var(guruReflection, &vGuruReflection)
	if vGuruReflection.(int64) == int64(1) {
		useReflection = true
	}
	useKeepCursor := false
	v.Var(guruKeepCursor, &vGuruKeepCursor)
	if vGuruKeepCursor.(int64) == int64(1) {
		useKeepCursor = true
	}
	debug := false
	v.Var(guruDebug, &vGuruDebug)
	if vGuruDebug.(int64) == int64(1) {
		debug = true
	}

	var (
		b vim.Buffer
		w vim.Window
	)
	p := v.NewPipeline()
	p.CurrentBuffer(&b)
	p.CurrentWindow(&w)
	if err := p.Wait(); err != nil {
		return err
	}

	dir := strings.Split(eval.Cwd, "src/")
	scopeFlag := dir[len(dir)-1]

	mode := args[0]

	pos, err := nvim.ByteOffset(p)
	if err != nil {
		return nvim.Echomsg(v, "%v", err)
	}
	nvim.Debugln(v, debug, pos)

	query := guru.Query{
		Mode:       mode,
		Pos:        eval.File + ":#" + strconv.FormatInt(int64(pos), 10),
		Build:      &build.Default,
		Scope:      []string{scopeFlag},
		Reflection: useReflection,
	}

	nvim.Echohl(v, "GoGuru: ", "Identifier", "analysing %s ...", mode)

	if err := guru.Run(&query); err != nil {
		return nvim.Echomsg(v, "%s", err)
	}

	d, err := parseSerial(mode, query.Serial())
	if err != nil {
		return nvim.Echomsg(v, "GoGuru: %v", err)
	}
	nvim.Debugln(v, debug, d)

	if err := nvim.SetLoclist(p, d); err != nil {
		return nvim.Echomsg(v, "GoGuru: %v", err)
	}
	p.Command("redraw!")
	return nvim.OpenLoclist(p, w, d, useKeepCursor)
}

func parseSerial(mode string, s *serial.Result) ([]*nvim.ErrorlistData, error) {
	var loclist []*nvim.ErrorlistData

	switch mode {
	case "callees":
		var calleers string
		for _, n := range s.Callees.Callees {
			calleers += calleers + n.Name
		}
		file, line, col := nvim.SplitPos(s.Callees.Pos)
		loclist = append(loclist, &nvim.ErrorlistData{
			FileName: file,
			LNum:     line,
			Col:      col,
			Text:     s.Callees.Desc + " " + calleers,
		})
	case "callers":
		for _, e := range s.Callers {
			file, line, col := nvim.SplitPos(e.Pos)
			loclist = append(loclist, &nvim.ErrorlistData{
				FileName: file,
				LNum:     line,
				Col:      col,
				Text:     e.Desc + " " + e.Caller,
			})
		}
	case "callstack":
		if len(s.Callstack.Callers) != 0 {
			for _, n := range s.Callstack.Callers {
				file, line, col := nvim.SplitPos(n.Pos)
				loclist = append(loclist, &nvim.ErrorlistData{
					FileName: file,
					LNum:     line,
					Col:      col,
					Text:     n.Desc,
				})
			}
		}
	case "definition":
		file, line, col := nvim.SplitPos(s.Definition.ObjPos)
		loclist = append(loclist, &nvim.ErrorlistData{
			FileName: file,
			LNum:     line,
			Col:      col,
			Text:     s.Definition.Desc,
		})
	case "describe":
		file, line, col := nvim.SplitPos(s.Describe.Value.ObjPos)
		loclist = append(loclist, &nvim.ErrorlistData{
			FileName: file,
			LNum:     line,
			Col:      col,
			Text:     s.Describe.Value.Type,
		})
	case "freevars":
		for _, e := range s.Freevars {
			file, line, col := nvim.SplitPos(e.Pos)
			log.Debugln(e)
			loclist = append(loclist, &nvim.ErrorlistData{
				FileName: file,
				LNum:     line,
				Col:      col,
				Text:     e.Type + "\n" + e.Kind + "\n" + e.Ref,
			})
		}
	case "implements":
		file, line, col := nvim.SplitPos(s.Implements.T.Pos)
		loclist = append(loclist, &nvim.ErrorlistData{
			FileName: file,
			LNum:     line,
			Col:      col,
			Text:     s.Implements.T.Name,
		})
	case "peers":
		for _, e := range s.Peers.Allocs {
			file, line, col := nvim.SplitPos(e)
			loclist = append(loclist, &nvim.ErrorlistData{
				FileName: file,
				LNum:     line,
				Col:      col,
				Text:     s.Peers.Type + ": Allocs",
			})
		}
		for _, e := range s.Peers.Sends {
			file, line, col := nvim.SplitPos(e)
			loclist = append(loclist, &nvim.ErrorlistData{
				FileName: file,
				LNum:     line,
				Col:      col,
				Text:     s.Peers.Type + ": Sends",
			})
		}
		for _, e := range s.Peers.Receives {
			file, line, col := nvim.SplitPos(e)
			loclist = append(loclist, &nvim.ErrorlistData{
				FileName: file,
				LNum:     line,
				Col:      col,
				Text:     s.Peers.Type + ": Receives",
			})
		}
		for _, e := range s.Peers.Closes {
			file, line, col := nvim.SplitPos(e)
			loclist = append(loclist, &nvim.ErrorlistData{
				FileName: file,
				LNum:     line,
				Col:      col,
				Text:     s.Peers.Type + ": Closes",
			})
		}
	case "pointsto":
		for _, e := range s.PointsTo {
			if e.NamePos != "" {
				file, line, col := nvim.SplitPos(e.NamePos)
				loclist = append(loclist, &nvim.ErrorlistData{
					FileName: file,
					LNum:     line,
					Col:      col,
					Text:     e.Type,
				})
			} else {
				loclist = append(loclist, &nvim.ErrorlistData{
					Text: e.Type,
				})
			}
		}
	case "referrers":
		for _, e := range s.Referrers.Refs {
			file, line, col := nvim.SplitPos(e)
			loclist = append(loclist, &nvim.ErrorlistData{
				FileName: file,
				LNum:     line,
				Col:      col,
				Text:     s.Referrers.Desc,
			})
		}
	case "what":
		log.Debugln("s.What.Enclosing:", s.What.Enclosing)
		log.Debugln("s.What.Modes:", s.What.Modes)
		log.Debugln("s.What.SrcDir:", s.What.SrcDir)
		log.Debugln("s.What.ImportPath:", s.What.ImportPath)
		log.Debugln("s.What.Object:", s.What.Object)
		log.Debugln("s.What.SameIDs:", s.What.SameIDs)
		var modesText string
		for _, mode := range s.What.Modes {
			modesText += mode + " "
		}
		loclist = append(loclist, &nvim.ErrorlistData{
			Text: "Modes: " + modesText[:len(modesText)-2],
		})
		loclist = append(loclist, &nvim.ErrorlistData{
			Text: "SrcDir: " + s.What.SrcDir,
		})
		loclist = append(loclist, &nvim.ErrorlistData{
			Text: "ImportPath: " + s.What.ImportPath,
		})
		loclist = append(loclist, &nvim.ErrorlistData{
			Text: "Object: " + s.What.Object,
		})
		sameIDsText := "SameIDs: "
		for _, sameid := range s.What.SameIDs {
			sameIDsText += sameid
		}
		loclist = append(loclist, &nvim.ErrorlistData{
			Text: sameIDsText,
		})
	case "whicherrs":
		// log.Debugln("s.WhichErrs.ErrPos:", s.WhichErrs.ErrPos)
		// log.Debugln("s.WhichErrs.Globals:", s.WhichErrs.Globals)
		// log.Debugln("s.WhichErrs.Constants:", s.WhichErrs.Constants)
		// log.Debugln("s.WhichErrs.Types:", s.WhichErrs.Types)
		for _, e := range s.WhichErrs.Types {
			file, line, col := nvim.SplitPos(e.Position)
			loclist = append(loclist, &nvim.ErrorlistData{
				FileName: file,
				LNum:     line,
				Col:      col,
				Text:     e.Type,
			})
		}
	}

	if len(loclist) == 0 {
		return loclist, errors.New(fmt.Sprintf("%s not fount", mode))
	}
	return loclist, nil
}

func onComplete(v *vim.Vim) ([]string, error) {
	return []string{
		"callers",
		"callees",
		"callstack",
		"definition",
		"describe",
		"freevars",
		"implements",
		"peers",
		"pointsto",
		"referrers",
		"what",
		"whicherrs",
	}, nil
}

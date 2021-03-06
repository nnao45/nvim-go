// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nvimutil

import (
	"fmt"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/pkg/errors"
)

// Fade represents a Fade highlighting.
type Fade struct {
	n              *nvim.Nvim
	buffer         nvim.Buffer
	hlGroup        string
	startLine      int
	endLine        int
	startCol       int
	endCol         int
	duration       time.Duration
	timingFunction string // WIP
}

// NewFader returns a new Fade.
func NewFader(n *nvim.Nvim, buffer nvim.Buffer, hlGroup string, startLine, endLine, startCol, endCol int, duration int) *Fade {
	return &Fade{
		n:         n,
		buffer:    buffer,
		hlGroup:   hlGroup,
		startLine: startLine,
		endLine:   endLine,
		startCol:  startCol,
		endCol:    endCol,
		duration:  time.Duration(int64(duration)),
	}
}

// SetHighlight sets the highlight to at once.
func (f *Fade) SetHighlight() error {
	if f.startLine == f.endLine {
		if _, err := f.n.AddBufferHighlight(f.buffer, 0, f.hlGroup, f.startLine, f.startCol, f.endCol); err != nil {
			return ErrorWrap(f.n, errors.WithStack(err))
		}
		return nil
	}

	for i := f.startLine; i < f.endLine; i++ {
		if _, err := f.n.AddBufferHighlight(f.buffer, 0, f.hlGroup, f.startLine, f.startCol, f.endCol); err != nil {
			return ErrorWrap(f.n, errors.WithStack(err))
		}
	}
	return nil
}

// FadeOut fade out the highlights.
func (f *Fade) FadeOut() error {
	var srcID int

	for i := 1; i < 5; i++ {
		if srcID != 0 {
			f.n.ClearBufferHighlight(f.buffer, srcID, f.startLine, -1)
		}
		srcID, _ = f.n.AddBufferHighlight(f.buffer, 0, fmt.Sprintf("%s%d", f.hlGroup, i), f.startLine, f.startCol, f.endCol)

		time.Sleep(time.Duration(f.duration * time.Millisecond))
	}
	f.n.ClearBufferHighlight(f.buffer, srcID, f.startLine, -1)

	return nil
}

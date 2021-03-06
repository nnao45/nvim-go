// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ctx

import (
	"fmt"
	"go/build"
	"path/filepath"
	"strings"
)

// GbJoinPath joins the sequence of path fragments into a single path for build.Default.JoinPath.
func (ctx *Build) GbJoinPath(elem ...string) string {
	res := filepath.Join(elem...)

	if gbrel, err := filepath.Rel(ctx.ProjectRoot, res); err == nil {
		gbrel = filepath.ToSlash(gbrel)
		gbrel, _ = match(gbrel, "vendor/")
		if gbrel, ok := match(gbrel, fmt.Sprintf("pkg/%s_%s", build.Default.GOOS, build.Default.GOARCH)); ok {
			gbrel, hasSuffix := match(gbrel, "_")

			if hasSuffix {
				gbrel = "-" + gbrel
			}
			gbrel = fmt.Sprintf("pkg/%s-%s/", build.Default.GOOS, build.Default.GOARCH) + gbrel
			gbrel = filepath.FromSlash(gbrel)
			res = filepath.Join(ctx.ProjectRoot, gbrel)
		}
	}

	return res
}

func match(s, prefix string) (string, bool) {
	rest := strings.TrimPrefix(s, prefix)
	return rest, len(rest) < len(s)
}

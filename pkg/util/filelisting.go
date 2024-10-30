// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// ListFiles prints all files in a given directory to the provided writer
func ListFiles(w io.Writer, dir string) error {
	t := table.NewWriter()
	defer t.Render()

	t.SetOutputMirror(w)
	t.SetColumnConfigs([]table.ColumnConfig{{Number: 5, Align: text.AlignRight}})
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false

	return filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Printf("ignoring error for path %s: %v\n", path, err)
			return nil
		}

		// Omit the target path itself
		if path == dir {
			return nil
		}

		// use relative paths to keep output compact
		if l, err := filepath.Rel(dir, path); err == nil {
			path = l
		}

		// if possible, try to obtain nlink count and user/group details
		var nlink, user, group string = "?", "?", "?"
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			user = strconv.FormatUint(uint64(stat.Uid), 10)
			group = strconv.FormatUint(uint64(stat.Gid), 10)
			nlink = strconv.FormatUint(uint64(stat.Nlink), 10)
		}

		t.AppendRow(table.Row{
			filemode(info),
			nlink,
			user,
			group,
			humanReadableSize(info.Size()),
			path,
		})

		return nil
	})
}

// filemode is a minimal effort function to translate os.FileMode to the
// commonly known human representation, i.e. rw-r--r--. However, it does
// not implement all features such as sticky bits.
func filemode(info fs.FileInfo) string {
	var translate = func(i os.FileMode) string {
		var result = []rune{'-', '-', '-'}
		if i&0x1 != 0 {
			result[2] = 'x'
		}

		if i&0x2 != 0 {
			result[1] = 'w'
		}

		if i&0x4 != 0 {
			result[0] = 'r'
		}

		return string(result)
	}

	var dirBit = func(i os.FileMode) string {
		if i&fs.ModeDir != 0 {
			return "d"
		}

		return "-"
	}

	var mode = info.Mode()
	return dirBit(mode) + translate((mode>>6)&0x7) + translate((mode>>3)&0x7) + translate(mode&0x7)
}

// humanReadableSize is a minimal effort function to return a human readable
// size of the given number of bytes in a compact form
func humanReadableSize(bytes int64) string {
	value := float64(bytes)

	var mods = []string{"B", "K", "M", "G", "T"}
	var i int
	for value > 1023.9 {
		value /= 1024.0
		i++
	}

	return strings.TrimRight(fmt.Sprintf("%.1f", value), ".0") + mods[i]
}

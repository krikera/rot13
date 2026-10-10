//go:build !windows

package main

import (
	"os"
	"syscall"
)

func preserveOwner(tmpFile *os.File, info os.FileInfo) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		_ = tmpFile.Chown(int(stat.Uid), int(stat.Gid))
	}
}

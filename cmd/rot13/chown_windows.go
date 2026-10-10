//go:build windows

package main

import (
	"os"
)

func preserveOwner(tmpFile *os.File, info os.FileInfo) {
	// No-op on Windows
}

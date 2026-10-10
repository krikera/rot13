package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"rot13/pkg/rot13"
)

var (
	version        = "0.1.0"
	activeTempLock sync.Mutex
	activeTemp     string
)

func setActiveTemp(path string) {
	activeTempLock.Lock()
	activeTemp = path
	activeTempLock.Unlock()
}

func getActiveTemp() string {
	activeTempLock.Lock()
	defer activeTempLock.Unlock()
	return activeTemp
}

func isPipeError(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}

func initSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		path := getActiveTemp()
		if path != "" {
			_ = os.Remove(path)
		}
		os.Exit(130)
	}()
}

func run(args []string, defaultStdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("rot13", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var outFile string
	var inPlace, showVer bool

	fs.StringVar(&outFile, "o", "", "write output to `file`")
	fs.BoolVar(&inPlace, "i", false, "modify files in-place")
	fs.BoolVar(&showVer, "v", false, "print version")
	fs.BoolVar(&showVer, "version", false, "print version")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: rot13 [OPTIONS] [FILE...]\n\n"+
			"ROT13 encoder and decoder.\n\n"+
			"Options:\n"+
			"  -o <file>       Write output to specified file\n"+
			"  -i              Modify file(s) in-place\n"+
			"  -v, --version   Display version information and exit\n"+
			"  -h, --help      Display this help message and exit\n\n"+
			"Examples:\n"+
			"  rot13 secret.txt\n"+
			"  echo \"hello\" | rot13\n"+
			"  rot13 -i secret.txt\n"+
			"  rot13 -o encrypted.txt document.txt\n")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}

	if showVer {
		fmt.Fprintf(stdout, "rot13 version %s\n", version)
		return 0
	}

	if inPlace && outFile != "" {
		fmt.Fprintf(stderr, "rot13: flags -i (in-place) and -o (output file) cannot be used together\n")
		return 1
	}

	files := fs.Args()
	if inPlace {
		if len(files) == 0 || (len(files) == 1 && files[0] == "-") {
			fmt.Fprintf(stderr, "rot13: -i (in-place) cannot be used with standard input\n")
			return 1
		}
		exitCode := 0
		for _, f := range files {
			if err := processInPlace(f); err != nil {
				fmt.Fprintf(stderr, "rot13: %v\n", err)
				exitCode = 1
			}
		}
		return exitCode
	}

	var out io.Writer = stdout
	var tmpOutFile *os.File
	var tmpOutName string
	var origOutStat os.FileInfo

	if outFile != "" {
		for _, f := range files {
			if f == "-" {
				continue
			}
			if f == outFile {
				fmt.Fprintf(stderr, "rot13: output file %q is also an input file (use -i for in-place modification)\n", outFile)
				return 1
			}
			outStat, outErr := os.Stat(outFile)
			inStat, inErr := os.Stat(f)
			if outErr == nil && inErr == nil && os.SameFile(outStat, inStat) {
				fmt.Fprintf(stderr, "rot13: output file %q is also an input file (use -i for in-place modification)\n", outFile)
				return 1
			}
		}

		outStat, outErr := os.Stat(outFile)
		if outErr == nil {
			origOutStat = outStat
		}

		if origOutStat != nil && !origOutStat.Mode().IsRegular() {
			f, err := os.OpenFile(outFile, os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(stderr, "rot13: failed to open output file %s: %v\n", outFile, err)
				return 1
			}
			defer f.Close()
			out = f
		} else {
			dir := filepath.Dir(outFile)
			tmpFile, err := os.CreateTemp(dir, "rot13-out-*")
			if err != nil {
				fmt.Fprintf(stderr, "rot13: failed to create temporary output file: %v\n", err)
				return 1
			}
			tmpOutFile = tmpFile
			tmpOutName = tmpFile.Name()
			setActiveTemp(tmpOutName)
			out = tmpFile
		}
	}

	if len(files) == 0 {
		files = []string{"-"}
	}

	exitCode := 0
	for _, filePath := range files {
		var in io.Reader
		var closer io.Closer

		if filePath == "-" {
			in = defaultStdin
			if in == nil {
				in = os.Stdin
			}
		} else {
			info, err := os.Stat(filePath)
			if err != nil {
				fmt.Fprintf(stderr, "rot13: cannot open %s: %v\n", filePath, err)
				exitCode = 1
				continue
			}
			if info.IsDir() {
				fmt.Fprintf(stderr, "rot13: %s: Is a directory\n", filePath)
				exitCode = 1
				continue
			}
			f, err := os.Open(filePath)
			if err != nil {
				fmt.Fprintf(stderr, "rot13: cannot open %s: %v\n", filePath, err)
				exitCode = 1
				continue
			}
			in, closer = f, f
		}

		err := rot13.TransformStream(in, out)
		if closer != nil {
			closer.Close()
		}
		if err != nil {
			if isPipeError(err) {
				if tmpOutFile != nil {
					tmpOutFile.Close()
					_ = os.Remove(tmpOutName)
					setActiveTemp("")
				}
				return 0
			}
			fmt.Fprintf(stderr, "rot13: stream error: %v\n", err)
			exitCode = 1
		}
	}

	if tmpOutFile != nil {
		tmpOutFile.Close()
		if exitCode != 0 {
			_ = os.Remove(tmpOutName)
			setActiveTemp("")
			return exitCode
		}
		if origOutStat != nil {
			_ = os.Chmod(tmpOutName, origOutStat.Mode().Perm())
		}
		if err := os.Rename(tmpOutName, outFile); err != nil {
			_ = os.Remove(tmpOutName)
			setActiveTemp("")
			fmt.Fprintf(stderr, "rot13: failed to update output file %s: %v\n", outFile, err)
			return 1
		}
		setActiveTemp("")
	}

	return exitCode
}

func processInPlace(filePath string) error {
	target, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		target = filePath
	}

	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", filePath)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", filePath)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(target), "rot13-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpName := tmpFile.Name()
	setActiveTemp(tmpName)
	defer func() {
		tmpFile.Close()
		path := getActiveTemp()
		if path != "" {
			_ = os.Remove(path)
			setActiveTemp("")
		}
	}()

	srcFile, err := os.Open(target)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer srcFile.Close()

	if err := rot13.TransformStream(srcFile, tmpFile); err != nil {
		return fmt.Errorf("failed to transform %s: %w", filePath, err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	if err := tmpFile.Chmod(info.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}
	preserveOwner(tmpFile, info)
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to flush temp file: %w", err)
	}
	srcFile.Close()

	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("failed to update %s: %w", filePath, err)
	}
	setActiveTemp("")
	return nil
}

func main() {
	initSignalHandler()
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
)

func TestRunCLI(t *testing.T) {
	tempDir := t.TempDir()
	sampleFile := filepath.Join(tempDir, "sample.txt")
	if err := os.WriteFile(sampleFile, []byte("Hello World\n"), 0644); err != nil {
		t.Fatalf("failed to create sample file: %v", err)
	}

	t.Run("stdin to stdout", func(t *testing.T) {
		stdin := strings.NewReader("Attack at dawn!\n")
		var stdout, stderr bytes.Buffer

		code := run([]string{}, stdin, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr.String())
		}
		expected := "Nggnpx ng qnja!\n"
		if stdout.String() != expected {
			t.Errorf("stdout = %q, want %q", stdout.String(), expected)
		}
	})

	t.Run("file argument to stdout", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{sampleFile}, nil, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr.String())
		}
		expected := "Uryyb Jbeyq\n"
		if stdout.String() != expected {
			t.Errorf("stdout = %q, want %q", stdout.String(), expected)
		}
	})

	t.Run("output flag -o", func(t *testing.T) {
		outFile := filepath.Join(tempDir, "output.txt")
		var stdout, stderr bytes.Buffer
		code := run([]string{"-o", outFile, sampleFile}, nil, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr.String())
		}
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}
		expected := "Uryyb Jbeyq\n"
		if string(content) != expected {
			t.Errorf("output file content = %q, want %q", string(content), expected)
		}
	})

	t.Run("in-place flag -i", func(t *testing.T) {
		targetFile := filepath.Join(tempDir, "inplace.txt")
		if err := os.WriteFile(targetFile, []byte("Secret Message"), 0600); err != nil {
			t.Fatalf("failed to write target file: %v", err)
		}

		var stdout, stderr bytes.Buffer
		code := run([]string{"-i", targetFile}, nil, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr.String())
		}
		content, err := os.ReadFile(targetFile)
		if err != nil {
			t.Fatalf("failed to read target file: %v", err)
		}
		expected := "Frperg Zrffntr"
		if string(content) != expected {
			t.Errorf("inplace content = %q, want %q", string(content), expected)
		}

		info, err := os.Stat(targetFile)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
		}
	})

	t.Run("conflicting -o and -i flags", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-i", "-o", "out.txt", sampleFile}, nil, &stdout, &stderr)
		if code == 0 {
			t.Errorf("expected non-zero code for conflicting flags, got 0")
		}
		if !strings.Contains(stderr.String(), "cannot be used together") {
			t.Errorf("unexpected error message: %s", stderr.String())
		}
	})

	t.Run("in-place with stdin error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader("input")
		code := run([]string{"-i"}, stdin, &stdout, &stderr)
		if code == 0 {
			t.Errorf("expected non-zero code for -i without files, got 0")
		}
		if !strings.Contains(stderr.String(), "cannot be used with standard input") {
			t.Errorf("unexpected error message: %s", stderr.String())
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"/path/to/nonexistent/file.txt"}, nil, &stdout, &stderr)
		if code == 0 {
			t.Errorf("expected non-zero code for missing file, got 0")
		}
	})
}

func TestRunCLI_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()
	f1 := filepath.Join(tempDir, "file1.txt")
	f2 := filepath.Join(tempDir, "file2.txt")
	f3 := filepath.Join(tempDir, "file3.txt")

	if err := os.WriteFile(f1, []byte("Hello "), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("Beautiful "), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f3, []byte("World\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{f1, f2, f3}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr.String())
	}

	expected := "Uryyb Ornhgvshy Jbeyq\n"
	if stdout.String() != expected {
		t.Errorf("stdout = %q, want %q", stdout.String(), expected)
	}
}

func TestRunCLI_MultiFileErrorContinuation(t *testing.T) {
	tempDir := t.TempDir()
	missingFile := filepath.Join(tempDir, "missing.txt")
	validFile := filepath.Join(tempDir, "valid.txt")
	if err := os.WriteFile(validFile, []byte("Hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{missingFile, validFile}, nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 when one file fails, got %d", code)
	}

	expectedStdout := "Uryyb\n"
	if stdout.String() != expectedStdout {
		t.Errorf("stdout = %q, want %q", stdout.String(), expectedStdout)
	}

	if !strings.Contains(stderr.String(), "cannot open") {
		t.Errorf("stderr missing cannot open error: %s", stderr.String())
	}
}

func TestRunCLI_SymlinkInPlaceSafety(t *testing.T) {
	tempDir := t.TempDir()
	targetFile := filepath.Join(tempDir, "target.txt")
	symlinkFile := filepath.Join(tempDir, "link.txt")

	if err := os.WriteFile(targetFile, []byte("Secret Target"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetFile, symlinkFile); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-i", symlinkFile}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	// Symlink must still be a symlink
	linkInfo, err := os.Lstat(symlinkFile)
	if err != nil {
		t.Fatal(err)
	}
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink was replaced by regular file instead of updating target")
	}

	// Target content must be rotated
	targetBytes, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(targetBytes) != "Frperg Gnetrg" {
		t.Errorf("target content = %q, want %q", string(targetBytes), "Frperg Gnetrg")
	}
}

func TestRunCLI_MixedStdinAndFiles(t *testing.T) {
	tempDir := t.TempDir()
	f1 := filepath.Join(tempDir, "prefix.txt")
	f2 := filepath.Join(tempDir, "suffix.txt")

	if err := os.WriteFile(f1, []byte("Prefix: "), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte(" [End]\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdin := strings.NewReader("Streamed Input")
	var stdout, stderr bytes.Buffer

	code := run([]string{f1, "-", f2}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr.String())
	}

	expected := "Cersvk: Fgernzrq Vachg [Raq]\n"
	if stdout.String() != expected {
		t.Errorf("stdout = %q, want %q", stdout.String(), expected)
	}
}

func TestRunCLI_InPlacePermissionsPreservation(t *testing.T) {
	tempDir := t.TempDir()
	testModes := []os.FileMode{0600, 0644, 0755}

	for _, mode := range testModes {
		path := filepath.Join(tempDir, "file_"+mode.String()+".txt")
		content := "Sensitive Content 123\n"
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		code := run([]string{"-i", path}, nil, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("mode %v: expected exit 0, got %d: %s", mode, code, stderr.String())
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("mode %v: failed to stat: %v", mode, err)
		}
		if info.Mode().Perm() != mode {
			t.Errorf("mode %v: expected %o permissions, got %o", mode, mode, info.Mode().Perm())
		}

		readBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(readBytes) != "Frafvgvir Pbagrag 123\n" {
			t.Errorf("mode %v: unexpected rotated content: %s", mode, string(readBytes))
		}
	}
}

func TestRunCLI_InPlaceEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	emptyPath := filepath.Join(tempDir, "empty.txt")
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-i", emptyPath}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0 for empty file in-place, got %d: %s", code, stderr.String())
	}

	content, err := os.ReadFile(emptyPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(content))
	}
}

func TestRunCLI_InPlaceSubdirectoryRelativePath(t *testing.T) {
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "nested", "level2")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(nestedDir, "target.txt")
	if err := os.WriteFile(filePath, []byte("Hello Subdir"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-i", filePath}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "Uryyb Fhoqve" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

func TestRunCLI_DoubleDashOptionTerminator(t *testing.T) {
	tempDir := t.TempDir()
	dashFile := filepath.Join(tempDir, "-dashfile.txt")
	if err := os.WriteFile(dashFile, []byte("Data inside dash file\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--", dashFile}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0 when using -- option terminator, got %d: %s", code, stderr.String())
	}

	expected := "Qngn vafvqr qnfu svyr\n"
	if stdout.String() != expected {
		t.Errorf("stdout = %q, want %q", stdout.String(), expected)
	}
}

func TestRunCLI_HelpFlag(t *testing.T) {
	for _, flagArg := range []string{"-h", "--help"} {
		var stdout, stderr bytes.Buffer
		code := run([]string{flagArg}, nil, &stdout, &stderr)
		if code != 0 {
			t.Errorf("flag %s: expected code 0, got %d", flagArg, code)
		}
		if !strings.Contains(stderr.String(), "Usage: rot13") {
			t.Errorf("flag %s: expected usage in stderr, got: %s", flagArg, stderr.String())
		}
	}
}

func TestRunCLI_VersionFlag(t *testing.T) {
	for _, flagArg := range []string{"-v", "--version"} {
		var stdout, stderr bytes.Buffer
		code := run([]string{flagArg}, nil, &stdout, &stderr)
		if code != 0 {
			t.Errorf("flag %s: expected code 0, got %d", flagArg, code)
		}
		if !strings.Contains(stdout.String(), "rot13 version") {
			t.Errorf("flag %s: expected version in stdout, got: %s", flagArg, stdout.String())
		}
	}
}

func TestRunCLI_UnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--nonexistent-flag"}, nil, &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected non-zero exit for unknown flag, got 0")
	}
}

type brokenPipeWriter struct{}

func (b *brokenPipeWriter) Write(p []byte) (int, error) {
	return 0, syscall.EPIPE
}

func TestRunCLI_BrokenPipe(t *testing.T) {
	stdin := strings.NewReader("Stream that gets truncated by downstream tool\n")
	var stderr bytes.Buffer
	out := &brokenPipeWriter{}

	code := run([]string{}, stdin, out, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0 on broken pipe (SIGPIPE/EPIPE), got %d, stderr: %s", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("expected silent exit on broken pipe, got stderr: %s", stderr.String())
	}
}

func TestRunCLI_DirectoryError(t *testing.T) {
	tempDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run([]string{tempDir}, nil, &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected non-zero exit code when given directory, got 0")
	}
	if !strings.Contains(stderr.String(), "Is a directory") {
		t.Errorf("expected 'Is a directory' in stderr, got: %s", stderr.String())
	}
}

func TestRunCLI_UnwritableOutputFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-o", "/nonexistent_root_dir/never_create_this.txt"}, strings.NewReader("test"), &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected non-zero exit code when output path is unwritable, got 0")
	}
}

func TestRunCLI_SelfOverwriteProtection(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "data.txt")
	original := "Critical Secret Data\n"
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-o", filePath, filePath}, nil, &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected error code when output file equals input file, got 0")
	}
	if !strings.Contains(stderr.String(), "also an input file") {
		t.Errorf("expected 'also an input file' warning, got: %s", stderr.String())
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != original {
		t.Errorf("file was truncated/corrupted! got %q, want %q", string(content), original)
	}
}

func TestRunCLI_SymlinkSelfOverwriteProtection(t *testing.T) {
	tempDir := t.TempDir()
	realFile := filepath.Join(tempDir, "real.txt")
	linkFile := filepath.Join(tempDir, "link.txt")
	original := "Symlink Secret Data\n"
	if err := os.WriteFile(realFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realFile, linkFile); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-o", linkFile, realFile}, nil, &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected error code when output symlink resolves to input file, got 0")
	}
	if !strings.Contains(stderr.String(), "also an input file") {
		t.Errorf("expected 'also an input file' warning, got: %s", stderr.String())
	}

	content, err := os.ReadFile(realFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != original {
		t.Errorf("real file was truncated/corrupted! got %q, want %q", string(content), original)
	}
}

func TestRunCLI_OutputFilePreservedOnInputError(t *testing.T) {
	tempDir := t.TempDir()
	outFile := filepath.Join(tempDir, "important_output.txt")
	originalContent := "Do not erase me!\n"
	if err := os.WriteFile(outFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	nonExistentInput := filepath.Join(tempDir, "does_not_exist.txt")
	code := run([]string{"-o", outFile, nonExistentInput}, nil, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit code when input file does not exist")
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != originalContent {
		t.Fatalf("output file was truncated/corrupted on input error! got %q, want %q", string(content), originalContent)
	}
}

func TestRunCLI_OutputFilePreservedOnMultipleInputsWithOneError(t *testing.T) {
	tempDir := t.TempDir()
	outFile := filepath.Join(tempDir, "preserved.txt")
	validFile := filepath.Join(tempDir, "valid.txt")
	missingFile := filepath.Join(tempDir, "missing.txt")
	originalOut := "Original destination data\n"

	if err := os.WriteFile(outFile, []byte(originalOut), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validFile, []byte("Valid data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-o", outFile, validFile, missingFile}, nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != originalOut {
		t.Fatalf("output file overwritten despite partial failure! got %q, want %q", string(content), originalOut)
	}
}

func TestRunCLI_InPlaceDirectoryError(t *testing.T) {
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-i", nestedDir}, nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 when attempting in-place on directory, got %d", code)
	}
	if !strings.Contains(stderr.String(), "is a directory") {
		t.Errorf("expected 'is a directory' error, got: %s", stderr.String())
	}
}

func TestRunCLI_InPlaceNonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	missing := filepath.Join(tempDir, "not_there.txt")

	var stdout, stderr bytes.Buffer
	code := run([]string{"-i", missing}, nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 when in-place target does not exist, got %d", code)
	}
	if !strings.Contains(stderr.String(), "no such file or directory") {
		t.Errorf("expected file not found error, got: %s", stderr.String())
	}
}

func TestRunCLI_InPlaceSpecialFile(t *testing.T) {
	if _, err := os.Stat("/dev/null"); err != nil {
		t.Skip("skipping /dev/null test on this environment")
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-i", "/dev/null"}, nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 when attempting in-place on special device, got %d", code)
	}
	if !strings.Contains(stderr.String(), "not a regular file") {
		t.Errorf("expected 'not a regular file' error, got: %s", stderr.String())
	}
}

func TestRunCLI_InPlaceBrokenSymlink(t *testing.T) {
	tempDir := t.TempDir()
	brokenLink := filepath.Join(tempDir, "broken_link")
	if err := os.Symlink(filepath.Join(tempDir, "non_existent"), brokenLink); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-i", brokenLink}, nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 for broken symlink, got %d", code)
	}
	if !strings.Contains(stderr.String(), "no such file or directory") {
		t.Errorf("expected not found error for broken symlink target, got: %s", stderr.String())
	}
}

func TestRunCLI_OutputFlagUnwritableDirectory(t *testing.T) {
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("Data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	unwritableDir := filepath.Join(tempDir, "readonly_dir")
	if err := os.Mkdir(unwritableDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(unwritableDir, 0755)

	targetOut := filepath.Join(unwritableDir, "out.txt")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-o", targetOut, inputFile}, nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 for unwritable output directory, got %d", code)
	}
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestRunCLI_StreamReadError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	faulty := &errorReader{err: syscall.EIO}
	code := run([]string{}, faulty, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 on stream read error, got %d", code)
	}
	if !strings.Contains(stderr.String(), "stream error") {
		t.Errorf("expected 'stream error' in stderr, got: %s", stderr.String())
	}
}

func TestRunCLI_OptionTerminatorNonExistent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--", "-nonexistent_file.txt"}, nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 for missing dash-file, got %d", code)
	}
	if !strings.Contains(stderr.String(), "cannot open") {
		t.Errorf("expected cannot open error, got: %s", stderr.String())
	}
}

func TestRunCLI_ActiveTempConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(val string) {
			defer wg.Done()
			setActiveTemp(val)
		}(fmt.Sprintf("/tmp/test-%d", i))
		go func() {
			defer wg.Done()
			_ = getActiveTemp()
		}()
	}
	wg.Wait()
	setActiveTemp("")
}



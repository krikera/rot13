# rot13

`rot13` is a command-line tool to encode and decode text using the ROT13 cipher.

## Features

- Reads from files or standard input
- In-place file modification with `-i` (atomic replacement, preserves permissions)
- Target file output with `-o`
- Includes Debian package build scripts, man page, and shell completions (Bash, Zsh, Fish)

## Usage

```bash
# Read a file and print to stdout
rot13 secret.txt

# Read from stdin
echo "Hello World" | rot13
# Output: Uryyb Jbeyq

# Decode by running rot13 again (symmetric cipher)
echo "Uryyb Jbeyq" | rot13
# Output: Hello World

# Modify a file in-place
rot13 -i notes.txt

# Write output directly to a new file
rot13 -o encrypted.txt original.txt

# Help and options
rot13 --help
```

## Building and Installation

### From Source

```bash
make
sudo make install
```

To uninstall:

```bash
sudo make uninstall
```

### Debian Package (`.deb`)

```bash
make deb
sudo apt install ./dist/rot13_0.1.0_amd64.deb
```

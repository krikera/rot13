#compdef rot13

_rot13() {
    _arguments -s \
        '-o[Write output to specified file]:output file:_files' \
        '-i[Modify file(s) in-place]' \
        '(-v --version)'{-v,--version}'[Display version information]' \
        '(-h --help)'{-h,--help}'[Display help message]' \
        '*:input file:_files'
}

_rot13 "$@"

# bash completion for rot13

_rot13() {
    local cur prev words cword
    _init_completion || return

    local options="-o -i -v --version -h --help"

    case "${prev}" in
        -o)
            _filedir
            return 0
            ;;
    esac

    if [[ "${cur}" == -* ]]; then
        COMPREPLY=( $(compgen -W "${options}" -- "${cur}") )
        return 0
    fi

    _filedir
} &&
complete -F _rot13 rot13

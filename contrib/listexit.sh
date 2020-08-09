#!/usr/bin/bash

set -eu

get_exit() {
    while read -r line; do
        if [[ ! "$line" =~ EXIT:(.+) ]]; then
            continue
        fi
        echo "${BASH_REMATCH[1]}"
        return 0
    done < "$1"

    return 1
}

usage() {
    echo "usage: $(basename "$BASH_ARGV0") [-rih] [DIR]"
    echo ""
    echo "options:"
    echo " -r   Remove directories with EXIT != 0"
    echo " -i   Use rm's interactive mode, ask before delete"
    echo " -h   Show this page and exit"
}

main() {
    local remove=0
    local rmopts=("-r" "-f" "-v")

    while getopts "rih" arg; do
        case "$arg" in
            r) remove=1;;
            i) rmopts+=("-i");;
            h) usage && exit 0;;
            *) usage && exit 1;;
        esac
    done

    shift $((OPTIND-1))

    local dir="$PWD"
    if (( $# >= 1 )); then
        dir="$1"
    fi

    for dir in "$dir/run-"*; do
        local meta
        local code
        meta="$dir/META"
        if [[ ! -e "$meta" ]]; then
            continue
        fi

        code="$(get_exit "$meta")"

        if (( remove && code != 0 )); then
            rm "${rmopts[@]}" "$dir"
        else
            echo "exit code '$dir': $code"
        fi
    done
}

main "$@"

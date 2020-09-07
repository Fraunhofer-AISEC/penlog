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

fix_symlink() {
    local dir
    local runs
    dir="$1"

    mapfile -t runs < <(find "$dir" -maxdepth 1 -type d -name "run-*" | sort -r)
    if (( "${#runs}" > 0 )); then
        ln -sfnr "${runs[0]}" "LATEST"
    fi
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

    local base="$PWD"
    if (( $# >= 1 )); then
        base="$1"
    fi

    for dir in "$base/run-"*; do
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

    if (( remove )); then
        fix_symlink "$base"
    fi
}

main "$@"

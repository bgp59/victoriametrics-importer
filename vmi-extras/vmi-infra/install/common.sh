#! /bin/bash

# Common definitions and functions for all installers:

vmi_infra_root=${VMI_INFRA_ROOT:-$HOME/vmi-infra}

check_os_arch() {
    local _this_script=${this_script:-${BASH_SOURCE##*/}}
    os=$(uname -s | tr A-Z a-z)
    case "$os" in
        linux) : ;;
        *)
            echo >&2 "$_this_script - $os: unsupported OS"
            return 1
        ;;
    esac

    arch=$(uname -m | tr A-Z a-z)
    case "$arch" in
        amd64|arm64) : ;;
        x86_64) arch="amd64";;
        aarch64) arch="arm64";;
        *)
            echo >&2 "$_this_script - $arch: unsupported arch"
            return 1
        ;;
    esac
    
    return 0
}

make_runtime_dirs() {
    local _this_script=${this_script:-${BASH_SOURCE##*/}}
    local root_dir="$1"; shift
    if [[ -z "$root_dir" ]]; then
        echo >&2 "$_this_script - empty root dir"
        return 1
    fi
    local runtime_dir="$1"; shift

    if [[ -n "$runtime_dir" ]]; then
        (set -x; mkdir -p $runtime_dir) || return 1
        if [[ "$(realpath $root_dir)" == $(realpath $runtime_dir) ]]; then
            runtime_dir=
        fi
    fi

    if [[ -n "$runtime_dir" ]]; then
        (
            set -ex
            cd $root_dir
            for d in $*; do
                mkdir -p $runtime_dir/$d
                rm -rf $d
                ln -fs $runtime_dir/$d $d
            done
        )
    else
        (
            set -ex
            cd $root_dir
            mkdir -p $*
        )
    fi
    return $?
}
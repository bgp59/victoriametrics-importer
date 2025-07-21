#! /bin/bash --noprofile

# Sourced by various scripts.

check_os_arch() {
    local _this_dir=${this_dir:-$(cd $(dirname ${BASH_SOURCE}) && pwd)}
    local _this_script=${this_script:-${BASH_SOURCE##*/}}
    local os=$(uname -s | tr A-Z a-z)
    case "$os" in
        linux) : ;;
        *)
            echo >&2 "$_this_script - $os: unsupported OS"
            return 1
        ;;
    esac

    local arch=$(uname -m | tr A-Z a-z)
    case "$arch" in
        amd64|arm64) : ;;
        x86_64) arch="amd64";;
        aarch64) arch="arm64";;
        *)
            echo >&2 "$_this_script - $arch: unsupported arch"
            return 1
        ;;
    esac
    
    os_arch="$os-$arch"

    local b
    for b in $_this_dir/bin $_this_dir/bin/$os_arch; do
        if [[ -d "$b" && "$PATH" != "$b"* ]]; then
            export PATH="$b${PATH:+:}$PATH"
        fi
    done
    return 0
}

check_if_not_running() {
    local _this_script=${this_script:-${BASH_SOURCE##*/}}
    local _running=$(pgrep -af "(.*/)?$*( |\$)")
    if [[ -n "$_running" ]] >&2; then
        echo >&2 "$_this_script - $@ running"
        echo >&2 "$_running"
        return 1
    fi
    return 0
}

kill_wait_proc() {
    local pids=$(pgrep -f "(.*/)?$*( |\$)")
    local _this_script=${this_script:-${BASH_SOURCE##*/}}
    if [[ -z "$pids" ]]; then
        echo >&2 "$_this_script - $@ not running"
        return 0
    fi

    echo >&2 "$_this_script - Killing $@..."

    local _max_pid_wait=${max_pid_wait:-8}
    local _kill_sig_list=${kill_sig_list:-TERM KILL}
    local sig
    local k
    for sig in $_kill_sig_list; do
        (set -x; kill -$sig $pids) || return 1
        for ((k=0; k<$_max_pid_wait; k++)); do
            sleep 1
            ps -p $pids > /dev/null || return 0
        done
    done
    return 1
}

create_dir_maybe_symlink() {
    (
        set +ex
        while [[ $# -gt 0 ]]; do
            target=$(readlink $1)
            if [[ -z "$target" ]]; then
                target="$1"
            fi
            (set -x; mkdir -p $target)
            shift
        done
    )
}

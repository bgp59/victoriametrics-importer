#! /bin/bash

this_script=${0##*/}
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac

check_if_not_running() {
    local _running=$(pgrep -af "(^| |/)$*( |\$)")
    if [[ -n "$_running" ]] >&2; then
        echo >&2 "$this_script - $@ already running"
        echo >&2 "$_running"
        return 1
    fi
    return 0
}

kill_wait_proc() {
    local pids=$(pgrep -f "(^| |/)$*( |\$)")
    if [[ -z "$pids" ]]; then
        echo >&2 "$this_script - $@ not running"
        return 0
    fi

    echo >&2 "$this_script - Killing $@..."

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

case "$this_script" in
    start*)
        set -e
        check_if_not_running grafana
        root_dir=$(realpath $(dirname $this_dir))
        export PATH="$root_dir/bin${PATH:+:}$PATH"
        (
            set -x
            cd $root_dir
            mkdir -p out
            setsid grafana server --config conf/grafana.ini >out/grafana.out 2>out/grafana.err </dev/null &
        )
    ;;
    stop*)
        kill_wait_proc grafana
    ;;
esac

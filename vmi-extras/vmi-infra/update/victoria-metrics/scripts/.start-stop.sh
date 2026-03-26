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

make_dir_follow_link() {
    (
        set +ex
        for d in $*; do
            set +e
            to=$(readlink $d)
            if [[ "$to" != "" && ! -d "$to" ]]; then
                (set -ex; mkdir -p $to)
            elif [[ ! -d $d ]]; then
                (set -ex; mkdir -p $d)
            fi || return 1
        done
    )
}

case "$this_script" in
    start-victoria-metrics*)
        set -e
        check_if_not_running victoria-metrics
        root_dir=$(realpath $(dirname $this_dir))
        export PATH="$root_dir/bin${PATH:+:}$PATH"
        (
            set -x
            cd $root_dir
            make_dir_follow_link data out
            setsid victoria-metrics \
                -httpListenAddr=:8428,:18428 \
                -tls=false,true \
                -tlsCertFile=tls/cert.pem \
                -tlsKeyFile=tls/key.pem \
                -storageDataPath=data \
                -retentionPeriod=2d \
                -selfScrapeInterval=10s \
                > out/victoria-metrics.out 2>out/victoria-metrics.err < /dev/null &
        )
    ;;
    start-vmagent*)
        set -e
        check_if_not_running vmagent
        root_dir=$(realpath $(dirname $this_dir))
        export PATH="$root_dir/bin${PATH:+:}$PATH"
        (
            set -x
            cd $root_dir
            make_dir_follow_link data out
            setsid vmagent \
                -httpListenAddr=:8429,:18429 \
                -tls=false,true \
                -tlsCertFile=tls/cert.pem \
                -tlsKeyFile=tls/key.pem \
                -httpAuth.username=vmi \
                -httpAuth.password=file://./auth/password \
                -remoteWrite.url=http://localhost:8428/api/v1/write \
                -metrics.exposeMetadata \
                -promscrape.config=conf/promscrape.yaml \
                -remoteWrite.tmpDataPath=data/vmagent-remotewrite-data \
                > out/vmagent.out 2>out/vmagent.err < /dev/null &
        )
    ;;
    stop-victoria-metrics*)
        kill_wait_proc victoria-metrics
    ;;
    stop-vmagent*)
        kill_wait_proc vmagent
    ;;
esac

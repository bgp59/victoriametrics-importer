#! /bin/bash

this_script=${0##*/}
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac
. $this_dir/../common.sh

case "$this_script" in
    start*)
        set -e
        check_if_not_running victoria-metrics
        export PATH="$this_dir/bin${PATH:+:}$PATH"
        (
            set -x
            cd $this_dir
            create_dir_maybe_symlink out data
            setsid victoria-metrics \
                -storageDataPath data \
                -retentionPeriod 2d \
                -selfScrapeInterval=10s \
                > out/victoria-metrics.out 2>out/victoria-metrics.err < /dev/null &
        )
    ;;
    stop*)
        kill_wait_proc victoria-metrics
    ;;
esac

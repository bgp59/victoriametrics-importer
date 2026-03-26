#! /bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) this_dir=$(realpath $(dirname $0));;
    *) this_dir=$(realpath $(dirname $(which $0)));;
esac

root_dir=$(realpath $(dirname $this_dir))

case "$this_script" in
    start*)
        set -x
        cd $root_dir/victoria-metrics
        ./scripts/start-victoria-metrics
        ./scripts/start-vmagent
        cd $root_dir/grafana
        ./scripts/start-grafana
        ;;
    stop*)
        set -x
        cd $root_dir/grafana
        ./scripts/stop-grafana
        cd $root_dir/victoria-metrics
        ./scripts/stop-vmagent
        ./scripts/stop-victoria-metrics
        ;;
    run*)
        sleep_pid=
        trap '
        cd $this_dir
        ./stop-vmi-infra
        if [[ -n "$"sleep_pid ]]; then
            kill -KILL $sleep_pid
            sleep_pid=
        fi
        ' HUP INT TERM
        set -x
        cd $this_dir
        ./start-vmi-infra
        sleep infinity &
        sleep_pid="$!"
        wait
        ;;
esac

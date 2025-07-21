#! /bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac

case "$this_script" in
    start*)
        set -x
        cd $this_dir || exit 1
        ./victoria-metrics/start-victoria-metrics.sh
        ./grafana/start-grafana.sh
        ;;
    run*)
        sleep_pid=
        trap '
        ./stop-vmi-infra.sh
        set +x
        if [[ -n "$sleep_pid" ]]; then
            kill -KILL $sleep_pid
            sleep_pid=
        fi
        ' HUP INT TERM
        set -x
        cd $this_dir || exit 1
        ./victoria-metrics/start-victoria-metrics.sh
        ./grafana/start-grafana.sh
        sleep infinity &
        sleep_pid="$!"
        wait
        ;;
    stop*)
        set -x
        cd $this_dir || exit 1
        ./grafana/stop-grafana.sh
        ./victoria-metrics/stop-victoria-metrics.sh
        ;;
esac





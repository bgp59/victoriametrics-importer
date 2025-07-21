#!/bin/bash

this_script=${0##*/}

# Common functions, etc:
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac

if [[ -n "$HOME" && -d "$HOME" ]]; then
    vmi_infra_root_dir="$HOME/vmi-infra"
else
    vmi_infra_root_dir="/tmp/${USER:-$UID}/vmi-infra"
fi

vmi_runtime_dir=

usage="
Usage: $this_script [-r ROOT_DIR] [-R RUNTIME_DIR]

Install VictoriaMetrics & Grafana under ROOT_DIR, default: $vmi_infra_root_dir,
using RUNTIME_DIR as runtime dir, default: ROOT_DIR/runtime.
"

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h*|--h*)
            echo >&2 "$usage"
            exit 1
            ;;
        -r*|--root*)
            shift
            vmi_infra_root_dir="$1"
            ;;
        -R|--runtime*)
            shift
            vmi_runtime_dir="$1"
            ;;
    esac
    shift
done

if [[ -z "$vmi_infra_root_dir" ]]; then
    echo >&2 "$usage"
    exit 1
fi

set -ex
mkdir -p $vmi_infra_root_dir
vmi_infra_root_dir=$(realpath $vmi_infra_root_dir)

cd $this_dir/files
rsync -plrSH base/ $vmi_infra_root_dir
rsync -plrSH update/ $vmi_infra_root_dir

cd $vmi_infra_root_dir

./victoria-metrics/download-victoria-metrics.sh
./victoria-metrics/create-victoria-metrics-runtime-symlinks.sh ${vmi_runtime_dir:-../runtime}/victoria-metrics

./grafana/download-grafana.sh
./grafana/create-grafana-runtime-symlinks.sh ${vmi_runtime_dir:-../runtime}/grafana



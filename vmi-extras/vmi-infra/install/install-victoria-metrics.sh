#! /bin/bash --noprofile

# VictoriaMetrics version:
vm_ver=1.138.0

this_script=${0##*/}

usage="
Usage: $this_script [-b] [-r ROOT_DIR] [-R RUNTIME_DIR]

Install VictoriaMetrics $vm_ver under ROOT_DIR using RUNTIME_DIR for 
data/ runtime dir.

Default ROOT_DIR is \$VMI_INFRA_ROOT/victoria-metrics.

Default VMI_INFRA_ROOT is \$HOME/vmi-infra.

Default RUNTIME_DIR is \$VMI_INFRA_RUNTIME/victoria-metrics, 
if VMI_INFRA_RUNTIME is defined, otherwise it will 
use ROOT_DIR.
"

set -e

case "$0" in
    /*|*/*) this_dir=$(realpath $(dirname $0));;
    *) this_dir=$(realpath $(dirname $(which $0)));;
esac

. $this_dir/common.sh
base_only=
root_dir=$vmi_infra_root/victoria-metrics
if [[ -n "$VMI_INFRA_RUNTIME" ]]; then
    runtime_dir=$VMI_INFRA_RUNTIME/victoria-metrics
else
    runtime_dir=
fi
while [[ $# -gt 0 ]]; do
    case "$1" in
        -h*|--h*)
            echo >&2 "$usage"
            exit 1
            ;;
        -b|--base*)
            base_only=1
            ;;
        -r|--root*)
            shift
            root_dir=$1
            ;;
        -R|--runtime*)
            shift
            runtime_dir=$1
            ;;
    esac
    shift
done

check_os_arch

download_subdir=victoria-metrics-$vm_ver
download_url=https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v$vm_ver
(
    set -x
    mkdir -p $root_dir
    cd $root_dir
    mkdir -p $download_subdir
    curl -s -L $download_url/victoria-metrics-$os-$arch-v$vm_ver.tar.gz | tar -xzf - -C $download_subdir
    curl -s -L $download_url/vmutils-$os-$arch-v$vm_ver.tar.gz | tar -xzf - -C $download_subdir vmagent-prod 
    mkdir -p bin
    for f in victoria-metrics vmagent; do
        ln -fs ../$download_subdir/$f-prod bin/$f
    done
)

make_runtime_dirs "$root_dir" "$runtime_dir" data out

if [[ -z "$base_only" ]]; then
    update_dir=$(realpath $this_dir/../update/victoria-metrics)
    (
        set -x
        rsync -plrtHS $update_dir/ $root_dir
    )
fi

#! /bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) this_dir=$(realpath $(dirname $0));;
    *) this_dir=$(realpath $(dirname $(which $0)));;
esac

. $this_dir/common.sh

usage="
Usage: $this_script [-b] [-r ROOT_DIR] [-R RUNTIME_DIR]

Install VictoriaMetrics and Grafana under ROOT_DIR using RUNTIME_DIR 
for runtime dir.

Default ROOT_DIR is \$VMI_INFRA_ROOT or \$HOME/vmi-infra if the latter is 
not set.

Default RUNTIME_DIR is \$VMI_INFRA_RUNTIME if VMI_INFRA_RUNTIME is defined, 
otherwise it will use ROOT_DIR.
"

base_only_arg=
root_dir=$vmi_infra_root
if [[ -n "$VMI_INFRA_RUNTIME" ]]; then
    runtime_dir=$VMI_INFRA_RUNTIME
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
            base_only_arg=$1
            ;;
        -r|--root*)
            shift
            root_dir="$1"
            ;;
        -R|--runtime*)
            shift
            runtime_dir="$1"
            ;;
    esac
    shift
done

set -e

check_os_arch

(
    set -x
    cd $this_dir
    ./install-victoria-metrics.sh \
        -r $root_dir/victoria-metrics \
        ${runtime_dir:+-R $runtime_dir/victoria-metrics} \
        ${base_only_arg}
    ./install-grafana.sh \
        -r $root_dir/grafana \
        ${runtime_dir:+-R $runtime_dir/grafana} \
        ${base_only_arg}
)

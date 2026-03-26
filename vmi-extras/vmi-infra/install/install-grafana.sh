#! /bin/bash --noprofile

# Grafana version:
grafana_ver=12.4.0_22325204712
grafana_semver=${grafana_ver%_*}

this_script=${0##*/}

usage="
Usage: $this_script [-b] [-r ROOT_DIR] [-R RUNTIME_DIR]

Install Grafana $grafana_semver under ROOT_DIR using RUNTIME_DIR for 
data/ runtime dir.

If -b is provided, install only the base part: downloaded package
and basic dir structure. This option is for container build where
the update-able part is left as a separate layer.

Default ROOT_DIR is \$VMI_INFRA_ROOT/grafana.

Default VMI_INFRA_ROOT is \$HOME/vmi-infra.

Default RUNTIME_DIR is \$VMI_INFRA_RUNTIME/grafana, 
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
root_dir=$vmi_infra_root/grafana
if [[ -n "$VMI_INFRA_RUNTIME" ]]; then
    runtime_dir=$VMI_INFRA_RUNTIME/grafana
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

if [[ "$grafana_ver" == *_* ]]; then
    download_url=https://dl.grafana.com/grafana/release/${grafana_semver}/grafana_${grafana_ver}_${os}_${arch}.tar.gz
else
    download_url=https://dl.grafana.com/oss/release/grafana-$grafana_ver.$os-$arch.tar.gz
fi
download_subdir=grafana-$grafana_semver

(
    set -x
    mkdir -p $root_dir
    cd $root_dir
    curl -s -L $download_url | tar xzf -
    if [[ -d grafana-v$grafana_semver ]]; then
        rm -rf grafana-$grafana_semver
        mv grafana-v$grafana_semver grafana-$grafana_semver
    fi
    ln -fs $download_subdir/bin .
    ln -fs $download_subdir/public .
    mkdir -p conf
    ln -fs ../$download_subdir/conf/defaults.ini conf
)


make_runtime_dirs "$root_dir" "$runtime_dir" data out log

if [[ -z "$base_only" ]]; then
    update_dir=$(realpath $this_dir/../update/grafana)
    (
        set -x
        rsync -plrtHS $update_dir/ $root_dir
    )
fi

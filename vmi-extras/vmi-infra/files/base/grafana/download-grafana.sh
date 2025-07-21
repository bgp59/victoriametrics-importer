#!/bin/bash

this_script=${0##*/}

# Grafana version:
grafana_ver=9.4.7

# Common functions, etc:
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac
. $this_dir/../common.sh

set -e
check_os_arch

set -x
cd $this_dir
if [[ ! -d bin ]]; then
    curl -s -L https://dl.grafana.com/oss/release/grafana-$grafana_ver.$os_arch.tar.gz | tar xzf -
    ln -fs grafana-$grafana_ver/bin .
    ln -fs grafana-$grafana_ver/public .
fi

#!/bin/bash

this_script=${0##*/}

# VictoriaMetrics version:
vm_ver=1.90.0

# Common functions, etc:
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac
. $this_dir/../common.sh


set -e
check_os_arch
vm_install_subdir=victoria-metrics-$os_arch-$vm_ver

set -x
cd $this_dir
mkdir -p $vm_install_subdir
if [[ ! -x $vm_install_subdir/victoria-metrics-prod ]]; then
    curl -s -L https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v$vm_ver/victoria-metrics-$os_arch-v$vm_ver.tar.gz | tar -xzf - -C $vm_install_subdir 
fi
if [[ ! -x $vm_install_subdir/vmagent-prod ]]; then
    curl -s -L https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v$vm_ver/vmutils-$os_arch-v$vm_ver.tar.gz | tar -xzf - -C $vm_install_subdir vmagent-prod 
fi
mkdir -p bin
for f in victoria-metrics vmagent; do
    ln -fs ../$vm_install_subdir/$f-prod bin/$f
done

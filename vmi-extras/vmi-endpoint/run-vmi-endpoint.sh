#!/bin/bash --noprofile

# Run vmi-endpoint in run/pause loop:

this_script=${0##*/}

usage="
Usage: $this_script RUN PAUSE [ARG...]
Run vmi-endpoint ARG... in a loop, RUN sec active, PAUSE down

For vmi-endpoint invoke $this_script -H | --vmi-endpoint-help

"

case "$1" in
    ""|-h|--h*) echo >&2 "$usage"; exit 1;;
    -H|--vmi-endpoint-help) run=0; pause=0; shift;;
    *) run="$1"; shift;  pause="$1"; shift;;
esac

case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0) && pwd));;
esac

if [[ -z "$this_dir" ]]; then
    echo >&2 "Cannot infer dir for $0"
    exit 1
fi

go_os=$(uname -s | tr '[:upper:]' '[:lower:]')
go_arch=$(uname -m)
case "$go_arch" in
    x86_64) go_arch=amd64 ;;
    aarch64|arm64*) go_arch=arm64 ;;
    arm*) go_arch=arm ;;
esac

export PATH="$this_dir/bin/$go_os-$go_arch:$PATH"

if [[ "$run" == 0 ]]; then
    vmi-endpoint --help
    exit 0
fi

cleanup() {
    (set -x; pkill -P $$ vmi-endpoint)
    exit 1
}

trap cleanup 1 2 3 15

while [[ true ]]; do
    (set -x; exec vmi-endpoint $*) &
    sleep $run
    (set -x; pkill -P $$ vmi-endpoint; sleep $pause)
done

#! /bin/bash --noprofile

case "$0" in
    /*|*/*) 
        this_dir=$(cd $(dirname $0) && pwd)
    ;;
    *) 
        this_dir=$(cd $(dirname $(which $0)) && pwd)
    ;;
esac

go_os=$(uname -s | tr '[:upper:]' '[:lower:]')
go_arch=$(uname -m)
case "$go_arch" in
    x86_64) go_arch=amd64 ;;
    aarch64|arm64*) go_arch=arm64 ;;
    arm*) go_arch=arm ;;
esac

export PATH="$this_dir/bin/$go_os-$go_arch:$PATH"
exec refvmi "$@"

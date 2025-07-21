this_script=${0##*/}

# Common functions, etc:
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac

runtime=${1:-/volumes/runtime/grafana}

set -ex
cd $this_dir
for d in data log out; do
    ln -fs $runtime/$d .
done

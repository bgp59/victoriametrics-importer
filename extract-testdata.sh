#! /bin/bash --noprofile

# Prepare testdata/:

case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0)) && pwd);;
esac

root_dir=$this_dir

set -ex
cd $root_dir
tar xzf testdata.tgz

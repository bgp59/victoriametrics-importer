#!/bin/bash

this_script=${0##*/}
    
usage="
Usage: $this_script [-f|--force]

Apply extras SEMVER tag locally and to the remote. Requires
a clean git status. Use --force to reapply the tag.

"

force=
case "$1" in
    -h|--h*)
        echo >&2 "$usage"
        exit 1
        ;;
    -f|--force)
        force="--force"
        shift
        ;;
esac

case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0)) && pwd);;
esac
if [[ -z "$this_dir" ]]; then
    echo >&2 "$this_script: cannot determine script's directory"
    exit 1
fi
project_root_dir=$this_dir
export PATH="$project_root_dir/vmi-extras/devtools${PATH+:}${PATH}"

set -e
# Must have semver:
semver=$(cat $project_root_dir/vmi-extras/semver.txt)
if [[ -z "$semver" ]]; then
    echo >&2 "$this_script: missing mandatory $semver"
    exit 1
fi

set -x
exec git-tag.sh $force vmi-extras-$semver

 


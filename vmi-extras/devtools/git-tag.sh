#!/bin/bash

this_script=${0##*/}
    
usage="
Usage: $this_script [-f|--force] TAG

Apply tag locally and to the remote. Requires
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
tag="$1"; shift
if [[ -z "$tag" ]]; then
    echo >&2 "$this_script: missing mandatory tag argument"
    echo >&2 "$usage"
    exit 1
fi

# Add this script's directory to PATH:
this_dir=$(dirname $(realpath $0))
export PATH="$this_dir${PATH+:}${PATH}"

# Must be in in proper git state:
if ! check-git-state.sh; then
    echo >&2 "$this_script: cannot continue"
    exit 1
fi

git tag $force $tag
git push $force origin tag $tag 

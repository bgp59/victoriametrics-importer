#!/bin/bash

this_script=${0##*/}
    
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

# Must be in clean state and properly tagged:
if [[ -z "$__vmi_skip_git_state_check" ]]; then
    check-git-state.sh vmi-extras-$semver
fi

# Finally create the release:
cd $this_dir
mkdir -p releases
archive=releases/vmi-extras-$semver.tar.gz
tar czf $archive --exclude-from=.gitignore vmi-extras
echo "Created release:  $archive"
 


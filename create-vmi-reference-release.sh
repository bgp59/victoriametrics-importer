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
semver=$(cat $project_root_dir/reference/semver.txt)
if [[ -z "$semver" ]]; then
    echo >&2 "$this_script: missing mandatory $semver"
    exit 1
fi

# Must be in clean state and properly tagged:
if [[ -z "$__vmi_skip_git_state_check" ]]; then
    check-git-state.sh vmi-reference-$semver
fi

# Finally create the release:
cd $this_dir

staging_dir=local/staging
release_subdir=vmi-reference
mkdir -p $staging_dir
rm -rf $staging_dir/$release_subdir
(
    cd reference
    ./pre-build
)
rsync -plrtHS \
    --exclude-from=.gitignore \
    --exclude="go-build*" \
    --exclude="pre-build*" \
    --exclude=go.mod --exclude=go.sum \
    reference/ $staging_dir/$release_subdir
cp -p reference/go-build.sh reference/buildinfo.go $staging_dir/$release_subdir
awk '$1 == "module"{print}' reference/go.mod > $staging_dir/$release_subdir/go.mod
cat <<EOM > $staging_dir/$release_subdir/pre-build
go mod tidy
EOM
chmod +x $staging_dir/$release_subdir/pre-build


mkdir -p releases
archive=releases/vmi-reference-$semver.tar.gz
tar czf $archive -C $staging_dir $release_subdir
echo "Created release:  $archive"
 


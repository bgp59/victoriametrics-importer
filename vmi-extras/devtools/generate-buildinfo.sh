#! /bin/bash --noprofile

# Invoked before go build, with the current directory already set to the root of
# the project.

semver_file=semver.txt

if [[ -z "$semver" && -r "$semver_file" ]]; then
    semver=$(cat $semver_file)
    case "$semver" in
        [0-9]*) semver="v$semver";;
    esac
fi

set -e
echo '
// This file was automatically generated at build time. 
// Note that it is excluded from git control.

package main

var GitInfo = "'$(
    if [[ -n "$(git status --porcelain)" ]]; then
        dirty="-dirty"
    else
        dirty=""
    fi
    echo $(git rev-parse HEAD)${dirty}
)'"
var Version = "'$semver'"
' > buildinfo.go


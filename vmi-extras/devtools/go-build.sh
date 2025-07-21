#! /bin/bash --noprofile

# Conveneince script for GO builds.

this_script=${0##*/}

go_os_arch_targets_file="go-os-arch.targets"
pre_build_script="pre-build"
post_build_script="post-build"
semver_file="semver.txt"

usage="
Usage: $this_script [TARGET_ROOT]

This script should be symlinked with a file, typically called build, under the same
directory where the build is expected to occur and the latter should be invoked.

The script will make that location the current directory before proceeding further.

If TARGET_ROOT arg is not provided then the last component from go list output is used.

The script will look for an optional '$semver_file' file under the main.go's
location and it will use its content as semantic version, SEMVER.

The script will look for a '$go_os_arch_targets_file' file under the main.go's 
location. The file is expected to contain GOOS GOARCH ... specifications, 
one per line; each pair GOOS GOARCH will generate an executable. If no such file 
is present then a single executable will be built for the native GOOS and GOARCH.

For each GOOS and GOARCH the script will generate:

    bin/TARGET_ROOT-GOOS-GOARCH[-SEMVER]
    bin/GOOS-GOARCH/TARGET_ROOT -> ../TARGET_ROOT-GOOS-GOARCH[-SEMVER]

The script will also look for '$pre_build_script' and '$post_build_script'
files under the main.go's location and it will invoke them, if found, before
and after go build accordingly. The scripts will be invoked with the following
env var set: goos, goarch, out_file, semver, target_root
"

target_root=
case "$1" in
    -h|--help)
        echo >&2 "$usage"
        exit 
    ;;
    *)
        target_root="$1"
    ;;
esac

case "$0" in
    /*|*/*) 
        script_dir=$(cd $(dirname $0) && pwd)
        real_script_dir=$(dirname $(realpath $0))
    ;;
    *) 
        script_dir=$(cd $(dirname $(which $0)) && pwd)
        real_script_dir=$(dirname $(realpath $(which $0)))
    ;;
esac

if [[ -z "$script_dir" ]]; then
    echo >&2 "$this_script: cannot infer location from invocation"
    exit 1
fi

export PATH="$real_script_dir${PATH:+:}$PATH"

set -e
cd $script_dir

if [[ -z "$target_root" ]]; then
    target_root=$(go list | awk '(NR == 1){gsub(".*/", "", $1); print $1}')
fi
if [[ -z "$target_root" ]]; then
    echo >&2 "$this_script: cannot infer target_root"
    exit 1
fi

semver=
if [[ -r $semver_file ]]; then
    semver=$(cat $semver_file)
    case "$semver" in
        [0-9]*) semver="v$semver";;
    esac
fi

echo >&2 "$this_script: target_root='$target_root', semver='$semver'"

do_build() {
    (
        set -e
        goos=$(go env GOOS)
        goarch=$(go env GOARCH)
        out_file=$target_root-$goos-$goarch${semver:+-}$semver
        export goos goarch out_file semver target_root
        if [[ -x "$pre_build_script" ]]; then
            (set -x; $(dirname $pre_build_script)/$pre_build_script)
        fi
        (
            set -x
            mkdir -p bin
            go build -o bin/$out_file
            mkdir -p bin/$goos-$goarch
            ln -fs ../$out_file bin/$goos-$goarch/$target_root
        )
        if [[ -x "$post_build_script" ]]; then
            (set -x; $(dirname $post_build_script)/$post_build_script)
        fi
    )
}

if [[ -r $go_os_arch_targets_file ]]; then
    export GOOS GOARCH
    while read GOOS GOARCH _; do
        [[ "$GOOS" = '#'* || -z "$GOOS" || -z "$GOARCH" ]] && continue
        do_build
    done <  $go_os_arch_targets_file
else
    do_build
fi

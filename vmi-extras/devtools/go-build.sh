#! /bin/bash --noprofile

# Convenience script for GO builds.

this_script=${0##*/}

pre_build_script="./pre-build"
post_build_script="./post-build"
semver_file="semver.txt"

usage="
Usage: $this_script [-t OS_ARCH_TARGET_FILE] [EXEC_ROOT]

This script should be symlinked with a file, typically called go-build, under the same
directory where the build is expected to occur and the latter should be invoked.

The script will make that location the current directory before proceeding further.

If EXEC_ROOT arg is not provided then the last component from go list output is used.

The script will look for an optional '$semver_file' file under the main.go's
location and it will use its content as semantic version, SEMVER.

If -t OS_ARCH_TARGET_FILE is specified then that file is expected to contain
GOOS GOARCH ... specifications, one per line; each pair GOOS GOARCH will generate 
an executable. If no such file is provide then a single executable will be built 
for the native GOOS and GOARCH.

For each GOOS and GOARCH the script will generate:

    bin/EXEC_ROOT-GOOS-GOARCH[-SEMVER]
    bin/GOOS-GOARCH/EXEC_ROOT -> ../EXEC_ROOT-GOOS-GOARCH[-SEMVER]

The script will also look for '$pre_build_script' and '$post_build_script'
files under the main.go's location and it will invoke them, if found, before
and after go build accordingly. The scripts will be invoked with the following
env var set: goos, goarch, out_file, semver, exec_root
"

exec_root=
go_os_arch_targets_file=
while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            echo >&2 "$usage"
            exit 
        ;;
        -t|--target*)
            shift
            go_os_arch_targets_file=$(realpath $1)
        ;;
        *)
            exec_root="$1"
            break
        ;;
    esac
    shift
done

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

cd $script_dir || exit 1

if [[ -z "$exec_root" ]]; then
    exec_root=$(go list | awk '(NR == 1){gsub(".*/", "", $1); print $1}')
fi
if [[ -z "$exec_root" ]]; then
    echo >&2 "$this_script: cannot infer exec_root"
    exit 1
fi

semver=
if [[ -r $semver_file ]]; then
    semver=$(cat $semver_file)
fi

echo >&2 "$this_script: exec_root='$exec_root', semver='$semver'"

do_build() {
    (
        goos=$(go env GOOS)
        goarch=$(go env GOARCH)
        out_file=$exec_root-$goos-$goarch${semver:+-}$semver
        export goos goarch out_file semver exec_root
        if [[ -x "$pre_build_script" ]]; then
            (set -x; $pre_build_script)
        fi
        real_bin=$(readlink bin)
        if [[ -z "$real_bin" ]]; then
            real_bin="bin"
        fi
        (
            set -ex
            mkdir -p $real_bin
            go build -o bin/$out_file
            mkdir -p bin/$goos-$goarch
            ln -fs ../$out_file bin/$goos-$goarch/$exec_root
        )
        if [[ -x "$post_build_script" ]]; then
            (set -x; $post_build_script)
        fi
    )
}

if [[ -n "$go_os_arch_targets_file" ]]; then
    export GOOS GOARCH
    while read GOOS GOARCH _; do
        [[ "$GOOS" = '#'* || -z "$GOOS" || -z "$GOARCH" ]] && continue
        do_build
    done <  $go_os_arch_targets_file
else
    do_build
fi

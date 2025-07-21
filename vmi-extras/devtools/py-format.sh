#! /bin/bash

# Apply Python code formatting tools:

this_script=${0##*/}

case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0)) && pwd);;
esac

project_root_dir=$(realpath $this_dir/../project-root)

case "$1" in
    -h|--h*)
        echo >&2 "Usage: $this_script DIR ..."
        exit 1
    ;;
esac

for d in ${@:-.}; do
    real_d=$(realpath $d) || continue
    if [[ "$real_d" != "$project_root_dir" && "$real_d" != "$project_root_dir/"* ]]; then
        echo >&2 "$this_script: '$d' ignored, its real path '$real_d' is outside '$project_root_dir', the project root"
        continue
    fi
    (
        set -x
        autoflake -v --config $this_dir/pyproject.toml $d
        isort --settings-path $this_dir $d
        black --config=$this_dir/pyproject.toml $d
    )
done

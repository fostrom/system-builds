#!/usr/bin/env bash
# Sets each line of a fragment in a busybox .config, replacing what is there:
# `merge-config.sh <.config> <fragment>`.
set -eu

config=$1
fragment=$2

while IFS= read -r line; do
    case $line in
        CONFIG_*=*)
            sym=${line%%=*}
            ;;
        "# CONFIG_"*" is not set")
            sym=${line#\# }
            sym=${sym% is not set}
            ;;
        *)
            continue
            ;;
    esac

    sed -i -e "/^${sym}=/d" -e "/^# ${sym} is not set\$/d" "$config"
    printf '%s\n' "$line" >> "$config"
done < "$fragment"

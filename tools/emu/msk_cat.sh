#!/bin/bash
# monsterskill (worker host): print lines FROM..TO of a read-only source file under ~/d2d-lab.  usage: msk_cat.sh FILE FROM TO
case $1 in "$HOME"/d2d-lab/*) ;; *) echo "path must be under ~/d2d-lab" >&2; exit 2 ;; esac
exec sed -n "${2:-1},${3:-200}p" "$1"

#!/usr/bin/env bash
# vim: ai:ts=8:sw=8:noet

set -eufo pipefail
export SHELLOPTS
IFS=$'\t\n'
umask 0077

(cd acceptance/ && go test -v ./...)

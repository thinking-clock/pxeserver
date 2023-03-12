#!/bin/bash
set -eo pipefail
/usr/sbin/pcscd

exec "${@}"

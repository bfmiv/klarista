#!/bin/bash

set -euo pipefail

CONTAINER_ID=$(basename $(cat /proc/1/cpuset))

echo $'
set -euo pipefail

INSTALL_DIR='${1:-"/usr/local/bin"}'
OS=$(echo $OSTYPE | grep -oE "^[[:alpha:]]+")
SUCCESS=

function on_exit {
	if ! [[ $SUCCESS ]]; then
		(docker rm -f '$CONTAINER_ID' > /dev/null) &
		kill -9 $$
	fi
	docker stop '$CONTAINER_ID' > /dev/null
}

trap on_exit EXIT

mkdir -p $INSTALL_DIR

docker cp '$CONTAINER_ID':/klarista-$OS-amd64 $INSTALL_DIR/klarista

echo "Installed klarista to $INSTALL_DIR"

$INSTALL_DIR/klarista --version

SUCCESS=1

exit'

DONE=

trap 'DONE=1' SIGTERM

while true; do
	[[ $DONE ]] && break
	sleep 0.1
done

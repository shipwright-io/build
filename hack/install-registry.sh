#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Spin up a simple container registry in docker, and using "show" parameter return its internal IP
# address.
#
#   $ ./install-registry
#   $ ./install-registry show
#

set -eu

ACTION="${1:-}"

if [[ ! -z "${ACTION}" ]] && [[ "${ACTION}" != "show" ]] ; then
    echo "[ERROR] Invalid action '${ACTION}'!" 1>&2
    exit 1
fi

# contrainer registry name
REGISTRY_NAME="${REGISTRY_NAME:-kind-registry}"
# container registry port
REGISTRY_PORT="${REGISTRY_PORT:-5000}"

function is_registry_running () {
    REGISTRY_RUNNING="$(docker inspect --format='{{ .State.Running }}' "${REGISTRY_NAME}" 2>/dev/null || true)"
    if [ "${REGISTRY_RUNNING}" != "true" ] ; then
        return 1
    else
        return 0
    fi
}

function start_registry () {
    docker run \
        --name="${REGISTRY_NAME}" \
        --publish="${REGISTRY_PORT}:5000" \
        --restart="always" \
        --detach \
        --net=kind \
        registry:2

    cat << EOS
# Container registry running, to stop it run:
#   $ docker container stop ${REGISTRY_NAME}
#   $ docker container rm --volumes ${REGISTRY_NAME}
EOS
}

function show_ipaddr () {
    docker inspect --format='{{ .NetworkSettings.Networks.kind.IPAddress }}' "${REGISTRY_NAME}"
}

if [ "${ACTION}" == "" ] ; then
    if ! is_registry_running ; then
        start_registry
    else
        echo "# Registry is already running!"
    fi
else
    REGISTRY_IP="$(show_ipaddr)"
    if [ -z "${REGISTRY_IP}" ]; then
        echo "[ERROR] Container registry is not running, not able to obtain address!" 1>&2
        exit 1
    fi
    echo $REGISTRY_IP
fi

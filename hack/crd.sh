#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Manages the deployment and removal of CRDs in a cluster. Usage:
#
#   $ crd.sh uninstall
#   $ crd.sh install
#

ACTION="${1}"
CRDS=(
    # operator components
    deploy/namespace.yaml
    deploy/role.yaml
    deploy/role_binding.yaml
    deploy/service_account.yaml
    deploy/operator.yaml
    deploy/crds/build.dev_buildstrategies_crd.yaml
    deploy/crds/build.dev_clusterbuildstrategies_crd.yaml
    deploy/crds/build.dev_builds_crd.yaml
    deploy/crds/build.dev_buildruns_crd.yaml
    # cluster scope build strategies
    samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml
    samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml
    samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml
    samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml
    samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml
    samples/buildstrategy/source-to-image/buildstrategy_source-to-image-redhat_cr.yaml
)

function die () {
    echo "[ERROR] ${@}" >&2
    exit 1
}

if [[ "${ACTION}" != "install" ]] && [[ "${ACTION}" != "uninstall" ]] ; then
    die "Invalid argument, it should be either 'install' or 'uninstall'"
fi

# apply resource file, and on error stop executing.
function kubectl_apply() {
    kubectl apply -f "${*}" || \
        die "Unable to install '${*}'"
}

# delete resource file, and on error print out warning.
function kubectl_delete() {
    kubectl delete -f "${*}" || \
        echo "[WARN] Unable to delete resource '${*}'"
}

for crd in ${CRDS[@]}; do
    if [[ ! -f $crd ]] ; then
        die "File not found at: '${crd}'"
    fi

    if [[ $ACTION == "install" ]] ; then
        kubectl_apply $crd
    fi
    if [[ $ACTION == "uninstall" ]] ; then
        kubectl_delete $crd
    fi
done

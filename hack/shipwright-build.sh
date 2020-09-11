#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

# Manages the deployment and removal of Shipwright Builds in a cluster. Usage:
#
#   $ shipwright-build.sh uninstall [component]
#   $ shipwright-build.sh install [component]
#
# If no component is specified, all components are installed/uninstalled.
# The following can be installed individually:
# 
# - apis: the Shipwright Build CRDs
# - controllers: the Shipwright Build controllers
# - strategies: the sample build strategies

ACTION="${1}"
COMPONENT="${2:-all}"
APIS=(
    deploy/crds/build.dev_buildstrategies_crd.yaml
    deploy/crds/build.dev_clusterbuildstrategies_crd.yaml
    deploy/crds/build.dev_builds_crd.yaml
    deploy/crds/build.dev_buildruns_crd.yaml
)
CONTROLLERS=(
    # controller components
    deploy/namespace.yaml
    deploy/role.yaml
    deploy/role_binding.yaml
    deploy/service_account.yaml
    deploy/operator.yaml
)
STRATEGIES=(
    # cluster scope build strategies
    samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml
    samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml
    samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml
    samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml
    samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml
    samples/buildstrategy/source-to-image/buildstrategy_source-to-image-redhat_cr.yaml
)

function die () {
    echo "[ERROR] ${*}" >&2
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

if [[ $COMPONENT == "apis" || $COMPONENT == "all" ]]; then
    echo "${ACTION}ing apis"
    for crd in "${APIS[@]}"; do
        if [[ ! -f $crd ]] ; then
            die "File not found at: '${crd}'"
        fi

        if [[ $ACTION == "install" ]] ; then
            kubectl_apply "$crd"
        fi
        if [[ $ACTION == "uninstall" ]] ; then
            kubectl_delete "$crd"
        fi
    done
fi

if [[ $COMPONENT == "controllers" || $COMPONENT == "all" ]]; then
    echo "${ACTION}ing controllers"
    for resource in "${CONTROLLERS[@]}"; do
        if [[ ! -f $resource ]] ; then
            die "File not found at: '${crd}'"
        fi

        if [[ $ACTION == "install" ]] ; then
            kubectl_apply "$resource"
        fi
        if [[ $ACTION == "uninstall" ]] ; then
            kubectl_delete "$resource"
        fi
    done
fi

if [[ $COMPONENT == "strategies" || $COMPONENT == "all" ]]; then
    echo "${ACTION}ing strategies"
    for strategy in "${STRATEGIES[@]}"; do
        if [[ ! -f $strategy ]] ; then
            die "File not found at: '${crd}'"
        fi

        if [[ $ACTION == "install" ]] ; then
            kubectl_apply "$strategy"
        fi
        if [[ $ACTION == "uninstall" ]] ; then
            kubectl_delete "$strategy"
        fi
    done
fi

# @mark.steps
# ----------------------------------------------------------------------------
# STEPS:
# ----------------------------------------------------------------------------

import os
import json
# Will be needed in future
# import time
# import urllib3

from behave import given, then, when
from kubernetes import client, config
from pyshould import should
from smoke.features.steps.openshift import Openshift
from smoke.features.steps.project import Project
from smoke.features.steps.command import Command as cmd

# Test results file path
scripts_dir = os.getenv('OUTPUT_DIR')

# variables needed to get the resource status
current_project = ''
config.load_kube_config()
oc = Openshift()
podStatus = {}

#scripts needed for installation
tekton_install = 'https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.30.0/release.yaml'
shipwright_deployment_install = 'https://github.com/shipwright-io/build/releases/download/v0.6.0/release.yaml'
shipwright_strategies_install = 'https://github.com/shipwright-io/build/releases/download/v0.6.0/sample-strategies.yaml'
# STEP
@given(u'Project "{project_name}" is used')
def given_project_is_used(context, project_name):
    project = Project(project_name)
    current_project = project_name
    context.current_project = current_project
    context.oc = oc
    if not project.is_present():
        print("Project is not present, creating project: {}...".format(project_name))
        project.create() | should.be_truthy.desc(
            "Project {} is created".format(project_name))
    print("Project {} is created!!!".format(project_name))
    context.project = project


def before_feature(context, feature):
    if scenario.name != None and "TEST_NAMESPACE" in scenario.name:
        print("Scenario using env namespace subtitution found: {0}, env: {}".format(scenario.name, os.getenv("TEST_NAMESPACE")))
        scenario.name = txt.replace("TEST_NAMESPACE", os.getenv("TEST_NAMESPACE"))

# STEP
@given(u'Project [{project_env}] is used')
def given_namespace_from_env_is_used(context, project_env):
    env = os.getenv(project_env)
    assert env is not None, f"{project_env} environment variable needs to be set"
    print(f"{project_env} = {env}")
    given_project_is_used(context, env)
    
@given(u'we have a openshift cluster')
def loginCluster(context):
    print("Using [{}]".format(current_project))

@when(u'we install tekton')
def install_tekton(context):
    res = oc.oc_apply(tekton_install)
    if res is None:
        raise AssertionError
    else:
        print(res)

@when(u'check if tekton-pipelines-controller & tekton-pipelines-webhook deployment are in READY state')
def check_tekton_deployments(context):
    tp_controller_status = oc.get_resource_info_by_jsonpath('deployment','tekton-pipelines-controller','tekton-pipelines','{.status.unavailableReplicas}')
    tp_webhook_status = oc.get_resource_info_by_jsonpath('deployment','tekton-pipelines-webhook','tekton-pipelines','{.status.unavailableReplicas}')
    if tp_controller_status is not 0 or tp_webhook_status is not 0:
        print('tekton-pipelines-controller unavailableReplicas needed 0 available ', tp_controller_status,' and tekton-pipelines-webhook unavailableReplicas needed 0 available ',tp_webhook_status)
        # raise AssertionError
    else:
        print('tekton-pipelines-controller unavailableReplicas needed 0 available ', tp_controller_status,' and tekton-pipelines-webhook unavailableReplicas needed 0 available ',tp_webhook_status)

@then(u'we install shipwright deployment')
def shp_deployment_install(context):
    res = oc.oc_apply(shipwright_deployment_install)
    if res is None:
        raise AssertionError
    else:
        print(res)

@then(u'namespace/shipwright-build should created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then namespace/shipwright-build should created')


@then(u'role.rbac.authorization.k8s.io/shipwright-build-controller should be created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then role.rbac.authorization.k8s.io/shipwright-build-controller should be created')


@then(u'clusterrole.rbac.authorization.k8s.io/shipwright-build-controller should be created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then clusterrole.rbac.authorization.k8s.io/shipwright-build-controller should be created')


@then(u'clusterrolebinding.rbac.authorization.k8s.io/shipwright-build-controller should be created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then clusterrolebinding.rbac.authorization.k8s.io/shipwright-build-controller should be created')


@then(u'rolebinding.rbac.authorization.k8s.io/shipwright-build-controller should be created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then rolebinding.rbac.authorization.k8s.io/shipwright-build-controller should be created')


@then(u'serviceaccount/shipwright-build-controller should be created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then serviceaccount/shipwright-build-controller should be created')


@then(u'deployment.apps/shipwright-build-controller should be created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then deployment.apps/shipwright-build-controller should be created')


@then(u'customresourcedefinition.apiextensions.k8s.io/buildruns.shipwright.io should be created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then customresourcedefinition.apiextensions.k8s.io/buildruns.shipwright.io should be created')


@then(u'customresourcedefinition.apiextensions.k8s.io/builds.shipwright.io should be created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then customresourcedefinition.apiextensions.k8s.io/builds.shipwright.io should be created')


@then(u'customresourcedefinition.apiextensions.k8s.io/buildstrategies.shipwright.io should be created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then customresourcedefinition.apiextensions.k8s.io/buildstrategies.shipwright.io should be created')


@then(u'customresourcedefinition.apiextensions.k8s.io/clusterbuildstrategies.shipwright.io should be created')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then customresourcedefinition.apiextensions.k8s.io/clusterbuildstrategies.shipwright.io should be created')


@then(u'we check shipwright-build-controller deployment should be in READY state')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then we check shipwright-build-controller deployment should be in READY state')


@then(u'shipwright-build-controller pod should be in READY state')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then shipwright-build-controller pod should be in READY state')


@then(u'we install Shipwright strategies')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then we install Shipwright strategies')


@then(u'check clusterbuildstrategy.shipwright.io/ with "oc get cbs"')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then check clusterbuildstrategy.shipwright.io/ with "oc get cbs"')
'''
This module is used as a wrapper for openshift cli,
one can create the Openshift object & use the 
functionalities to check the resources.
'''

import re
import time

from kubernetes import client, config
from pyshould import should
from smoke.features.steps.command import Command

'''
Openshift class provides the blueprint that we need to
interact with the openshift resources 
'''


class Openshift(object):
    def __init__(self):
        self.cmd = Command()

    def get_pod_lst(self, namespace):
        return self.get_resource_lst("pods", namespace)

    def get_resource_lst(self, resource_plural, namespace):
        (output, exit_code) = self.cmd.run(f'oc get {resource_plural} -n {namespace} -o "jsonpath={{.items['
                                           f'*].metadata.name}}"')
        exit_code | should.be_equal_to(0)
        return output

    def search_item_in_lst(self, lst, search_pattern):
        lst_arr = lst.split(" ")
        for item in lst_arr:
            if re.match(search_pattern, item) is not None:
                print(f"item matched {item}")
                return item
        print("Given item not matched from the list of pods")
        return None

    def search_pod_in_namespace(self, pod_name_pattern, namespace):
        return self.search_resource_in_namespace("pods", pod_name_pattern, namespace)

    def search_resource_in_namespace(self, resource_plural, name_pattern, namespace):
        print(f"Searching for {resource_plural} that matches {name_pattern} in {namespace} namespace")
        lst = self.get_resource_lst(resource_plural, namespace)
        if len(lst) != 0:
            print("Resource list is {}".format(lst))
            return self.search_item_in_lst(lst, name_pattern)
        else:
            print('Resource list is empty under namespace - {}'.format(namespace))
            return None

    def is_resource_in(self, resource_type):
        output, exit_code = self.cmd.run(f'oc get {resource_type}')
        return exit_code == 0

    def wait_for_pod(self, pod_name_pattern, namespace, interval=5, timeout=60):
        pod = self.search_pod_in_namespace(pod_name_pattern, namespace)
        start = 0
        if pod is not None:
            return pod
        else:
            while ((start + interval) <= timeout):
                pod = self.search_pod_in_namespace(pod_name_pattern, namespace)
                if pod is not None:
                    return pod
                time.sleep(interval)
                start += interval
        return None

    def check_pod_status(self, pod_name, namespace, wait_for_status="Running"):
        cmd = f'oc get pod {pod_name} -n {namespace} -o "jsonpath={{.status.phase}}"'
        status_found, output, exit_status = self.cmd.run_wait_for_status(cmd, wait_for_status)
        return status_found

    def get_pod_status(self, pod_name, namespace):
        cmd = f'oc get pod {pod_name} -n {namespace} -o "jsonpath={{.status.phase}}"'
        output, exit_status = self.cmd.run(cmd)
        print(f"Get pod status: {output}, {exit_status}")
        if exit_status == 0:
            return output
        return None
    
    def new_app(self, template_name, namespace):
        cmd = f'oc new-app {template_name} -n {namespace}'
        output, exit_status = self.cmd.run(cmd)
        print(f"starting: {output}, {exit_status}")
        if exit_status == 0:
            return output
        return None

    def oc_apply(self, yaml):
        cmd = f'oc apply -f {yaml}'
        (output, exit_status) = self.cmd.run(cmd)
        print(f"starting: {output}, {exit_status}")
        if exit_status == 0:
            return output
        return None
    
    def oc_create_from_yaml(self, yaml):
        cmd = f'oc create -f {yaml}'
        output, exit_status = self.cmd.run(cmd)
        print(f"starting: {output}, {exit_status}")
        if exit_status == 0:
            return output
        return None


    def new_app_with_params(self, template_name, paramsfile):
        # oc new-app ruby-helloworld-sample --param-file=helloworld.params
        cmd = f'oc new-app {template_name} --param-file={paramsfile}'
        output, exit_status = self.cmd.run(cmd)
        print(f"starting: {output}, {exit_status}")
        if exit_status == 0:
            return output
        return None
    
    def new_app_from_file(self,file_url,namespace):
        cmd = f'oc new-app -f {file_url} -n {namespace}'
        output, exit_status = self.cmd.run(cmd)
        print(f"starting: {output}, {exit_status}")
        if exit_status == 0:
            return output
        return None

    def start_build(self,buildconfig,namespace):
        cmd = f'oc start-build {buildconfig} -n {namespace}'
        output, exit_status = self.cmd.run(cmd)
        print(f"starting: {output}, {exit_status}")
        if exit_status == 0:
            return output
        return None

    def get_configmap(self, namespace):
        output, exit_code = self.cmd.run(f'oc get cm -n {namespace}')
        exit_code | should.be_equal_to(0)
        return output

    def get_deploymentconfig(self, namespace):
        output, exit_code = self.cmd.run(f'oc get dc -n {namespace}')
        exit_code | should.be_equal_to(0)
        return output

    def get_service(self, namespace):
        output, exit_code = self.cmd.run(f'oc get svc -n {namespace}')
        exit_code | should.be_equal_to(0)
        return output

    def get_service_account(self, namespace):
        output, exit_code = self.cmd.run(f'oc get sa -n {namespace}')
        exit_code | should.be_equal_to(0)
        return output

    def get_role_binding(self, namespace):
        output, exit_code = self.cmd.run(f'oc get rolebinding -n {namespace}')
        exit_code | should.be_equal_to(0)
        return output

    def get_route(self, route_name, namespace):
        output, exit_code = self.cmd.run(f'oc get route {route_name} -n {namespace}')
        exit_code | should.be_equal_to(0)
        return output

    def expose_service_route(self, service_name, namespace):
        output, exit_code = self.cmd.run(f'oc expose svc/{service_name} -n {namespace} --name={service_name}')
        return re.search(r'.*%s\sexposed' % service_name, output)

    def get_route_host(self, name, namespace):
        (output, exit_code) = self.cmd.run(
            f'oc get route {name} -n {namespace} -o "jsonpath={{.status.ingress[0].host}}"')
        exit_code | should.be_equal_to(0)
        return output

    def check_for_deployment_status(self, deployment_name, namespace, wait_for_status="True"):
        deployment_status_cmd = f'oc get deployment {deployment_name} -n {namespace} -o "jsonpath={{' \
                                f'.status.conditions[*].status}}" '
        deployment_status, exit_code = self.cmd.run_wait_for_status(deployment_status_cmd, wait_for_status, 5, 300)
        exit_code | should.be_equal_to(0)
        return deployment_status

    def check_for_deployment_config_status(self, dc_name, namespace, wait_for="condition=Available"):
        output, exit_code = self.cmd.run_wait_for('dc', 'jenkins', wait_for, timeout_seconds=300)
        if exit_code != 0:
            print(output)
        return output, exit_code

    def set_env_for_deployment_config(self, name, namespace, key, value):
        env_cmd = f'oc -n {namespace} set env dc/{name} {key}={value}'
        print( "oc set command: {}".format(env_cmd))
        output, exit_code = self.cmd.run(env_cmd)
        exit_code | should.be_equal_to(0)
        time.sleep(3)
        return  output, exit_code

    def get_deployment_env_info(self, name, namespace):
        env_cmd = f'oc get deploy {name} -n {namespace} -o "jsonpath={{.spec.template.spec.containers[0].env}}"'
        env, exit_code = self.cmd.run(env_cmd)
        exit_code | should.be_equal_to(0)
        return env

    def get_deployment_envFrom_info(self, name, namespace):
        env_from_cmd = f'oc get deploy {name} -n {namespace} -o "jsonpath={{.spec.template.spec.containers[0].envFrom}}"'
        env_from, exit_code = self.cmd.run(env_from_cmd)
        exit_code | should.be_equal_to(0)
        return env_from

    def get_resource_info_by_jsonpath(self, resource_type, name, namespace, json_path, wait=False):
        output, exit_code = self.cmd.run(f'oc get {resource_type} {name} -n {namespace} -o "jsonpath={json_path}"')
        if exit_code != 0:
            if wait:
                attempts = 5
                while exit_code != 0 and attempts > 0:
                    output, exit_code = self.cmd.run(
                        f'oc get {resource_type} {name} -n {namespace} -o "jsonpath={json_path}"')
                    attempts -= 1
                    time.sleep(5)
        exit_code | should.be_equal_to(0).desc(f'Exit code should be 0:\n OUTPUT:\n{output}')
        return output

    def get_resource_info_by_jq(self, resource_type, name, namespace, jq_expression, wait=False):
        output, exit_code = self.cmd.run(
            f'oc get {resource_type} {name} -n {namespace} -o json | jq -rc \'{jq_expression}\'')
        if exit_code != 0:
            if wait:
                attempts = 5
                while exit_code != 0 and attempts > 0:
                    output, exit_code = self.cmd.run(
                        f'oc get {resource_type} {name} -n {namespace} -o json | jq -rc \'{jq_expression}\'')
                    attempts -= 1
                    time.sleep(5)
        exit_code | should.be_equal_to(0).desc(f'Exit code should be 0:\n OUTPUT:\n{output}')
        return output.rstrip("\n")
    
    def exec_container_in_pod(self, container_name, pod_name, container_cmd):
        cmd = f'oc exec {pod_name} -c {container_name} {container_cmd}'
        output, exit_status = self.cmd.run(cmd)
        print(f"Inside the {pod_name} container {container_name}: {output}, {exit_status}")
        if exit_status == 0:
            return output
        return None
    
    def exec_in_pod(self, pod_name, container_cmd):
        cmd = f'oc exec {pod_name} -- {container_cmd}'
        output, exit_status = self.cmd.run(cmd)
        print(f"Inside the {pod_name}: {output}, {exit_status}")
        if exit_status == 0:
            return output
        return None
    
    def oc_process_template(self,file_path):
        cmd = f'oc process -f {file_path}|oc create -f -'
        output, exit_status = self.cmd.run(cmd)
        print(f"Proccesing {file_path} template with : {output}")
        if exit_status == 0:
            return output
        return None

    def delete(self,resource_type: str,resource: str, namespace: str):
        '''
        Delete resources in a specific namespace
        '''
        cmd = f'oc delete {resource_type} {resource} -n {namespace}'
        output, exit_status = self.cmd.run(cmd)
        print(f"{output}")
        if exit_status == 0:
            return output
        return None

    def getmasterpod(self, namespace: str)-> str:
        '''
        returns the jenkins master pod name
        '''
        v1 = client.CoreV1Api()
        pods = v1.list_namespaced_pod(namespace, label_selector='deploymentconfig=jenkins')
        if len(pods.items) > 1:
            raise AssertionError
        return pods.items[0].metadata.name
    
    def scaleReplicas(self, namespace: str, replicas: int, rep_controller: str):
        '''
        Scales up or down the pod count that ensures that a specified number of replicas of a pod are running at all times.\n
        namespace -> project name\n
        replicas -> desried count of the pod running all times.\n
        rep_controller -> replication controller name\n
        e.g: oc scale --replicas=2 rc/jenkins-1
        '''
        cmd = f'oc scale --replicas={replicas} rc/{rep_controller} -n {namespace}'
        output, exit_status = self.cmd.run(cmd)
        print(f"{output}")
        if exit_status == 0:
            return output
        return None
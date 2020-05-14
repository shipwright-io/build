#!/bin/bash

set -euo pipefail

# Read the build run name

if [ $# -eq 0 ]; then
    echo "Usage: build-logs <BUILD_RUN_NAME> [-f|--follow] [-n|--namespace <NAMESPACE_NAME>] [--timestamps]"
    exit 1
fi

BUILD_RUN_NAME=$1
shift

# Read the other arguments
FOLLOW=0
NAMESPACE=
TIMESTAMPS=0

while [[ $# -gt 0 ]]
do
    ARGKEY="$1"

    if [ "${ARGKEY}" == "-f" ] || [ "${ARGKEY}" == "--follow" ]; then
        FOLLOW=1
        shift
    elif [ "${ARGKEY}" == "-n" ] || [ "${ARGKEY}" == "--namespace" ]; then
        NAMESPACE=$2
        shift
        shift
    elif [ "${ARGKEY}" == "--timestamps" ]; then
        TIMESTAMPS=1
        shift
    else
        echo "Usage: build-logs <BUILD_RUN_NAME> [-f|--follow] [-n|--namespace <NAMESPACE_NAME>] [--timestamps]"
        exit 1
    fi
done

# Retrieve the namespace if none was provided, use the kubeconfig's current context and fall back to default
# if nothing is set. This is the same logic that a kubectl get would do. Do not use kubectl namespace here as
# the user might not be allowed to read them.

if [ "${NAMESPACE}" == "" ]; then
    CONFIG=$(kubectl config view -o json)

    CURRENT_CONTEXT=$(echo "${CONFIG}" | jq -r '.["current-context"]')

    NAMESPACE=$(echo "${CONFIG}" | jq -r ".contexts[] | select(.name == \"${CURRENT_CONTEXT}\") | .context.namespace")
    if [ "${NAMESPACE}" == "null" ]; then
        NAMESPACE=default
    fi
fi

# Verify the build run exists

if ! kubectl get buildrun "${BUILD_RUN_NAME}" -n "${NAMESPACE}" >/dev/null 2>&1; then
    echo "A build run with the name '${BUILD_RUN_NAME}' cannot be found in the namespace '${NAMESPACE}'."
    exit 2
fi

echo "Build run: ${BUILD_RUN_NAME}"

# Extract the task run name from the buildrun

TASK_RUN_NAME=$(kubectl get buildrun "${BUILD_RUN_NAME}" -n "${NAMESPACE}" -o json | jq -r '.status.latestTaskRunRef')

echo "Task run: ${TASK_RUN_NAME}"

# Find the pod, it has labels for the build run and task run name

PODS=$(kubectl get pods -l "buildrun.build.dev/name=${BUILD_RUN_NAME},tekton.dev/taskRun=${TASK_RUN_NAME}" -n "${NAMESPACE}" -o json)
PODS_LENGTH=$(echo "${PODS}" | jq ".items | length")

if [ "${PODS_LENGTH}" -eq 0 ]; then
    echo "No pod found. There is probably a problem in your build configuration."
    exit 3
elif [ "${PODS_LENGTH}" -ne 1 ]; then
    echo "More than one pod found, that's unexpected."
    exit 4
fi

POD=$(echo "${PODS}" | jq ".items[0]")
POD_NAME=$(echo "${POD}" | jq -r ".metadata.name")

echo "Pod: ${POD_NAME}"
echo

# Retrieve the number of containers

CONTAINERS_LENGTH=$(echo "${POD}" | jq ".spec.containers | length")

# Iterate the containers

for (( i = 0; i < CONTAINERS_LENGTH; i++ ))
do
    # Extract the container name

    CONTAINER_NAME=$(echo "${POD}" | jq -r ".spec.containers[${i}].name")
    echo "Logs of container ${CONTAINER_NAME}:"
    echo

    # Check if the container is waiting

    WAITING_REASON=$(echo "${POD}" | jq -r ".status.containerStatuses[${i}].state.waiting.reason")

    if [ "${WAITING_REASON}" != "null" ]; then
        # In follow mode, wait for the container not to be waiting anymore, otherwise stop here
        if [ ${FOLLOW} = 1 ]; then
            while [ "${WAITING_REASON}" != "null" ]; do
                echo "Container is not yet running. Waiting reason: ${WAITING_REASON}"
                sleep 1

                POD=$(kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o json)
                WAITING_REASON=$(echo "${POD}" | jq -r ".status.containerStatuses[${i}].state.waiting.reason")
            done
        else
            echo "Container is not yet running. Waiting reason: ${WAITING_REASON}"
            echo
            break
        fi
    fi

    # Extract the exit code of the container

    EXIT_CODE=$(echo "${POD}" | jq ".status.containerStatuses[${i}].state.terminated.exitCode")

    if [ ${FOLLOW} = 1 ] && [ "${EXIT_CODE}" == "null" ]; then
        # Container is still running and follow logs is requested
        if [ ${TIMESTAMPS} = 1 ]; then
            kubectl logs "${POD_NAME}" "${CONTAINER_NAME}" -f -n "${NAMESPACE}" --timestamps
        else
            kubectl logs "${POD_NAME}" "${CONTAINER_NAME}" -f -n "${NAMESPACE}"
        fi
        
        # Refresh the pod to get the exit code of the container, sometimes this takes a moment
        while [ "${EXIT_CODE}" == "null" ]; do
            POD=$(kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o json)
            EXIT_CODE=$(echo "${POD}" | jq ".status.containerStatuses[${i}].state.terminated.exitCode")
        done
    else
        # Just print the logs that we have
        if [ ${TIMESTAMPS} = 1 ]; then
            kubectl logs "${POD_NAME}" "${CONTAINER_NAME}" -n "${NAMESPACE}" --timestamps
        else
            kubectl logs "${POD_NAME}" "${CONTAINER_NAME}" -n "${NAMESPACE}"
        fi

        if [ "${EXIT_CODE}" == "null" ]; then
            # Refresh the pod to get the exit code of the container if it terminated in the meantime
            POD=$(kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o json)
            EXIT_CODE=$(echo "${POD}" | jq ".status.containerStatuses[${i}].state.terminated.exitCode")
        fi
    fi

    echo

    # Print a message depending on the exit code

    if [ "${EXIT_CODE}" == "null" ]; then
        echo "Container is still running."
        echo
        break
    elif [ "${EXIT_CODE}" == "0" ]; then
        echo "Container terminated successfully."
    else 
        echo "Container terminated with an error. Status code: ${EXIT_CODE}"
    fi

    echo
done

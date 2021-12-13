Feature: Install and verify the behaviour of Shipwright

    With Shipwright, developers get a simplified approach for building container images, by defining 
    a minimal YAML that does not require any previous knowledge of containers or container tooling. 
    All you need is your source code in git and access to a container registry.

    Scenario: Install and verify the behaviour of shipwright on OpenShift
        Given we have a openshift cluster
        When we install tekton
        And check if tekton-pipelines-controller & tekton-pipelines-webhook deployment are in READY state
        Then we install shipwright deployment
        And namespace/shipwright-build should created
        And role.rbac.authorization.k8s.io/shipwright-build-controller should be created
        And clusterrole.rbac.authorization.k8s.io/shipwright-build-controller should be created
        And clusterrolebinding.rbac.authorization.k8s.io/shipwright-build-controller should be created
        And rolebinding.rbac.authorization.k8s.io/shipwright-build-controller should be created
        And serviceaccount/shipwright-build-controller should be created
        And deployment.apps/shipwright-build-controller should be created
        And customresourcedefinition.apiextensions.k8s.io/buildruns.shipwright.io should be created
        And customresourcedefinition.apiextensions.k8s.io/builds.shipwright.io should be created
        And customresourcedefinition.apiextensions.k8s.io/buildstrategies.shipwright.io should be created
        And customresourcedefinition.apiextensions.k8s.io/clusterbuildstrategies.shipwright.io should be created
        And we check shipwright-build-controller deployment should be in READY state
        And shipwright-build-controller pod should be in READY state
        Then we install Shipwright strategies
        And check clusterbuildstrategy.shipwright.io/ with "oc get cbs"
        |name                     |created|
        |buildkit                 |True   |
        |buildkit-v3              |True   |
        |buildpacks-v3-heroku     |True   |
        |kaniko                   |True   |
        |kaniko-trivy             |True   |
        |ko                       |True   |
        |source-to-image-redhat   |True   |
        |source-to-image          |True   |

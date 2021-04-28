// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	v1 "k8s.io/api/core/v1"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
)

const (
	// runtimeDockerfileTmpl Dockerfile template to be used with runtime-image, it uses Build
	// attributes directly as template input.
	runtimeDockerfileTmpl = `FROM $(params.shp-output-image) as builder

FROM {{ .Spec.Runtime.Base.Image }}

{{- range $k, $v := .Spec.Runtime.Env }}
ENV {{ $k }}="{{ $v }}"
{{- end }}

{{- range $k, $v := .Spec.Runtime.Labels }}
LABEL {{ $k }}="{{ $v }}"
{{- end }}

{{- range $cmd := .Spec.Runtime.Run }}
RUN {{ $cmd }}
{{- end }}

{{- $userAndGroup := renderUserAndGroup .Spec.Runtime.User -}}
{{- $chown := "" }}
{{- if ne $userAndGroup "" -}}
{{- $chown = printf "--chown=\"%s\"" $userAndGroup }}
{{- end -}}

{{- range $dir := .Spec.Runtime.Paths }}
{{- $parts := splitPaths $dir }}
COPY {{ $chown }} --from=builder "{{ index $parts 0 }}" "{{ index $parts 1 }}"
{{- end }}

{{- if .Spec.Runtime.WorkDir }}
WORKDIR "{{ .Spec.Runtime.WorkDir }}"
{{- end }}

{{- if ne $userAndGroup "" }}
USER {{ $userAndGroup }}
{{- end }}

{{- if .Spec.Runtime.Entrypoint }}
ENTRYPOINT [ {{ renderEntrypoint .Spec.Runtime.Entrypoint }} ]
{{- end -}}
`

	// runtimeDockerfile runtime Dockerfile file name.
	runtimeDockerfile = "Dockerfile.runtime"

	// defultShellImage default image for a simple shell instance.
	defultShellImage = "busybox:latest"
)

// rootUserID root's UID
var rootUserID = int64(0)

// renderUserAndGroup based on informed user, returns it joined by colon (":"), or empty string when
// nil or user not informed. Follows the rules for Dockerfile's USER and "COPY --chown" directives.
func renderUserAndGroup(user *buildv1alpha1.User) string {
	if user == nil || user.Name == "" {
		return ""
	}
	if user.Group == "" {
		return user.Name
	}
	return fmt.Sprintf("%s:%s", user.Name, user.Group)
}

// splitPaths split informed path by colon, returning a slice with parts. When colon is not present,
// return the informed directory twice. This method always returns a slice with two entries.
func splitPaths(dir string) []string {
	parts := strings.Split(dir, ":")
	if len(parts) == 2 {
		return parts
	}
	return []string{dir, dir}
}

// renderEntrypoint will take a slice of strings and render the notation expected on ENTRYPOINT.
func renderEntrypoint(e []string) string {
	entrypoint := []string{}
	for _, cmd := range e {
		entrypoint = append(entrypoint, strconv.Quote(cmd))
	}
	return strings.Join(entrypoint, ", ")
}

// renderRuntimeDockerfile render runtime Dockerfile using build instance and pre-defined template.
func renderRuntimeDockerfile(b *buildv1alpha1.Build) (*bytes.Buffer, error) {
	tmpl, err := template.New(runtimeDockerfile).
		Funcs(template.FuncMap{
			"renderUserAndGroup": renderUserAndGroup,
			"splitPaths":         splitPaths,
			"renderEntrypoint":   renderEntrypoint,
		}).
		Parse(runtimeDockerfileTmpl)
	if err != nil {
		return nil, err
	}

	dockerfile := new(bytes.Buffer)
	if err = tmpl.Execute(dockerfile, b); err != nil {
		return nil, err
	}
	return dockerfile, nil
}

// runtimeDockerfileTransformations search and replace special variables `$()` in informed string.
func runtimeDockerfileTransformations(b *buildv1alpha1.Build, str string) string {
	transformations := map[string]string{
		"$(workspace)": fmt.Sprintf("$(params.%s%s)", prefixParamsResults, paramSourceContext),
	}
	for k, v := range transformations {
		str = strings.ReplaceAll(str, k, v)
	}
	return str
}

// runtimeDockerfileStep trigger the rendering of Dockerfile.runtime, and use this input as a
// build-step to create a new file.
func runtimeDockerfileStep(b *buildv1alpha1.Build) (*v1beta1.Step, error) {
	dockerfile, err := renderRuntimeDockerfile(b)
	if err != nil {
		return nil, err
	}
	// appling known transformation to dockerfile payload, therefore the Shipwright build controller variables are
	// applicable to all parts of the runtime Dockerfile as well
	dockerfileTransformed := runtimeDockerfileTransformations(b, dockerfile.String())

	// using builder-image when defined, or falling back to a default
	imageURL := defultShellImage
	if isBuilderDefined(b) {
		imageURL = b.Spec.Builder.Image
	}

	container := v1.Container{
		Name:  "runtime-dockerfile",
		Image: imageURL,
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &rootUserID,
		},
		WorkingDir: fmt.Sprintf("$(params.%s%s)", prefixParamsResults, paramSourceRoot),
		Command:    []string{"/bin/sh"},
		Args: []string{
			"-x",
			"-c",
			fmt.Sprintf("echo '%s' >%s", dockerfileTransformed, runtimeDockerfile),
		},
	}
	return &v1beta1.Step{Container: container}, nil
}

// runtimeBuildAndPushStep returns a Task step to build the Dockerfile.runtime with kaniko.
func runtimeBuildAndPushStep(b *buildv1alpha1.Build, kanikoImage string) *v1beta1.Step {
	container := v1.Container{
		Name:       "kaniko-build-and-push",
		Image:      kanikoImage,
		WorkingDir: fmt.Sprintf("$(params.%s%s)", prefixParamsResults, paramSourceRoot),
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &rootUserID,
			Capabilities: &v1.Capabilities{
				Add: []v1.Capability{
					v1.Capability("CHOWN"),
					v1.Capability("DAC_OVERRIDE"),
					v1.Capability("FOWNER"),
					v1.Capability("SETGID"),
					v1.Capability("SETUID"),
					v1.Capability("SETFCAP"),
					v1.Capability("KILL"),
				},
			},
		},
		Env: []v1.EnvVar{
			{Name: "DOCKER_CONFIG", Value: "/tekton/home/.docker"},
			{Name: "AWS_ACCESS_KEY_ID", Value: "NOT_SET"},
			{Name: "AWS_SECRET_KEY", Value: "NOT_SET"},
		},
		Command: []string{"/kaniko/executor"},
		Args: []string{
			"--skip-tls-verify=true",
			fmt.Sprintf("--dockerfile=%s", runtimeDockerfile),
			fmt.Sprintf("--destination=$(params.%s%s)", prefixParamsResults, paramOutputImage),
			"--snapshotMode=redo",
			"--oci-layout-path=/workspace/output/image",
		},
	}
	return &v1beta1.Step{Container: container}
}

// AmendTaskSpecWithRuntimeImage add more steps to Tekton's Task in order to create the
// runtime-image.
func AmendTaskSpecWithRuntimeImage(
	cfg *config.Config,
	spec *v1beta1.TaskSpec,
	b *buildv1alpha1.Build,
) error {
	step, err := runtimeDockerfileStep(b)
	if err != nil {
		return err
	}
	spec.Steps = append(spec.Steps, *step)

	step = runtimeBuildAndPushStep(b, cfg.KanikoContainerImage)
	spec.Steps = append(spec.Steps, *step)
	return nil
}

// isBuilderDefined inspect if build contains `.spec.Builder` defined.
func isBuilderDefined(b *buildv1alpha1.Build) bool {
	if b.Spec.Builder == nil {
		return false
	}
	if b.Spec.Builder.Image == "" {
		return false
	}
	return true
}

// IsRuntimeDefined inspect if build has `.spec.runtime` defined, checking intermediary attributes
// and making sure Image is informed.
func IsRuntimeDefined(b *buildv1alpha1.Build) bool {
	if b.Spec.Runtime == nil {
		return false
	}
	if b.Spec.Runtime.Base.Image == "" {
		return false
	}
	return true
}

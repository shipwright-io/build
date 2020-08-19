package buildrun

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"
	"text/template"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller/utils"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
)

const (
	// runtimeDockerfileTmpl Dockerfile template to be used with runtime-image, it uses Build
	// attributes directly as template input.
	runtimeDockerfileTmpl = `FROM {{ .Spec.Output.ImageURL }} as builder

FROM {{ .Spec.Runtime.Base.ImageURL }}

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

	// workspaceDir common workspace directory, where the source code is located.
	workspaceDir = "/workspace/source"

	// runtimeDockerfile runtime Dockerfile file name.
	runtimeDockerfile = "Dockerfile.runtime"

	// defultShellImage default image for a simple shell instance.
	defultShellImage = "busybox:latest"
)

// rootUserID root's UID
var rootUserID = int64(0)

// runtimeDockerfilePath path to runtime Dockerfile on workspace directory.
var runtimeDockerfilePath = path.Join(workspaceDir, runtimeDockerfile)

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
	contextDir := getContextDir(b)
	transformations := map[string]string{
		"$(workspace)": path.Join(workspaceDir, contextDir),
	}
	for k, v := range transformations {
		str = strings.ReplaceAll(str, k, v)
	}
	return str
}

// getContextDir retrieve contextDir from Source, or empty string.
func getContextDir(b *buildv1alpha1.Build) string {
	contextDir := ""
	if b != nil && b.Spec.Source.ContextDir != nil {
		contextDir = path.Join(workspaceDir, *b.Spec.Source.ContextDir)
	}
	return contextDir
}

// runtimeDockerfileStep trigger the rendering of Dockerfile.runtime, and use this input as a
// build-step to create a new file.
func runtimeDockerfileStep(b *buildv1alpha1.Build) (*v1beta1.Step, error) {
	dockerfile, err := renderRuntimeDockerfile(b)
	if err != nil {
		return nil, err
	}
	// appling known transformation to dockerfile payload, therefore operator variables are
	// applicable to all parts of the runtime Dockerfile as well
	dockerfileTransformed := runtimeDockerfileTransformations(b, dockerfile.String())

	// using builder-image when defined, or falling back to a default
	imageURL := defultShellImage
	if utils.IsBuilderImageDefined(b) {
		imageURL = b.Spec.BuilderImage.ImageURL
	}

	container := v1.Container{
		Name:  "runtime-dockerfile",
		Image: imageURL,
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &rootUserID,
		},
		WorkingDir: workspaceDir,
		Command:    []string{"/bin/sh"},
		Args: []string{
			"-x",
			"-c",
			fmt.Sprintf("echo '%s' >%s", dockerfileTransformed, runtimeDockerfilePath),
		},
	}
	return &v1beta1.Step{Container: container}, nil
}

// runtimeBuildAndPushStep returns a Task step to build the Dockerfile.runtime with kaniko.
func runtimeBuildAndPushStep(b *buildv1alpha1.Build, kanikoImage string) *v1beta1.Step {
	contextDir := getContextDir(b)
	container := v1.Container{
		Name:       "kaniko-build-and-push",
		Image:      kanikoImage,
		WorkingDir: workspaceDir,
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &rootUserID,
			Capabilities: &v1.Capabilities{
				Add: []v1.Capability{
					v1.Capability("CHOWN"),
					v1.Capability("DAC_OVERRIDE"),
					v1.Capability("FOWNER"),
					v1.Capability("SETGID"),
					v1.Capability("SETUID"),
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
			fmt.Sprintf("--context=%x", path.Join(workspaceDir, contextDir)),
			fmt.Sprintf("--destination=%s", b.Spec.Output.ImageURL),
			"--snapshotMode=redo",
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

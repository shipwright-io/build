package buildrun

import (
	"reflect"
	"testing"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/redhat-developer/build/pkg/config"
)

var runtimeBuild = &buildv1alpha1.Build{
	Spec: buildv1alpha1.BuildSpec{
		BuilderImage: &buildv1alpha1.Image{
			ImageURL: "test/builder-image:latest",
		},
		Output: buildv1alpha1.Image{
			ImageURL: "test/output-image:latest",
		},
		Runtime: &buildv1alpha1.Runtime{
			Base: buildv1alpha1.Image{
				ImageURL: "test/base-image:latest",
			},
			Env: map[string]string{
				"ENVIRONMENT_VARIABLE": "VALUE",
			},
			Labels: map[string]string{
				"label": "value",
			},
			WorkDir: "/workdir",
			Run:     []string{"command --args"},
			User: &buildv1alpha1.User{
				Name:  "username",
				Group: "1001",
			},
			Paths:      []string{"/path/to/a:/new/path/to/a", "/path/to/b"},
			Entrypoint: []string{"/bin/bash", "-x", "-c"},
		},
	},
}

const dockerfile = `FROM test/output-image:latest as builder

FROM test/base-image:latest
ENV ENVIRONMENT_VARIABLE="VALUE"
LABEL label="value"
RUN command --args
COPY --chown="username:1001" --from=builder "/path/to/a" "/new/path/to/a"
COPY --chown="username:1001" --from=builder "/path/to/b" "/path/to/b"
WORKDIR "/workdir"
USER username:1001
ENTRYPOINT [ "/bin/bash", "-x", "-c" ]`

func TestAmendTaskSpecWithRuntimeImage(t *testing.T) {
	taskSpec := &v1beta1.TaskSpec{
		Steps: []v1beta1.Step{},
	}
	err := AmendTaskSpecWithRuntimeImage(config.NewDefaultConfig(), taskSpec, runtimeBuild)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(taskSpec.Steps) != 2 {
		t.Errorf("expected taskSpec to have %d steps, got %d", 2, len(taskSpec.Steps))
	}

}

func TestRenderEntrypoint(t *testing.T) {
	entrypointArray := []string{"/bin/bash", "-x", "-c"}
	expectedResult := "\"/bin/bash\", \"-x\", \"-c\""
	renderedEntrypoint := renderEntrypoint(entrypointArray)
	if renderedEntrypoint != expectedResult {
		t.Errorf("expected entrypoint to render to %s, got %s", renderedEntrypoint, expectedResult)
	}
}

func TestRenderUserAndGroup(t *testing.T) {
	testCases := []struct {
		name           string
		user           *buildv1alpha1.User
		expectedResult string
	}{
		{
			name: "empty",
			user: &buildv1alpha1.User{},
		},
		{
			name: "group only",
			user: &buildv1alpha1.User{
				Group: "group",
			},
		},
		{
			name: "user only",
			user: &buildv1alpha1.User{
				Name: "username",
			},
			expectedResult: "username",
		},
		{
			name: "user and group",
			user: &buildv1alpha1.User{
				Name:  "username",
				Group: "1001",
			},
			expectedResult: "username:1001",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userGroup := renderUserAndGroup(tc.user)
			if userGroup != tc.expectedResult {
				t.Errorf("expected user:group %s, got %s", tc.expectedResult, userGroup)
			}
		})
	}
}

func TestSplitPaths(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedResult []string
	}{
		{
			name:           "single",
			input:          "a",
			expectedResult: []string{"a", "a"},
		},
		{
			name:           "double",
			input:          "a:b",
			expectedResult: []string{"a", "b"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := splitPaths(tc.input)
			if !reflect.DeepEqual(output, tc.expectedResult) {
				t.Errorf("expected split paths to be %s, got %s", tc.expectedResult, output)
			}
		})
	}
}

func TestRenderRuntimeDockerfile(t *testing.T) {
	result, err := renderRuntimeDockerfile(runtimeBuild)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resultString := result.String()
	if resultString != dockerfile {
		t.Errorf("expected runtime dockerfile to be:\n%s\n\ngot:\n%s", dockerfile, resultString)
	}
}

package build

/*
func generateS2iTaskRun(instance *buildv1alpha1.BuildStrategy) *taskv1.Task {
	var steps []taskv1.Step

	for _, value := range steps {
		steps = append(steps,
			corev1.Container{
				Image:   value.Image,
				Command: value.Command,
				VolumeMounts: []corev1.VolumeMount{
					{
						MountPath: "/var/lib/containers",
						Name:      "varlibcontainers",
					},
					{
						MountPath: "/gen/source",
						Name:      "gen-source",
					},
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &truePr,
				},
				Command: value.Command,
			},
		)
	}

}
*/

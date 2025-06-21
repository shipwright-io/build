variable "IMAGE" {
}

variable "NAMESPACE" {
}

variable "TAG" {
	default = "ubi10"
}

variable "DOCKERFILE" {
	default = "Dockerfile"
}

variable "BUILD_IMAGE" {
	default = "registry.access.redhat.com/ubi10-minimal:latest"
}
group "default" {
	targets = ["all"]
}

target "all" {
	args = {
		BASE = "ghcr.io/${NAMESPACE}/base-base:${TAG}"
		BUILD_IMAGE = "${BUILD_IMAGE}"
	}
	tags = ["${IMAGE}:${TAG}"]
	dockerfile = "${DOCKERFILE}"
	platforms = ["linux/amd64", "linux/arm64", "linux/ppc64le", "linux/s390x"]
}



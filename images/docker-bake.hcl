variable "IMAGE" {
}

variable "NAMESPACE" {
}

group "default" {
	targets = ["all"]
}

target "all" {
	args = {
		BASE = "ghcr.io/${NAMESPACE}/base-base"
	}
	tags = ["${IMAGE}:latest"]
	platforms = ["linux/amd64", "linux/arm64", "linux/ppc64le", "linux/s390x"]
}



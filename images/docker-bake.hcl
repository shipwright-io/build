variable "IMAGE" {
}

group "default" {
	targets = ["all"]
}

target "all" {
	tags = ["${IMAGE}:latest"]
	platforms = ["linux/amd64", "linux/arm64", "linux/ppc64le", "linux/s390x"]
}

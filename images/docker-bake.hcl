variable "IMAGE" {
}

variable "NAMESPACE" {
}
variable "UBI" {
}

variable "DOCKERFILE" {
}
group "default" {
	targets = ["all"]
}

target "all" {
	args = {
		BASE = "ghcr.io/${NAMESPACE}/base-base"
	}
	dockerfile = DOCKERFILE
	tags = ["${IMAGE}:${UBI}-latest"]
	platforms = ["linux/amd64", "linux/arm64", "linux/ppc64le", "linux/s390x"]
}



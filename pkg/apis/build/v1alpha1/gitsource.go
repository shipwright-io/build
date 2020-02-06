package v1alpha1

// GitSource contains the versioned source code metadata
// This is similar to OpenShift BuildConfig Git Source API
type GitSource struct {

	// URL of the git repo
	URL string `json:"url"`

	// Ref is a git reference. Optional. "master" is used by default.
	Ref string `json:"ref,omitempty"`

	// ContextDir is a path to subfolder in the repo. Optional.
	ContextDir string `json:"contextDir,omitempty"`

	// HttpProxy is optional.
	HttpProxy string `json:"httpProxy,omitempty"`

	// HttpsProxy is optional.
	HttpsProxy string `json:"httpsProxy,omitempty"`

	// NoProxy can be used to specify domains for which no proxying should be performed. Optional.
	NoProxy string `json:"noProxy,omitempty"`

	// SecretRef refers to the secret that contains credentials to access the git repo. Optional.
	SecretRef *SecretRef `json:"secretRef,omitempty"`

	// Flavor of the git provider like github, gitlab, bitbucket, generic, etc. Optional.
	Flavor string `json:"flavor,omitempty"`
}

// SecretRef holds information about the secret that contains credentials to access the git repo
type SecretRef struct {
	// Name is the name of the secret that contains credentials to access the git repo
	Name string `json:"name"`
}

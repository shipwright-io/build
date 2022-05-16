package tektonrun

// ExtraFields carry on metainformation to link a given Tekton Run object with Shipwright.
type ExtraFields struct {
	BuildRunName string `json:"buildRunName,omitempty"`
}

// IsEmpty checks if the BuildRunName is defined.
func (s *ExtraFields) IsEmpty() bool {
	return s.BuildRunName == ""
}

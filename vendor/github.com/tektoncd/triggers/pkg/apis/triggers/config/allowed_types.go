package config

import (
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var Decoder runtime.Decoder

// TODO(dibyom): We should have a way of configuring this instead of an init function?
func init() {
	scheme := runtime.NewScheme()
	utilruntime.Must(pipelinev1alpha1.AddToScheme(scheme))
	utilruntime.Must(pipelinev1beta1.AddToScheme(scheme))
	codec := serializer.NewCodecFactory(scheme)
	Decoder = codec.UniversalDecoder(
		pipelinev1alpha1.SchemeGroupVersion,
		pipelinev1beta1.SchemeGroupVersion,
	)
}

// EnsureAllowedType returns nil if the resourceTemplate has an apiVersion
// and kind field set to one of the allowed ones.
func EnsureAllowedType(rt runtime.RawExtension) error {
	_, err := runtime.Decode(Decoder, rt.Raw)
	return err
}

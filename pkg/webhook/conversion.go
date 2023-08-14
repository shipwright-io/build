// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package conversion

/*
* This code is influenced by the conversion webhook example
* tested in the Kubernetes E2E(see https://github.com/kubernetes/kubernetes/tree/v1.25.3/test/images/agnhost/crd-conversion-webhook/converter),
* as mentioned in the Kubernetes official documentation: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#write-a-conversion-webhook-server
 */

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/munnerz/goautoneg"
	"github.com/shipwright-io/build/pkg/ctxlog"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var scheme = runtime.NewScheme()

func init() {
	addToScheme(scheme)
}

func addToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(v1.AddToScheme(scheme))
	utilruntime.Must(beta1.AddToScheme(scheme))

}

var serializers = map[mediaType]runtime.Serializer{
	{"application", "json"}: json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{Pretty: false}),
	{"application", "yaml"}: json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{Yaml: true}),
}

type mediaType struct {
	Type, SubType string
}

// convertFunc serves as the Custom Resource conversiob function
type convertFunc func(Object *unstructured.Unstructured, version string, ctx context.Context) (*unstructured.Unstructured, metav1.Status)

// CRDConvertHandler is a handle func for the /convert endpoint
func CRDConvertHandler(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		CRDConvert(w, r, ctx)
	}
}

// CRDConvert serves the /convert endpoint by passing an additional argument(ctx)
func CRDConvert(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	serve(w, r, convertSHPCR, ctx)
}

// serve handles a ConversionReview object type, it will process a ConversionRequest object
// and convert that into a ConversionResponse one.
func serve(w http.ResponseWriter, r *http.Request, convert convertFunc, ctx context.Context) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	contentType := r.Header.Get("Content-Type")
	serializer := getInputSerializer(contentType)
	if serializer == nil {
		msg := fmt.Sprintf("invalid Content-Type header `%s`", contentType)
		ctxlog.Error(ctx, errors.New(msg), "invalid header")
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	ctxlog.Info(ctx, "handling request")

	obj, gvk, err := serializer.Decode(body, nil, nil)
	if err != nil {
		msg := fmt.Sprintf("failed to deserialize body (%v) with error %v", string(body), err)
		ctxlog.Error(ctx, errors.New(msg), "failed to deserialize")
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var responseObj runtime.Object

	switch *gvk {
	case v1.SchemeGroupVersion.WithKind("ConversionReview"):
		convertReview, ok := obj.(*v1.ConversionReview)
		if !ok {
			msg := fmt.Sprintf("Expected v1beta1.ConversionReview but got: %T", obj)
			ctxlog.Error(ctx, errors.New(msg), msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		convertReview.Response = doConversion(convertReview.Request, convert, ctx)
		convertReview.Response.UID = convertReview.Request.UID
		ctxlog.Info(ctx, fmt.Sprintf("sending response: %v", convertReview.Response))

		convertReview.Request = &v1.ConversionRequest{}
		responseObj = convertReview
	default:
		msg := fmt.Sprintf("Unsupported group version kind: %v", gvk)
		ctxlog.Error(ctx, errors.New(msg), msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	accept := r.Header.Get("Accept")
	outSerializer := getOutputSerializer(accept)
	if outSerializer == nil {
		msg := fmt.Sprintf("invalid accept header `%s`", accept)
		ctxlog.Error(ctx, errors.New(msg), msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	err = outSerializer.Encode(responseObj, w)
	if err != nil {
		ctxlog.Error(ctx, err, "outserializer enconding failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getInputSerializer(contentType string) runtime.Serializer {
	parts := strings.SplitN(contentType, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	return serializers[mediaType{parts[0], parts[1]}]
}

func getOutputSerializer(accept string) runtime.Serializer {
	if len(accept) == 0 {
		return serializers[mediaType{"application", "json"}]
	}
	clauses := goautoneg.ParseAccept(accept)
	for _, clause := range clauses {
		for k, v := range serializers {
			switch {
			case clause.Type == k.Type && clause.SubType == k.SubType,
				clause.Type == k.Type && clause.SubType == "*",
				clause.Type == "*" && clause.SubType == "*":
				return v
			}
		}
	}
	return nil
}

// doConversion takes a CR in the v1 ConversionRequest using the convert function
// and returns a ConversionResponse with a CR
func doConversion(convertRequest *v1.ConversionRequest, convert convertFunc, ctx context.Context) *v1.ConversionResponse {
	var convertedObjects []runtime.RawExtension
	for _, obj := range convertRequest.Objects {
		cr := unstructured.Unstructured{}
		if err := cr.UnmarshalJSON(obj.Raw); err != nil {
			ctxlog.Error(ctx, err, "unmarshalling json on convertrequest")
			return &v1.ConversionResponse{
				Result: metav1.Status{
					Message: fmt.Sprintf("failed to unmarshall object (%v) with error: %v", string(obj.Raw), err),
					Status:  metav1.StatusFailure,
				},
			}
		}
		convertedCR, status := convert(&cr, convertRequest.DesiredAPIVersion, ctx)
		if status.Status != metav1.StatusSuccess {
			ctxlog.Error(ctx, errors.New(status.String()), "status is not Success")
			return &v1.ConversionResponse{
				Result: status,
			}
		}
		convertedCR.SetAPIVersion(convertRequest.DesiredAPIVersion)
		convertedObjects = append(convertedObjects, runtime.RawExtension{Object: convertedCR})
	}
	return &v1.ConversionResponse{
		ConvertedObjects: convertedObjects,
		Result:           statusSucceed(),
	}
}

func statusSucceed() metav1.Status {
	return metav1.Status{
		Status: metav1.StatusSuccess,
	}
}

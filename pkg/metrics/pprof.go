// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

// +build pprof_enabled

package metrics

import "net/http/pprof"

func init() {
	// Extra handlers based on https://github.com/golang/go/blob/master/src/net/http/pprof/pprof.go#L80-L86
	metricsExtraHandlers["/debug/pprof/"] = pprof.Index
	metricsExtraHandlers["/debug/pprof/cmdline"] = pprof.Cmdline
	metricsExtraHandlers["/debug/pprof/profile"] = pprof.Profile
	metricsExtraHandlers["/debug/pprof/symbol"] = pprof.Symbol
	metricsExtraHandlers["/debug/pprof/trace"] = pprof.Trace
}

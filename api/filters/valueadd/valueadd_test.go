// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package valueadd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	filtertest_test "sigs.k8s.io/kustomize/api/testutils/filtertest"
)

const someResource = `
kind: SomeKind
spec:
  resourceRef:
    external: projects/whatever
`

func TestValueAddFilter(t *testing.T) {
	testCases := map[string]struct {
		input          string
		expectedOutput string
		filter         Filter
	}{
		"simpleAdd": {
			input: `
kind: SomeKind
`,
			expectedOutput: `
kind: SomeKind
spec:
  resourceRef:
    external: valueAdded
`,
			filter: Filter{
				Value:     "valueAdded",
				FieldPath: "spec/resourceRef/external",
			},
		},
		"addToList": {
			input: `
kind: VirtualService
metadata:
  name: service # {"$kpt-set":"name"}
spec:
  hosts:
  - service.dev.acme.io # {"$kpt-set":"hosts"}
  gateways:
  - gateway.istio-system.svc.cluster.local # {"$kpt-set":"istio-gateway"}
  http:
  - name: service # {"$kpt-set":"name"}
    route:
    - destination:
        host: service # {"$kpt-set":"name"}
        port:
          number: 80
`,
			expectedOutput: `
kind: VirtualService
metadata:
  name: service # {"$kpt-set":"name"}
spec:
  hosts:
  - service.dev.acme.io # {"$kpt-set":"hosts"}
  gateways:
  - gateway.istio-system.svc.cluster.local # {"$kpt-set":"istio-gateway"}
  http:
  - name: service # {"$kpt-set":"name"}
    route:
    - destination:
        host: service # {"$kpt-set":"name"}
        port:
		  number: 80
	corsPolicy:
		allowOrigins:
		  - regex: https://ui.dev.acme.io
		allowMethods:
		  - POST
		  - GET
		  - OPTIONS
		  - HEAD
		  - PUT
		  - DELETE
		allowHeaders:
		  - "*"
		allowCredentials: true
		maxAge: "24h"
`,
			filter: Filter{
				Value: `
				allowOrigins:
				  - regex: https://ui.dev.acme.io
				allowMethods:
				  - POST
				  - GET
				  - OPTIONS
				  - HEAD
				  - PUT
				  - DELETE
				allowHeaders:
				  - "*"
				allowCredentials: true
				maxAge: "24h"
					`,
				FieldPath: "spec/http/0/corsPolicy",
			},
		},
		"replaceExisting": {
			input: someResource,
			expectedOutput: `
kind: SomeKind
spec:
  resourceRef:
    external: valueAdded
`,
			filter: Filter{
				Value:     "valueAdded",
				FieldPath: "spec/resourceRef/external",
			},
		},
		"prefixExisting": {
			input: someResource,
			expectedOutput: `
kind: SomeKind
spec:
  resourceRef:
    external: valueAdded/projects/whatever
`,
			filter: Filter{
				Value:            "valueAdded",
				FieldPath:        "spec/resourceRef/external",
				FilePathPosition: 1,
			},
		},
		"postfixExisting": {
			input: someResource,
			expectedOutput: `
kind: SomeKind
spec:
  resourceRef:
    external: projects/whatever/valueAdded
`,
			filter: Filter{
				Value:            "valueAdded",
				FieldPath:        "spec/resourceRef/external",
				FilePathPosition: 99,
			},
		},
		"placeInMiddleOfExisting": {
			input: someResource,
			expectedOutput: `
kind: SomeKind
spec:
  resourceRef:
    external: projects/valueAdded/whatever
`,
			filter: Filter{
				Value:            "valueAdded",
				FieldPath:        "spec/resourceRef/external",
				FilePathPosition: 2,
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			filter := tc.filter
			if !assert.Equal(t,
				strings.TrimSpace(tc.expectedOutput),
				strings.TrimSpace(filtertest_test.RunFilter(t, tc.input, filter))) {
				t.FailNow()
			}
		})
	}
}

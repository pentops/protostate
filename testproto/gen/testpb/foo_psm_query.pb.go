// Code generated by protoc-gen-go-psm. DO NOT EDIT.

package testpb

import (
	psm "github.com/pentops/protostate/psm"
)

// State Query Service for %sFoo
// QuerySet is the query set for the Foo service.

type FooPSMQuerySet = psm.StateQuerySet[
	*GetFooRequest,
	*GetFooResponse,
	*ListFoosRequest,
	*ListFoosResponse,
	*ListFooEventsRequest,
	*ListFooEventsResponse,
]

func NewFooPSMQuerySet(
	smSpec psm.QuerySpec[
		*GetFooRequest,
		*GetFooResponse,
		*ListFoosRequest,
		*ListFoosResponse,
		*ListFooEventsRequest,
		*ListFooEventsResponse,
	],
	options psm.StateQueryOptions,
) (*FooPSMQuerySet, error) {
	return psm.BuildStateQuerySet[
		*GetFooRequest,
		*GetFooResponse,
		*ListFoosRequest,
		*ListFoosResponse,
		*ListFooEventsRequest,
		*ListFooEventsResponse,
	](smSpec, options)
}

type FooPSMQuerySpec = psm.QuerySpec[
	*GetFooRequest,
	*GetFooResponse,
	*ListFoosRequest,
	*ListFoosResponse,
	*ListFooEventsRequest,
	*ListFooEventsResponse,
]

func DefaultFooPSMQuerySpec(tableSpec psm.QueryTableSpec) FooPSMQuerySpec {
	return psm.QuerySpec[
		*GetFooRequest,
		*GetFooResponse,
		*ListFoosRequest,
		*ListFoosResponse,
		*ListFooEventsRequest,
		*ListFooEventsResponse,
	]{
		QueryTableSpec: tableSpec,
		ListRequestFilter: func(req *ListFoosRequest) (map[string]interface{}, error) {
			filter := map[string]interface{}{}
			if req.TenantId != nil {
				filter["tenant_id"] = *req.TenantId
			}
			return filter, nil
		},
		ListEventsRequestFilter: func(req *ListFooEventsRequest) (map[string]interface{}, error) {
			filter := map[string]interface{}{}
			filter["foo_id"] = req.FooId
			return filter, nil
		},
	}
}

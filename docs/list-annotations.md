# List Annotations

A 'Listify' endpoint is configured through annotations and patterns on the request and
response messages for an RPC endpoint.

For an RPC to be valid as a list endpoint, the following must be true:

- The Request message must contain:
  - a field of type `psm.list.v1.PageRequest` with any name, but suggest 'page'
  - a field of type `psm.list.v1.QueryRequest` with any name, but suggest 'query'

- The Response message must contain:
  - a field of type `psm.list.v1.PageResponse`, with any name, but suggest
	'page'
  - a field of any Message type, with the 'repeated' flag set, this defines the
	response type for the endpoint, and defines the available sort and filter
	fields which can be requested in the `psm.list.v1.QueryRequest` in the
	Request field.
- There must be at least one sort field or sort tiebreaker available

## Sort Tie Breaker

An annotation on a Request message can specify a field to use at the end of any
sort query to ensure the results are always unique (and therefore consistently
sorted).

To specify a tie breaker, add the psm.list.v1.list_request.sort_tiebreaker
field, e.g.

```proto
message ListBarsRequest {
  psm.list.v1.PageRequest page = 1;
  psm.list.v1.QueryRequest query = 2;
  option (psm.list.v1.list_request) = {
    sort_tiebreaker: ["bar_id"]
  };
}
```

The field is specified using dot notation and the proto field names,
`foo.foo_id` means that the response type message must have a field named `foo`,
which is a Message type, and that message must have a field `foo_id` which is
unique across all *Bar* messages.


```proto
message ListBarsRequest {
  psm.list.v1.PageRequest page = 1;
  psm.list.v1.QueryRequest query = 2;
  option (psm.list.v1.list_request) = {
    sort_tiebreaker: ["foo.foo_id"]
  };
}

message ListBarsResponse {
  repeated Bar bars = 1;
  psm.list.v1.PageResponse page = 2;
}

message Bar {
	Foo foo = 1;
}

message Foo {
	string foo_id = 1;
}
```

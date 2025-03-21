module github.com/pentops/protostate

go 1.24.0

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.5-20250219170025-d39267d9df8f.1
	buf.build/go/protoyaml v0.3.1
	github.com/bufbuild/protovalidate-go v0.9.2
	github.com/elgris/sqrl v0.0.0-20210727210741-7e0198b30236
	github.com/google/uuid v1.6.0
	github.com/iancoleman/strcase v0.3.0
	github.com/lib/pq v1.10.9
	github.com/pentops/flowtest v0.0.0-20241110231021-42663ac00b63
	github.com/pentops/j5 v0.0.0-20250220215902-96e4ad10ce0b
	github.com/pentops/log.go v0.0.15
	github.com/pentops/o5-messaging v0.0.0-20250222021813-347361ac64f8
	github.com/pentops/pgtest.go v0.0.0-20241223222214-7638cc50e15b
	github.com/pentops/sqrlx.go v0.0.0-20250107233303-36a524e154c7
	github.com/pressly/goose v2.7.0+incompatible
	github.com/stretchr/testify v1.10.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250303144028-a0af3efb3deb
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/utils v0.0.0-20241210054802-24370beab758
)

require (
	cel.dev/expr v0.21.2 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/bufbuild/protocompile v0.14.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.24.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/jhump/protoreflect v1.17.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	golang.org/x/exp v0.0.0-20250228200357-dead58393ab7 // indirect
	golang.org/x/net v0.36.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250303144028-a0af3efb3deb // indirect
)

replace github.com/pentops/j5 => /Users/daemonl/pentops/j5

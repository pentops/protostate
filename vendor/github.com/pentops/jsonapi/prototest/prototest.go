package prototest

// Package prototest provides utilities for dynamically parsing proto files
// into reflection for test cases

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type ResultSet struct {
	messages map[protoreflect.FullName]protoreflect.MessageDescriptor
	services map[protoreflect.FullName]protoreflect.ServiceDescriptor
}

func (rs ResultSet) MessageByName(t testing.TB, name protoreflect.FullName) protoreflect.MessageDescriptor {
	t.Helper()

	md, ok := rs.messages[name]
	if !ok {
		t.Fatalf("message not found: %s", name)
	}
	return md
}

func (rs ResultSet) ServiceByName(t testing.TB, name protoreflect.FullName) protoreflect.ServiceDescriptor {
	t.Helper()
	md, ok := rs.services[name]
	if !ok {
		t.Fatalf("service not found: %s", name)
	}
	return md
}

func DescriptorsFromSource(t testing.TB, source map[string]string) *ResultSet {
	t.Helper()

	allFiles := make([]string, 0, len(source))
	for filename := range source {
		allFiles = append(allFiles, filename)
	}

	parser := protoparse.Parser{
		ImportPaths:           []string{""},
		IncludeSourceCodeInfo: false,

		Accessor: func(filename string) (io.ReadCloser, error) {
			src, ok := source[filename]
			if !ok {
				return nil, fmt.Errorf("file not found: %s", filename)
			}
			return io.NopCloser(strings.NewReader(src)), nil
		},
	}

	customDesc, err := parser.ParseFilesButDoNotLink(allFiles...)
	if err != nil {
		t.Fatal(err)
	}

	rs := &ResultSet{
		messages: make(map[protoreflect.FullName]protoreflect.MessageDescriptor),
		services: make(map[protoreflect.FullName]protoreflect.ServiceDescriptor),
	}

	for _, file := range customDesc {
		fd, err := protodesc.NewFile(file, protoregistry.GlobalFiles)
		if err != nil {
			t.Fatal(err)
		}

		messages := fd.Messages()
		for i := 0; i < messages.Len(); i++ {
			msg := messages.Get(i)
			rs.messages[msg.FullName()] = msg
		}

		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			rs.services[svc.FullName()] = svc
		}
	}

	return rs
}

type MessageOption func(*messageOption)

type messageOption struct {
	name    string
	imports []string
}

func WithMessageName(name string) MessageOption {
	return func(o *messageOption) {
		o.name = name
	}
}

func WithMessageImports(imports ...string) MessageOption {
	return func(o *messageOption) {
		o.imports = append(o.imports, imports...)
	}
}

func SingleMessage(t testing.TB, content ...interface{}) protoreflect.MessageDescriptor {
	t.Helper()
	options := &messageOption{
		name: "Wrapper",
	}
	lines := make([]string, 0, len(content))
	for _, c := range content {
		if opt, ok := c.(MessageOption); ok {
			opt(options)
			continue
		}
		if str, ok := c.(string); ok {
			lines = append(lines, str)
			continue
		}
		t.Fatalf("unknown content type: %T", c)
	}

	importLines := make([]string, 0, len(options.imports))
	for _, imp := range options.imports {
		importLines = append(importLines, fmt.Sprintf(`import "%s";`, imp))
	}

	rs := DescriptorsFromSource(t, map[string]string{
		"test.proto": fmt.Sprintf(`
		syntax = "proto3";
		%s
		package test;
		message %s {
			%s
		}
		`, strings.Join(importLines, "\n"), options.name, strings.Join(lines, "\n")),
	})

	return rs.MessageByName(t, protoreflect.FullName("test."+options.name))
}

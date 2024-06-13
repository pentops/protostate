package main

import (
	"context"
	"fmt"

	"github.com/pentops/protostate/internal/pgstore/pgmigrate"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/prototools/protosrc"
	"github.com/pentops/runner/commander"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

var Version = "dev"

func main() {
	cmdGroup := commander.NewCommandSet()

	cmdGroup.Add("migrate", commander.NewCommand(runFmt))
	cmdGroup.RunMain("psmtool", Version)
}

func runFmt(ctx context.Context, cfg struct {
	SourceDir string   `flag:"src"`
	Package   string   `flag:"package"`
	Machines  []string `flag:"machines"`
}) error {
	img, err := protosrc.ReadImageFromSourceDir(ctx, cfg.SourceDir)
	if err != nil {
		return fmt.Errorf("reading source %s: %w", cfg.SourceDir, err)
	}

	descriptors, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{
		File: append(img.Files, img.Dependencies...),
	})
	if err != nil {
		return err
	}

	specs := make([]psm.QueryTableSpec, 0, len(cfg.Machines))

	for _, machine := range cfg.Machines {

		stateName := fmt.Sprintf("%s.%sState", cfg.Package, machine)
		stateMsgI, err := descriptors.FindDescriptorByName(protoreflect.FullName(stateName))
		if err != nil {
			return fmt.Errorf("message %s: %w", stateName, err)
		}
		stateMsg, ok := stateMsgI.(protoreflect.MessageDescriptor)
		if !ok {
			return fmt.Errorf("message %s is not a message", stateName)
		}

		eventName := fmt.Sprintf("%s.%sEvent", cfg.Package, machine)
		eventMsgI, err := descriptors.FindDescriptorByName(protoreflect.FullName(eventName))
		if err != nil {
			return fmt.Errorf("message %s: %w", eventName, err)
		}
		eventMsg, ok := eventMsgI.(protoreflect.MessageDescriptor)
		if !ok {
			return fmt.Errorf("message %s is not a message", eventName)
		}

		spec, err := psm.BuildQueryTableSpec(stateMsg, eventMsg)
		if err != nil {
			return fmt.Errorf("table map for %s: %w", machine, err)
		}

		specs = append(specs, spec)
	}

	migrationFile, err := pgmigrate.BuildStateMachineMigrations(specs...)
	if err != nil {
		return fmt.Errorf("build migration file: %w", err)
	}

	fmt.Println(string(migrationFile))

	return nil
}

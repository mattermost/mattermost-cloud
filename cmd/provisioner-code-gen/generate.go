// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"bytes"
	"fmt"
	"go/types"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Runs specified code generators for specified types",
		// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg, names, structTypes, err := loadTypes(cmd)
			if err != nil {
				return errors.Wrap(err, "failed to load package and types")
			}

			codeBuffer, err := setupCodeBuffer(cmd, pkg.Name)
			if err != nil {
				return errors.Wrap(err, "field to setup code buffer")
			}

			generators, _ := cmd.Flags().GetStringSlice("generator")
			if len(generators) == 0 {
				return errors.New("no generators specified")
			}

			for i, generator := range generators {
				gen := strings.ToLower(generator)
				generators[i] = gen
				if _, found := generatorsMapping[gen]; !found {
					return errors.Errorf("generator %q not found", generator)
				}
			}

			for i, structType := range structTypes {
				runGeneratorsForType(names[i], structType, codeBuffer, generators)
			}

			return printOutput(cmd, codeBuffer.Bytes(), pkg.Name)
		},
	}

	cmd.Flags().StringSlice("generator", []string{}, "Generators to run for specified types.")
	cmd.Flags().StringSlice("type", []string{}, "Types for which to run code generation.")

	return cmd
}

func runGeneratorsForType(sourceTypeName string, structType *types.Struct, codeBuffer *bytes.Buffer, generators []string) {
	for _, generator := range generators {
		receiverName := strings.ToLower(sourceTypeName)[:1]
		codeBuffer.WriteString(generatorsMapping[generator](receiverName, sourceTypeName, structType))
	}
}

// GeneratorFunc is signature for generator function.
type GeneratorFunc func(receiver, typeName string, structType *types.Struct) string

var generatorsMapping = map[string]GeneratorFunc{
	"get_id":       generateGetID,
	"get_state":    generateGetState,
	"is_deleted":   generateIsDeleted,
	"as_resources": generateAsResources,
}

func generateGetID(receiverName, sourceTypeName string, _ *types.Struct) string {
	return fmt.Sprintf(`
// GetID returns ID of the resource.
func (%s *%s) GetID() string {
	return %s.ID
}
`, receiverName, sourceTypeName, receiverName)
}

func generateGetState(receiverName, sourceTypeName string, _ *types.Struct) string {
	return fmt.Sprintf(`
// GetState returns State of the resource.
func (%s *%s) GetState() string {
	return string(%s.State)
}
`, receiverName, sourceTypeName, receiverName)
}

func generateIsDeleted(receiverName, sourceTypeName string, _ *types.Struct) string {
	return fmt.Sprintf(`
// IsDeleted determines whether the resource is deleted.
func (%s *%s) IsDeleted() bool {
	return %s.DeleteAt > 0
}
`, receiverName, sourceTypeName, receiverName)
}

func generateAsResources(_, sourceTypeName string, _ *types.Struct) string {
	return fmt.Sprintf(`
// %ssAsResources returns collection as Resource objects.
func %ssAsResources(collection []*%s) []Resource {
	resources := make([]Resource, 0, len(collection))
	for _, elem := range collection {
		resources = append(resources, Resource(elem))
	}
	return resources
}
`, sourceTypeName, sourceTypeName, sourceTypeName)
}

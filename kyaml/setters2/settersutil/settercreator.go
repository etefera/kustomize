// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package settersutil

import (
	"fmt"
	"strings"

	"github.com/go-openapi/spec"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/setters2"
)

// SetterCreator creates or updates a setter in the OpenAPI definitions, and inserts references
// to the setter from matching resource fields.
type SetterCreator struct {
	// Name is the name of the setter to create or update.
	Name string

	SetBy string

	Description string

	Type string

	Schema string

	// FieldName if set will add the OpenAPI reference to fields with this name or path
	// FieldName may be the full name of the field, full path to the field, or the path suffix.
	// e.g. all of the following would match spec.template.spec.containers.image --
	// [image, containers.image, spec.containers.image, template.spec.containers.image,
	//  spec.template.spec.containers.image]
	// Optional.  If unspecified match all field names.
	FieldName string

	// FieldValue if set will add the OpenAPI reference to fields if they have this value.
	// Optional.  If unspecified match all field values.
	FieldValue string

	// Required indicates that the setter must be set by package consumer before
	// live apply/preview. This field is added to the setter definition to record
	// the package publisher's intent to make the setter required to be set.
	Required bool
}

func (c SetterCreator) Create(openAPIPath, resourcesPath string) error {
	err := validateSchema(c.Schema)
	if err != nil {
		return errors.Errorf("invalid schema: %v", err)
	}

	// Update the resources with the setter reference
	inout := &kio.LocalPackageReadWriter{PackagePath: resourcesPath}
	a := &setters2.Add{
		FieldName:  c.FieldName,
		FieldValue: c.FieldValue,
		Ref:        fieldmeta.DefinitionsPrefix + fieldmeta.SetterDefinitionPrefix + c.Name,
		Type:       c.Type,
	}
	err = kio.Pipeline{
		Inputs:  []kio.Reader{inout},
		Filters: []kio.Filter{kio.FilterAll(a)},
		Outputs: []kio.Writer{inout},
	}.Execute()
	if a.Count == 0 {
		fmt.Printf("setter %s doesn't match any field in resources, but creating setter definition\n", c.Name)
	}
	if err != nil {
		return err
	}

	// Update the OpenAPI definitions to hace the setter
	sd := setters2.SetterDefinition{
		Name: c.Name, Value: c.FieldValue, SetBy: c.SetBy, Description: c.Description,
		Type: c.Type, Schema: c.Schema, Required: c.Required,
	}
	if err := sd.AddToFile(openAPIPath); err != nil {
		return err
	}

	// Load the updated definitions
	if err := openapi.AddSchemaFromFile(openAPIPath); err != nil {
		return err
	}
	// if the setter is of array type write the derived list values back to
	// openAPI definitions
	if len(a.ListValues) > 0 {
		sd.ListValues = a.ListValues
		sd.Value = ""
		if err := sd.AddToFile(openAPIPath); err != nil {
			return err
		}
	}

	return nil
}

// The types recognized by by the go openapi validation library:
// https://github.com/go-openapi/validate/blob/master/helpers.go#L35
var validTypeValues = []string{
	"object", "array", "string", "integer", "number", "boolean", "file", "null",
}

// validateSchema parses the provided schema and validates it.
func validateSchema(schema string) error {
	var sc spec.Schema
	err := sc.UnmarshalJSON([]byte(schema))
	if err != nil {
		return errors.Errorf("unable to parse schema: %v", err)
	}
	return validateSchemaTypes(&sc)
}

// validateSchemaTypes traverses the schema and checks that only valid types
// are used.
func validateSchemaTypes(sc *spec.Schema) error {
	if len(sc.Type) > 1 {
		return errors.Errorf("only one type is supported: %s", strings.Join(sc.Type, ", "))
	}

	if len(sc.Type) == 1 {
		t := sc.Type[0]
		var match bool
		for _, validType := range validTypeValues {
			if t == validType {
				match = true
			}
		}
		if !match {
			return errors.Errorf("type %q is not supported. Must be one of: %s",
				t, strings.Join(validTypeValues, ", "))
		}
	}

	if items := sc.Items; items != nil {
		if items.Schema != nil {
			err := validateSchemaTypes(items.Schema)
			if err != nil {
				return err
			}
		}
		for i := range items.Schemas {
			schema := items.Schemas[i]
			err := validateSchemaTypes(&schema)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

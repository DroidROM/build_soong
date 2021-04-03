// Copyright 2020 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bazel

import (
	"fmt"
	"regexp"
	"sort"
)

// BazelTargetModuleProperties contain properties and metadata used for
// Blueprint to BUILD file conversion.
type BazelTargetModuleProperties struct {
	// The Bazel rule class for this target.
	Rule_class string `blueprint:"mutated"`

	// The target label for the bzl file containing the definition of the rule class.
	Bzl_load_location string `blueprint:"mutated"`
}

const BazelTargetModuleNamePrefix = "__bp2build__"

var productVariableSubstitutionPattern = regexp.MustCompile("%(d|s)")

// Label is used to represent a Bazel compatible Label. Also stores the original bp text to support
// string replacement.
type Label struct {
	Bp_text string
	Label   string
}

// LabelList is used to represent a list of Bazel labels.
type LabelList struct {
	Includes []Label
	Excludes []Label
}

// Append appends the fields of other labelList to the corresponding fields of ll.
func (ll *LabelList) Append(other LabelList) {
	if len(ll.Includes) > 0 || len(other.Includes) > 0 {
		ll.Includes = append(ll.Includes, other.Includes...)
	}
	if len(ll.Excludes) > 0 || len(other.Excludes) > 0 {
		ll.Excludes = append(other.Excludes, other.Excludes...)
	}
}

func UniqueBazelLabels(originalLabels []Label) []Label {
	uniqueLabelsSet := make(map[Label]bool)
	for _, l := range originalLabels {
		uniqueLabelsSet[l] = true
	}
	var uniqueLabels []Label
	for l, _ := range uniqueLabelsSet {
		uniqueLabels = append(uniqueLabels, l)
	}
	sort.SliceStable(uniqueLabels, func(i, j int) bool {
		return uniqueLabels[i].Label < uniqueLabels[j].Label
	})
	return uniqueLabels
}

func UniqueBazelLabelList(originalLabelList LabelList) LabelList {
	var uniqueLabelList LabelList
	uniqueLabelList.Includes = UniqueBazelLabels(originalLabelList.Includes)
	uniqueLabelList.Excludes = UniqueBazelLabels(originalLabelList.Excludes)
	return uniqueLabelList
}

const (
	ARCH_X86    = "x86"
	ARCH_X86_64 = "x86_64"
	ARCH_ARM    = "arm"
	ARCH_ARM64  = "arm64"
)

var (
	// This is the list of architectures with a Bazel config_setting and
	// constraint value equivalent. is actually android.ArchTypeList, but the
	// android package depends on the bazel package, so a cyclic dependency
	// prevents using that here.
	selectableArchs = []string{ARCH_X86, ARCH_X86_64, ARCH_ARM, ARCH_ARM64}
)

// Arch-specific label_list typed Bazel attribute values. This should correspond
// to the types of architectures supported for compilation in arch.go.
type labelListArchValues struct {
	X86    LabelList
	X86_64 LabelList
	Arm    LabelList
	Arm64  LabelList
	// TODO(b/181299724): this is currently missing the "common" arch, which
	// doesn't have an equivalent platform() definition yet.
}

// LabelListAttribute is used to represent a list of Bazel labels as an
// attribute.
type LabelListAttribute struct {
	// The non-arch specific attribute label list Value. Required.
	Value LabelList

	// The arch-specific attribute label list values. Optional. If used, these
	// are generated in a select statement and appended to the non-arch specific
	// label list Value.
	ArchValues labelListArchValues
}

// MakeLabelListAttribute initializes a LabelListAttribute with the non-arch specific value.
func MakeLabelListAttribute(value LabelList) LabelListAttribute {
	return LabelListAttribute{Value: UniqueBazelLabelList(value)}
}

// HasArchSpecificValues returns true if the attribute contains
// architecture-specific label_list values.
func (attrs *LabelListAttribute) HasArchSpecificValues() bool {
	for _, arch := range selectableArchs {
		if len(attrs.GetValueForArch(arch).Includes) > 0 || len(attrs.GetValueForArch(arch).Excludes) > 0 {
			return true
		}
	}
	return false
}

// GetValueForArch returns the label_list attribute value for an architecture.
func (attrs *LabelListAttribute) GetValueForArch(arch string) LabelList {
	switch arch {
	case ARCH_X86:
		return attrs.ArchValues.X86
	case ARCH_X86_64:
		return attrs.ArchValues.X86_64
	case ARCH_ARM:
		return attrs.ArchValues.Arm
	case ARCH_ARM64:
		return attrs.ArchValues.Arm64
	default:
		panic(fmt.Errorf("Unknown arch: %s", arch))
	}
}

// SetValueForArch sets the label_list attribute value for an architecture.
func (attrs *LabelListAttribute) SetValueForArch(arch string, value LabelList) {
	switch arch {
	case "x86":
		attrs.ArchValues.X86 = value
	case "x86_64":
		attrs.ArchValues.X86_64 = value
	case "arm":
		attrs.ArchValues.Arm = value
	case "arm64":
		attrs.ArchValues.Arm64 = value
	default:
		panic(fmt.Errorf("Unknown arch: %s", arch))
	}
}

// StringListAttribute corresponds to the string_list Bazel attribute type with
// support for additional metadata, like configurations.
type StringListAttribute struct {
	// The base value of the string list attribute.
	Value []string

	// Optional additive set of list values to the base value.
	ArchValues stringListArchValues
}

// Arch-specific string_list typed Bazel attribute values. This should correspond
// to the types of architectures supported for compilation in arch.go.
type stringListArchValues struct {
	X86    []string
	X86_64 []string
	Arm    []string
	Arm64  []string
	// TODO(b/181299724): this is currently missing the "common" arch, which
	// doesn't have an equivalent platform() definition yet.
}

// HasArchSpecificValues returns true if the attribute contains
// architecture-specific string_list values.
func (attrs *StringListAttribute) HasArchSpecificValues() bool {
	for _, arch := range selectableArchs {
		if len(attrs.GetValueForArch(arch)) > 0 {
			return true
		}
	}
	return false
}

// GetValueForArch returns the string_list attribute value for an architecture.
func (attrs *StringListAttribute) GetValueForArch(arch string) []string {
	switch arch {
	case ARCH_X86:
		return attrs.ArchValues.X86
	case ARCH_X86_64:
		return attrs.ArchValues.X86_64
	case ARCH_ARM:
		return attrs.ArchValues.Arm
	case ARCH_ARM64:
		return attrs.ArchValues.Arm64
	default:
		panic(fmt.Errorf("Unknown arch: %s", arch))
	}
}

// SetValueForArch sets the string_list attribute value for an architecture.
func (attrs *StringListAttribute) SetValueForArch(arch string, value []string) {
	switch arch {
	case ARCH_X86:
		attrs.ArchValues.X86 = value
	case ARCH_X86_64:
		attrs.ArchValues.X86_64 = value
	case ARCH_ARM:
		attrs.ArchValues.Arm = value
	case ARCH_ARM64:
		attrs.ArchValues.Arm64 = value
	default:
		panic(fmt.Errorf("Unknown arch: %s", arch))
	}
}

// TryVariableSubstitution, replace string substitution formatting within each string in slice with
// Starlark string.format compatible tag for productVariable.
func TryVariableSubstitutions(slice []string, productVariable string) ([]string, bool) {
	ret := make([]string, 0, len(slice))
	changesMade := false
	for _, s := range slice {
		newS, changed := TryVariableSubstitution(s, productVariable)
		ret = append(ret, newS)
		changesMade = changesMade || changed
	}
	return ret, changesMade
}

// TryVariableSubstitution, replace string substitution formatting within s with Starlark
// string.format compatible tag for productVariable.
func TryVariableSubstitution(s string, productVariable string) (string, bool) {
	sub := productVariableSubstitutionPattern.ReplaceAllString(s, "{"+productVariable+"}")
	return sub, s != sub
}
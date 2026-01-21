package jsontype

import (
	"fmt"
	"strings"
)

// Merger represents aggregated information for a single JSON path.
// Path is the full path (slice of keys). Children map contains sub-path nodes.
type Merger struct {
	// full path of this node (immutable after creation)
	Path []string
	// map( label : set(type) )
	LabeledTypesMap map[string]map[DetectedType]struct{}
	// how much times each type was met
	TypesMap map[DetectedType]struct{}
	// children keyed by the immediate child key (for arrays use "0", "1", etc as keys).
	// if type isn't mixed (for some labels) -- all data is written under the same key ""
	ChildrenMap map[string]*Merger
	// tracks the order in which children were added
	ChildrenKeys []string
}

func NewMerger(path []string) *Merger {
	return &Merger{
		Path:            path,
		LabeledTypesMap: make(map[string]map[DetectedType]struct{}),
		TypesMap:        make(map[DetectedType]struct{}),
		ChildrenMap:     make(map[string]*Merger),
		ChildrenKeys:    make([]string, 0),
	}
}

func (m *Merger) AddTypes(label string, types ...DetectedType) {
	if m.LabeledTypesMap[label] == nil {
		m.LabeledTypesMap[label] = make(map[DetectedType]struct{})
	}
	for _, t := range types {
		if _, exist := m.LabeledTypesMap[label][t]; !exist {
			m.LabeledTypesMap[label][t] = struct{}{}
			m.TypesMap[t] = struct{}{}
		}
	}
}

func (m *Merger) AddChild(key string, label string, child *Merger) *Merger {
	existingChild, exists := m.ChildrenMap[key]
	if !exists {
		m.ChildrenMap[key] = child
		m.ChildrenKeys = append(m.ChildrenKeys, key)
		return child
	}
	// copy types
	typesBuf := make([]DetectedType, len(child.TypesMap))
	i := 0
	for t := range child.TypesMap {
		typesBuf[i] = t
		i++
	}
	existingChild.AddTypes(label, typesBuf...)
	// copy children
	for _, newChildKey := range child.ChildrenKeys {
		existingChild.AddChild(newChildKey, label, child.ChildrenMap[newChildKey])
	}
	return existingChild
}

// ComparePaths shows the difference between two paths
func ComparePaths(path1, path2 []string) string {
	maxLen := max(len(path1), len(path2))

	var sb strings.Builder
	sb.WriteString("Path comparison:\n")

	for i := range maxLen {
		seg1 := "<missing>"
		seg2 := "<missing>"
		match := "✗"

		if i < len(path1) {
			seg1 = fmt.Sprintf("%q", path1[i])
		}
		if i < len(path2) {
			seg2 = fmt.Sprintf("%q", path2[i])
		}
		if i < len(path1) && i < len(path2) && path1[i] == path2[i] {
			match = "✓"
		}

		fmt.Fprintf(&sb, "  [%d] %s %-20s vs %-20s\n", i, match, seg1, seg2)
	}

	return sb.String()
}

// PlanToString converts a MergePlan to a human-readable ASCII tree
// This makes future regressions scream immediately
func PlanToString(plan *MergePlan, indent string, isLast bool) string {
	var sb strings.Builder

	// Draw the tree branch
	prefix := indent
	if indent != "" {
		if isLast {
			prefix = indent + "└─ "
		} else {
			prefix = indent + "├─ "
		}
	}

	// Draw this node
	switch plan.Kind {
	case PlanPrimitive:
		sb.WriteString(prefix + "Primitive\n")

	case PlanArray:
		strategyStr := "Collapse"
		if plan.ArrayStrategy == ArrayKeepIndices {
			strategyStr = "KeepIndices"
		}
		fmt.Fprintf(&sb, "%sArray(%s)\n", prefix, strategyStr)

		// Determine next indent
		nextIndent := indent
		if indent != "" {
			if isLast {
				nextIndent = indent + "   "
			} else {
				nextIndent = indent + "│  "
			}
		}

		// Draw element plan
		if plan.Elem != nil {
			sb.WriteString(PlanToString(plan.Elem, nextIndent, true))
		}

	case PlanObject:
		fmt.Fprintf(&sb, "%sObject{%d fields}\n", prefix, len(plan.Fields))

		// Determine next indent
		nextIndent := indent
		if indent != "" {
			if isLast {
				nextIndent = indent + "   "
			} else {
				nextIndent = indent + "│  "
			}
		}

		// Draw fields in sorted order for consistency
		fieldNames := make([]string, 0, len(plan.Fields))
		for name := range plan.Fields {
			fieldNames = append(fieldNames, name)
		}
		// Simple bubble sort to avoid importing sort
		for i := 0; i < len(fieldNames); i++ {
			for j := i + 1; j < len(fieldNames); j++ {
				if fieldNames[i] > fieldNames[j] {
					fieldNames[i], fieldNames[j] = fieldNames[j], fieldNames[i]
				}
			}
		}

		for i, name := range fieldNames {
			isLastField := i == len(fieldNames)-1
			fieldPrefix := nextIndent
			if isLastField {
				fieldPrefix += "└─ "
			} else {
				fieldPrefix += "├─ "
			}
			fmt.Fprintf(&sb, "%s%s:\n", fieldPrefix, name)

			fieldIndent := nextIndent
			if isLastField {
				fieldIndent += "   "
			} else {
				fieldIndent += "│  "
			}
			sb.WriteString(PlanToString(plan.Fields[name], fieldIndent, true))
		}
	}

	return sb.String()
}

// MergerToString converts a Merger to a human-readable tree
func MergerToString(m *Merger, indent string, isLast bool) string {
	var sb strings.Builder

	// Draw the tree branch
	prefix := indent
	if indent != "" {
		if isLast {
			prefix = indent + "└─ "
		} else {
			prefix = indent + "├─ "
		}
	}

	// Draw path and types
	types := make([]string, 0, len(m.TypesMap))
	for t := range m.TypesMap {
		types = append(types, string(t))
	}

	pathStr := PathToString(m.Path)
	if pathStr == "" {
		pathStr = "[]"
	}
	fmt.Fprintf(&sb, "%s%s => %v\n", prefix, pathStr, types)

	// Determine next indent
	nextIndent := indent
	if indent != "" {
		if isLast {
			nextIndent = indent + "   "
		} else {
			nextIndent = indent + "│  "
		}
	}

	// Draw children
	for i, key := range m.ChildrenKeys {
		isLastChild := i == len(m.ChildrenKeys)-1
		sb.WriteString(MergerToString(m.ChildrenMap[key], nextIndent, isLastChild))
	}

	return sb.String()
}

// SnapshotPlan creates a golden test snapshot of a plan
func SnapshotPlan(name string, field *FieldInfo) string {
	plan := PlanShape(field)
	return fmt.Sprintf("=== Plan Snapshot: %s ===\n%s", name, PlanToString(plan, "", true))
}

// SnapshotMerger creates a golden test snapshot of a merger
func SnapshotMerger(name string, m *Merger) string {
	return fmt.Sprintf("=== Merger Snapshot: %s ===\n%s", name, MergerToString(m, "", true))
}

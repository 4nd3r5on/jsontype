package jsontype

import (
	"fmt"
	"log/slog"
	"strings"
)

// Layer 3: Merge Executor
// Mechanical execution of the plan
// This layer does NOT:
// - inspect FieldInfo types
// - check if containers are mixed
// - infer nullability
// - normalize paths
// It just obeys the plan.

// ExecuteMerge executes a merge plan on the given field infos
// parentPath is the path we're building (with wildcards for collapsed arrays)
// fields are the FieldInfos at this level
func ExecuteMerge(
	plan *MergePlan,
	label string,
	fields []*FieldInfo,
	logger *slog.Logger,
) *Merger {
	if len(fields) == 0 {
		return NewMerger(nil)
	}

	// Start with empty path - we'll build it correctly as we go
	return executeMergeWithPath(plan, label, fields, nil, logger)
}

// executeMergeWithPath does the actual work with explicit path tracking
// This is exported so ObjectMergeStrategy can use it
func executeMergeWithPath(
	plan *MergePlan,
	label string,
	fields []*FieldInfo,
	currentPath []string,
	logger *slog.Logger,
) *Merger {
	if len(fields) == 0 {
		return NewMerger(currentPath)
	}

	logger.Debug("executing merge with path",
		"currentPath", PathToString(currentPath),
		"planKind", plan.Kind,
		"numFields", len(fields),
		"firstFieldPath", PathToString(fields[0].Path))

	switch plan.Kind {

	case PlanPrimitive:
		m := NewMerger(currentPath)
		logger.Debug("executing primitive merge",
			"path", PathToString(currentPath),
			"numFields", len(fields))

		for _, f := range fields {
			m.AddTypes(label, f.Type)
		}
		return m

	case PlanArray:
		m := NewMerger(currentPath)

		// Determine array type from fields
		arrayType := TypeArray
		if len(fields) > 0 && fields[0].Type == TypeObjInt {
			arrayType = TypeObjInt
		}
		m.AddTypes(label, arrayType)

		logger.Debug("executing array merge",
			"path", PathToString(currentPath),
			"strategy", plan.ArrayStrategy,
			"numFields", len(fields),
			"arrayType", arrayType)

		// Group array elements according to strategy
		buckets := groupArrayElements(fields, plan.ArrayStrategy, logger)

		logger.Debug("grouped array elements",
			"path", PathToString(currentPath),
			"numBuckets", len(buckets),
			"strategy", plan.ArrayStrategy)

		// Merge each bucket
		for key, elems := range buckets {
			// Build the child path: current path + key
			childPath := append(append([]string{}, currentPath...), key)

			logger.Debug("merging array bucket",
				"bucketKey", key,
				"childPath", PathToString(childPath),
				"numElems", len(elems))

			child := executeMergeWithPath(plan.Elem, label, elems, childPath, logger)
			m.AddChild(key, label, child)
		}
		return m

	case PlanObject:
		logger.Debug("executing object merge",
			"path", PathToString(currentPath),
			"numFields", len(fields))

		strategy := NewObjectMergeStrategy(logger)
		return strategy.Merge(currentPath, label, plan, fields)

	default:
		logger.Debug("unknown plan kind, treating as primitive",
			"path", PathToString(currentPath),
			"kind", plan.Kind)
		m := NewMerger(currentPath)
		for _, f := range fields {
			m.AddTypes(label, f.Type)
		}
		return m
	}
}

// groupArrayElements groups array element FieldInfos by key according to strategy
func groupArrayElements(
	fields []*FieldInfo,
	strategy ArrayStrategy,
	logger *slog.Logger,
) map[string][]*FieldInfo {
	buckets := make(map[string][]*FieldInfo)

	for _, field := range fields {
		if field.Type != TypeArray && field.Type != TypeObjInt {
			logger.Debug("skipping non-array field in groupArrayElements",
				"path", PathToString(field.Path),
				"type", field.Type)
			continue
		}

		logger.Debug("processing array field",
			"path", PathToString(field.Path),
			"numChildren", len(field.Children),
			"strategy", strategy)

		// Process each child element
		for _, child := range field.Children {
			var key string

			switch strategy {
			case ArrayCollapse:
				// All elements go to wildcard key
				key = ""
			case ArrayKeepIndices:
				// Keep original index
				key = lastPathSegment(child.Path)
			}

			logger.Debug("grouping array element",
				"parentPath", PathToString(field.Path),
				"childPath", PathToString(child.Path),
				"key", key,
				"childType", child.Type)

			buckets[key] = append(buckets[key], child)
		}
	}

	logger.Debug("array element grouping complete",
		"numBuckets", len(buckets),
		"bucketKeys", func() []string {
			keys := make([]string, 0, len(buckets))
			for k := range buckets {
				keys = append(keys, k)
			}
			return keys
		}())

	return buckets
}

// MergeFieldInfo is the main entry point - creates plan and executes it
func MergeFieldInfo(m *Merger, label string, field *FieldInfo, logger *slog.Logger) *Merger {
	logger.Debug("starting merge",
		"path", PathToString(field.Path),
		"label", label,
		"type", field.Type)

	// Step 1: Create the plan
	plan := PlanShape(field)

	logger.Debug("created merge plan",
		"path", PathToString(field.Path),
		"planKind", plan.Kind)

	// Step 2: Execute the plan - start with the field's actual path
	result := executeMergeWithPath(plan, label, []*FieldInfo{field}, field.Path, logger)

	// If a merger was provided, merge into it
	if m != nil {
		for t := range result.TypesMap {
			m.AddTypes(label, t)
		}
		for _, key := range result.ChildrenKeys {
			m.AddChild(key, label, result.ChildrenMap[key])
		}
		return m
	}

	return result
}

// FieldInfoToString converts a FieldInfo tree to a readable string
func FieldInfoToString(field *FieldInfo, indent string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%sPath: %s\n", indent, PathToString(field.Path))
	fmt.Fprintf(&sb, "%sType: %v\n", indent, field.Type)
	fmt.Fprintf(&sb, "%sChildren: %d\n", indent, len(field.Children))

	for i, child := range field.Children {
		fmt.Fprintf(&sb, "%s  Child %d:\n", indent, i)
		sb.WriteString(FieldInfoToString(child, indent+"    "))
	}

	return sb.String()
}

// DiagnoseFieldInfo prints detailed diagnostic info about a FieldInfo tree
func DiagnoseFieldInfo(field *FieldInfo, name string) {
	fmt.Printf("\n=== Diagnosing FieldInfo: %s ===\n", name)
	fmt.Print(FieldInfoToString(field, ""))
	fmt.Println("=== End Diagnosis ===")
}

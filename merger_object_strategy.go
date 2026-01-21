package jsontype

import "log/slog"

// Layer 2: Object Merge Strategy
// All object semantics live here and ONLY here:
// - field presence tracking
// - nullability determination
// - independent field merging

type ObjectMergeStrategy struct {
	logger *slog.Logger
}

func NewObjectMergeStrategy(logger *slog.Logger) *ObjectMergeStrategy {
	return &ObjectMergeStrategy{logger: logger}
}

// Merge handles all object merging logic
func (s *ObjectMergeStrategy) Merge(
	path []string,
	label string,
	plan *MergePlan,
	fields []*FieldInfo,
) *Merger {
	m := NewMerger(path)
	m.AddTypes(label, TypeObj)

	// Group fields by name
	fieldGroups := s.groupObjectFields(fields)
	totalElements := len(fields)

	s.logger.Debug("merging object fields",
		"path", PathToString(path),
		"totalElements", totalElements,
		"uniqueFields", len(fieldGroups))

	// Merge each field independently
	for fieldName, fieldInfos := range fieldGroups {
		childPlan := plan.Fields[fieldName]
		if childPlan == nil {
			s.logger.Debug("no plan for field (shouldn't happen)",
				"path", PathToString(path),
				"field", fieldName)
			childPlan = &MergePlan{Kind: PlanPrimitive}
		}

		// Build child path: parent path + field name
		childPath := append(append([]string{}, path...), fieldName)

		s.logger.Debug("merging object field",
			"fieldName", fieldName,
			"childPath", PathToString(childPath),
			"numOccurrences", len(fieldInfos))

		// Merge this field's occurrences using executeMergeWithPath
		child := executeMergeWithPath(childPlan, label, fieldInfos, childPath, s.logger)

		// Apply nullability: if this field doesn't appear in all elements, it's nullable
		appearances := len(fieldInfos)
		if appearances < totalElements {
			s.logger.Debug("marking field as nullable",
				"path", PathToString(path),
				"field", fieldName,
				"appearances", appearances,
				"totalElements", totalElements)
			child.AddTypes(label, TypeNull)
		}

		m.AddChild(fieldName, label, child)
	}

	return m
}

// groupObjectFields groups field infos by their field name
func (s *ObjectMergeStrategy) groupObjectFields(fields []*FieldInfo) map[string][]*FieldInfo {
	groups := make(map[string][]*FieldInfo)

	s.logger.Debug("grouping object fields - START",
		"numFields", len(fields))

	for i, field := range fields {
		s.logger.Debug("examining field",
			"index", i,
			"path", PathToString(field.Path),
			"type", field.Type,
			"numChildren", len(field.Children))

		if field.Type != TypeObj {
			s.logger.Debug("skipping non-object field in groupObjectFields",
				"path", PathToString(field.Path),
				"type", field.Type)
			continue
		}

		// Each child of this object represents a field
		for j, child := range field.Children {
			fieldName := lastPathSegment(child.Path)
			s.logger.Debug("found object field",
				"objectIndex", i,
				"childIndex", j,
				"objectPath", PathToString(field.Path),
				"fieldName", fieldName,
				"fieldPath", PathToString(child.Path),
				"fieldType", child.Type)
			groups[fieldName] = append(groups[fieldName], child)
		}
	}

	s.logger.Debug("object field grouping complete",
		"totalGroups", len(groups))

	for name, infos := range groups {
		s.logger.Debug("field group",
			"fieldName", name,
			"occurrences", len(infos))
	}

	return groups
}

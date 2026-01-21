package jsontype

import "maps"

// Layer 1: Shape Planner
// Pure, side-effect-free planning phase
// Answers ONE question: "Given this FieldInfo subtree, how should paths be structured?"

type PlanKind int

const (
	PlanPrimitive PlanKind = iota
	PlanArray
	PlanObject
)

type ArrayStrategy int

const (
	ArrayCollapse    ArrayStrategy = iota // use ""
	ArrayKeepIndices                      // use "0", "1", ...
)

// MergePlan describes the shape of the merged result
type MergePlan struct {
	Kind PlanKind

	// For arrays
	ArrayStrategy ArrayStrategy
	Elem          *MergePlan

	// For objects
	Fields map[string]*MergePlan
}

// PlanShape determines the merge strategy for a FieldInfo tree
// This is pure logic with no side effects
func PlanShape(field *FieldInfo) *MergePlan {
	switch field.Type {

	case TypeArray, TypeObjInt:
		elemPlans := make([]*MergePlan, 0, len(field.Children))
		for _, ch := range field.Children {
			elemPlans = append(elemPlans, PlanShape(ch))
		}

		// Decide array strategy
		strategy := ArrayCollapse
		if IsMixedContainer(field) {
			strategy = ArrayKeepIndices
		}

		// Arrays always have exactly ONE element plan
		return &MergePlan{
			Kind:          PlanArray,
			ArrayStrategy: strategy,
			Elem:          unifyPlans(elemPlans),
		}

	case TypeObj:
		// CRITICAL LAW: Objects never collapse keys. Ever.
		// Nested under arrays? Still preserved.
		// Non-mixed? Still preserved.
		// Generalized? Still preserved.
		fields := map[string]*MergePlan{}
		for _, ch := range field.Children {
			key := lastPathSegment(ch.Path)
			fields[key] = mergeObjectFieldPlans(fields[key], PlanShape(ch))
		}
		return &MergePlan{
			Kind:   PlanObject,
			Fields: fields,
		}

	default:
		return &MergePlan{Kind: PlanPrimitive}
	}
}

// unifyPlans combines multiple element plans into a single unified plan
func unifyPlans(plans []*MergePlan) *MergePlan {
	if len(plans) == 0 {
		return &MergePlan{Kind: PlanPrimitive}
	}

	// Start with the first plan
	result := plans[0]

	// Merge with subsequent plans
	for i := 1; i < len(plans); i++ {
		result = mergeTwoPlans(result, plans[i])
	}

	return result
}

// mergeTwoPlans combines two plans into one
func mergeTwoPlans(a, b *MergePlan) *MergePlan {
	// If kinds differ, we have mixed content - treat as primitive
	if a.Kind != b.Kind {
		return &MergePlan{Kind: PlanPrimitive}
	}

	switch a.Kind {
	case PlanPrimitive:
		return &MergePlan{Kind: PlanPrimitive}

	case PlanArray:
		// Use the more conservative strategy
		strategy := a.ArrayStrategy
		if b.ArrayStrategy == ArrayKeepIndices {
			strategy = ArrayKeepIndices
		}
		return &MergePlan{
			Kind:          PlanArray,
			ArrayStrategy: strategy,
			Elem:          mergeTwoPlans(a.Elem, b.Elem),
		}

	case PlanObject:
		// Merge object fields
		fields := make(map[string]*MergePlan)

		// Copy fields from a
		maps.Copy(fields, a.Fields)

		// Merge fields from b
		for k, v := range b.Fields {
			fields[k] = mergeObjectFieldPlans(fields[k], v)
		}

		return &MergePlan{
			Kind:   PlanObject,
			Fields: fields,
		}

	default:
		return &MergePlan{Kind: PlanPrimitive}
	}
}

// mergeObjectFieldPlans merges plans for the same object field
func mergeObjectFieldPlans(existing, newMergePlan *MergePlan) *MergePlan {
	if existing == nil {
		return newMergePlan
	}
	if newMergePlan == nil {
		return existing
	}
	return mergeTwoPlans(existing, newMergePlan)
}

// lastPathSegment returns the last segment of a path
func lastPathSegment(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return path[len(path)-1]
}

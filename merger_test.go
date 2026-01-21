package jsontype_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/4nd3r5on/jsontype"
)

// TestSimpleArrayOfObjects tests the most basic case first
func TestSimpleArrayOfObjects(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Simple: array of objects [{x:1}, {x:2, y:3}]
	root := &jsontype.FieldInfo{
		Path: []string{"arr"},
		Type: jsontype.TypeArray,
		Children: []*jsontype.FieldInfo{
			{
				Path: []string{"arr", "0"},
				Type: jsontype.TypeObj,
				Children: []*jsontype.FieldInfo{
					{Path: []string{"arr", "0", "x"}, Type: jsontype.TypeInt32},
				},
			},
			{
				Path: []string{"arr", "1"},
				Type: jsontype.TypeObj,
				Children: []*jsontype.FieldInfo{
					{Path: []string{"arr", "1", "x"}, Type: jsontype.TypeInt32},
					{Path: []string{"arr", "1", "y"}, Type: jsontype.TypeInt32},
				},
			},
		},
	}

	t.Log("Input structure:")
	t.Log(jsontype.FieldInfoToString(root, ""))

	merger := jsontype.MergeFieldInfo(nil, "test", root, logger)

	t.Log("\nOutput structure:")
	t.Log(jsontype.MergerToString(merger, "", true))

	// Expected: arr => array, arr[] => object with x and y (y nullable)
	arrNode, exists := merger.ChildrenMap[""]
	if !exists {
		t.Fatalf("Expected wildcard child for array")
	}

	if _, hasObj := arrNode.TypesMap[jsontype.TypeObj]; !hasObj {
		t.Errorf("Expected object type in array elements")
	}

	// Check fields
	_, hasX := arrNode.ChildrenMap["x"]
	yField, hasY := arrNode.ChildrenMap["y"]

	if !hasX {
		t.Errorf("Field 'x' is missing!")
	}
	if !hasY {
		t.Errorf("Field 'y' is missing!")
	}

	// y should be nullable
	if hasY {
		if _, hasNull := yField.TypesMap[jsontype.TypeNull]; !hasNull {
			t.Errorf("Field 'y' should be nullable")
		}
	}

	// Check path - should be ["arr", ""]
	expectedPath := []string{"arr", ""}
	if jsontype.PathToString(arrNode.Path) != jsontype.PathToString(expectedPath) {
		t.Errorf("Wrong path for array element:\nGot:      %s\nExpected: %s\n%s",
			jsontype.PathToString(arrNode.Path),
			jsontype.PathToString(expectedPath),
			jsontype.ComparePaths(arrNode.Path, expectedPath))
	}

	t.Logf("✓ Simple array of objects works correctly")
}

// TestArrayOfArraysOfObjects verifies the original bug is fixed
func TestArrayOfArraysOfObjects(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Construct the input: array<array<object>>
	// This matches your JSON example with array_of_arrays_of_objects
	root := &jsontype.FieldInfo{
		Path: []string{},
		Type: jsontype.TypeObj,
		Children: []*jsontype.FieldInfo{
			{
				Path: []string{"array_of_arrays_of_objects"},
				Type: jsontype.TypeArray,
				Children: []*jsontype.FieldInfo{
					// First inner array: [{x:1, y:2}, {x:3, y:4}]
					{
						Path: []string{"array_of_arrays_of_objects", "0"},
						Type: jsontype.TypeArray,
						Children: []*jsontype.FieldInfo{
							{
								Path: []string{"array_of_arrays_of_objects", "0", "0"},
								Type: jsontype.TypeObj,
								Children: []*jsontype.FieldInfo{
									{Path: []string{"array_of_arrays_of_objects", "0", "0", "x"}, Type: jsontype.TypeInt32},
									{Path: []string{"array_of_arrays_of_objects", "0", "0", "y"}, Type: jsontype.TypeInt32},
								},
							},
							{
								Path: []string{"array_of_arrays_of_objects", "0", "1"},
								Type: jsontype.TypeObj,
								Children: []*jsontype.FieldInfo{
									{Path: []string{"array_of_arrays_of_objects", "0", "1", "x"}, Type: jsontype.TypeInt32},
									{Path: []string{"array_of_arrays_of_objects", "0", "1", "y"}, Type: jsontype.TypeInt32},
								},
							},
						},
					},
					// Second inner array: [{x:5, y:6}, {x:7, y:8}]
					{
						Path: []string{"array_of_arrays_of_objects", "1"},
						Type: jsontype.TypeArray,
						Children: []*jsontype.FieldInfo{
							{
								Path: []string{"array_of_arrays_of_objects", "1", "0"},
								Type: jsontype.TypeObj,
								Children: []*jsontype.FieldInfo{
									{Path: []string{"array_of_arrays_of_objects", "1", "0", "x"}, Type: jsontype.TypeInt32},
									{Path: []string{"array_of_arrays_of_objects", "1", "0", "y"}, Type: jsontype.TypeInt32},
								},
							},
							{
								Path: []string{"array_of_arrays_of_objects", "1", "1"},
								Type: jsontype.TypeObj,
								Children: []*jsontype.FieldInfo{
									{Path: []string{"array_of_arrays_of_objects", "1", "1", "x"}, Type: jsontype.TypeInt32},
									{Path: []string{"array_of_arrays_of_objects", "1", "1", "y"}, Type: jsontype.TypeInt32},
								},
							},
						},
					},
					// Third inner array: [{x:9, y:10, z:11}, {x:12, y:13}]
					{
						Path: []string{"array_of_arrays_of_objects", "2"},
						Type: jsontype.TypeArray,
						Children: []*jsontype.FieldInfo{
							{
								Path: []string{"array_of_arrays_of_objects", "2", "0"},
								Type: jsontype.TypeObj,
								Children: []*jsontype.FieldInfo{
									{Path: []string{"array_of_arrays_of_objects", "2", "0", "x"}, Type: jsontype.TypeInt32},
									{Path: []string{"array_of_arrays_of_objects", "2", "0", "y"}, Type: jsontype.TypeInt32},
									{Path: []string{"array_of_arrays_of_objects", "2", "0", "z"}, Type: jsontype.TypeInt32},
								},
							},
							{
								Path: []string{"array_of_arrays_of_objects", "2", "1"},
								Type: jsontype.TypeObj,
								Children: []*jsontype.FieldInfo{
									{Path: []string{"array_of_arrays_of_objects", "2", "1", "x"}, Type: jsontype.TypeInt32},
									{Path: []string{"array_of_arrays_of_objects", "2", "1", "y"}, Type: jsontype.TypeInt32},
								},
							},
						},
					},
				},
			},
		},
	}

	t.Log("Input structure (root):")
	t.Log(jsontype.FieldInfoToString(root, ""))

	// Merge it
	merger := jsontype.MergeFieldInfo(nil, "test", root, logger)

	t.Log("\nOutput structure:")
	t.Log(jsontype.MergerToString(merger, "", true))

	// Verify the expected structure
	// Expected:
	// [] => object<array>
	//   ["array_of_arrays_of_objects"] => array<array>
	//     ["array_of_arrays_of_objects", ""] => array<object>
	//       ["array_of_arrays_of_objects", "", ""] => object<int32>
	//         ["array_of_arrays_of_objects", "", "", "x"] => int32
	//         ["array_of_arrays_of_objects", "", "", "y"] => int32
	//         ["array_of_arrays_of_objects", "", "", "z"] => int32 | null

	// Check root is object
	if _, hasObj := merger.TypesMap[jsontype.TypeObj]; !hasObj {
		t.Errorf("Expected root to be object")
	}

	// Check array_of_arrays_of_objects exists and is array
	arrayField, exists := merger.ChildrenMap["array_of_arrays_of_objects"]
	if !exists {
		t.Fatalf("Expected array_of_arrays_of_objects field")
	}
	if _, hasArray := arrayField.TypesMap[jsontype.TypeArray]; !hasArray {
		t.Errorf("Expected array_of_arrays_of_objects to be array")
	}

	// Check first level wildcard (outer array collapsed)
	innerArray, exists := arrayField.ChildrenMap[""]
	if !exists {
		t.Fatalf("Expected wildcard child for outer array")
	}
	if _, hasArray := innerArray.TypesMap[jsontype.TypeArray]; !hasArray {
		t.Errorf("Expected inner level to be array")
	}

	// Check second level wildcard (inner array collapsed)
	objectLevel, exists := innerArray.ChildrenMap[""]
	if !exists {
		t.Fatalf("Expected wildcard child for inner array")
	}
	if _, hasObj := objectLevel.TypesMap[jsontype.TypeObj]; !hasObj {
		t.Errorf("Expected object at element level")
	}

	// CRITICAL: Check that object fields are preserved
	xField, hasX := objectLevel.ChildrenMap["x"]
	yField, hasY := objectLevel.ChildrenMap["y"]
	zField, hasZ := objectLevel.ChildrenMap["z"]

	if !hasX {
		t.Errorf("Expected field 'x' to be preserved")
	}
	if !hasY {
		t.Errorf("Expected field 'y' to be preserved")
	}
	if !hasZ {
		t.Errorf("Expected field 'z' to be preserved")
	}

	// Verify x and y are int32
	if hasX {
		if _, hasInt := xField.TypesMap[jsontype.TypeInt32]; !hasInt {
			t.Errorf("Expected x to be int32")
		}
	}
	if hasY {
		if _, hasInt := yField.TypesMap[jsontype.TypeInt32]; !hasInt {
			t.Errorf("Expected y to be int32")
		}
	}

	// Verify z is int32 | null (appears in 1 of 6 objects)
	if hasZ {
		if _, hasInt := zField.TypesMap[jsontype.TypeInt32]; !hasInt {
			t.Errorf("Expected z to be int32")
		}
		if _, hasNull := zField.TypesMap[jsontype.TypeNull]; !hasNull {
			t.Errorf("Expected z to be nullable (doesn't appear in all objects)")
		}
	}

	// Verify the bug is fixed: there should NOT be a single wildcard child
	// containing mixed types
	if len(objectLevel.ChildrenMap) == 1 {
		for key := range objectLevel.ChildrenMap {
			if key == "" {
				t.Errorf("BUG STILL PRESENT: Object fields collapsed to wildcard!")
			}
		}
	}

	t.Logf("✓ Bug is fixed: object fields preserved as x, y, z (not collapsed)")
	t.Logf("✓ Structure: array ▸ array ▸ object{x, y, z}")
	t.Logf("✓ Field z is correctly nullable")
}

// TestPlanShape_ObjectsNeverCollapse verifies the core law
func TestPlanShape_ObjectsNeverCollapse(t *testing.T) {
	// Test: object nested under non-mixed array
	field := &jsontype.FieldInfo{
		Path: []string{"arr"},
		Type: jsontype.TypeArray,
		Children: []*jsontype.FieldInfo{
			{
				Path: []string{"arr", "0"},
				Type: jsontype.TypeObj,
				Children: []*jsontype.FieldInfo{
					{Path: []string{"arr", "0", "name"}, Type: jsontype.TypeString},
					{Path: []string{"arr", "0", "age"}, Type: jsontype.TypeInt32},
				},
			},
			{
				Path: []string{"arr", "1"},
				Type: jsontype.TypeObj,
				Children: []*jsontype.FieldInfo{
					{Path: []string{"arr", "1", "name"}, Type: jsontype.TypeString},
					{Path: []string{"arr", "1", "age"}, Type: jsontype.TypeInt32},
				},
			},
		},
	}

	plan := jsontype.PlanShape(field)

	t.Log("Plan:")
	t.Log(jsontype.PlanToString(plan, "", true))

	// Array should collapse (non-mixed)
	if plan.Kind != jsontype.PlanArray {
		t.Fatalf("Expected array plan")
	}
	if plan.ArrayStrategy != jsontype.ArrayCollapse {
		t.Errorf("Expected array to collapse")
	}

	// But the object inside should preserve fields
	objPlan := plan.Elem
	if objPlan.Kind != jsontype.PlanObject {
		t.Fatalf("Expected object plan for elements")
	}

	// CRITICAL: Object fields must be preserved
	if len(objPlan.Fields) != 2 {
		t.Errorf("Expected 2 object fields, got %d", len(objPlan.Fields))
	}
	if _, hasName := objPlan.Fields["name"]; !hasName {
		t.Errorf("Expected 'name' field to be preserved")
	}
	if _, hasAge := objPlan.Fields["age"]; !hasAge {
		t.Errorf("Expected 'age' field to be preserved")
	}

	t.Logf("✓ Core law verified: Objects never collapse keys")
	t.Logf("✓ Even when nested under non-mixed arrays")
}

package jsontype

type DetectedType string

const (
	TypeUnknown DetectedType = "unknown"
	TypeNull    DetectedType = "null"
	TypeString  DetectedType = "string"
	TypeBool    DetectedType = "bool"
	TypeInt32   DetectedType = "int32"
	TypeInt64   DetectedType = "int64"
	TypeFloat64 DetectedType = "float64"
	// Containers
	TypeObj    DetectedType = "object"
	TypeObjInt DetectedType = "object_int"
	TypeArray  DetectedType = "array"
)

// FieldInfo represents a single field/path in the JSON structure
// It serves only for a run through a single file (since)
type FieldInfo struct {
	Parent *FieldInfo
	// Full path like ["obj1", "obj2", "field"]
	Path []string
	// If container -- will contain one of the following types: TypeObj, TypeObjInt, TypeObjArray
	Type DetectedType

	// Container-specific info
	Children []*FieldInfo // Ordered children for objects/arrays
	// Key: "0", "1", "2" for arrays; "1", "5", "12" for obj_int
	ChildrenMap map[string]*FieldInfo // Quick lookup by name/key
}

func IsContainerType(t DetectedType) bool {
	switch t {
	case TypeObj, TypeArray, TypeObjInt:
		return true
	}
	return false
}

// IsMixedContainer returns true if container has different types of children elements
func IsMixedContainer(field *FieldInfo) bool {
	if len(field.Children) < 1 {
		return false
	}

	firstType := field.Children[0].Type
	for _, child := range field.Children {
		if child.Type != firstType {
			return true
		}
	}
	return false
}

func GetChilderTypes(field *FieldInfo) (types []DetectedType) {
	if len(field.Children) == 0 {
		return []DetectedType{TypeUnknown}
	}
	types = make([]DetectedType, 0)
	seen := make(map[DetectedType]struct{})
	for _, child := range field.Children {
		if _, isSeen := seen[child.Type]; isSeen {
			continue
		}
		types = append(types, child.Type)
		seen[child.Type] = struct{}{}
	}
	return types
}

func TypesToString(types []DetectedType) []string {
	out := make([]string, 0, len(types))
	for _, t := range types {
		out = append(out, string(t))
	}
	return out
}

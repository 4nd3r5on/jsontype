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

// Extended string detection/analysis
// Detection code is at ./detect_str_type.go
const (
	// Common
	TypeUUID            DetectedType = "string-uuid"             // f3a9c2e7-6b4d-4f81-9a6c-2d8e5b71c0fa
	TypeFilepathWindows DetectedType = "string-filepath-windows" // C:/user/file
	TypeEmail           DetectedType = "string-email"            // admin@email.com
	TypePhone           DetectedType = "string-phone"            // +380661153394

	// Web
	TypeLink   DetectedType = "string-link"   // https://google.com or google.com/search
	TypeDomain DetectedType = "string-domain" // google.com

	// Encoding
	TypeHEX       DetectedType = "string-hex"        // "a9f3c2e7b4d81f6a"
	TypeBase64    DetectedType = "string-base64"     // "Qm9yZGVybGluZUVudHJvcHk="
	TypeBase64URL DetectedType = "string-base64-url" // "Xk3rA9mZP2Q7Lw8N0B6f_Q"

	// Networking
	TypeIPv4         DetectedType = "string-ipv4"           // 127.0.0.1
	TypeIPv4WithMask DetectedType = "string-ipv4-with-mask" // 127.0.0.1/32
	TypeIPv6         DetectedType = "string-ipv6"           // 2a03:2880:21ff:001f:face:b00c:dead:beef
	TypeIPv4PortPair DetectedType = "string-ipv4-port-pair" // 127.0.0.1:443
	TypeIPv6PortPair DetectedType = "string-ipv6-port-pair" // [2a03:2880:21ff:1f:face:b00c:dead:beef]:43792
	TypeMAC          DetectedType = "string-mac"            // 9e:3b:74:a1:5f:c2
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

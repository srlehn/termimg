package desktop

// Type is the type of desktop entry.
type Type uint8

// These are the possible desktop entry types.
const (
	None Type = iota // No type. This is bad.
	Application
	Link
	Directory
	Unknown // Any unknown type.
)

// ParseType converts the given string s into a Type.
func ParseType(s string) Type {
	switch s {
	case None.String():
		return None
	case Application.String():
		return Application
	case Link.String():
		return Link
	case Directory.String():
		return Directory
	default:
		return Unknown
	}
}

// String returns the Type as a string.
func (t Type) String() string {
	switch t {
	case None:
		return ""
	case Application:
		return "Application"
	case Link:
		return "Link"
	case Directory:
		return "Directory"
	default:
		return "Unknown"
	}
}

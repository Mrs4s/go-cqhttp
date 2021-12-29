package log

// Level represents a logging level.
type Level int

// Level constants.
const (
	FatalLevel Level = iota
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

// String convert level to string; use upper to avoid strings.ToUpper
func (l Level) String() string {
	switch l {
	case FatalLevel:
		return "FATAL"
	case ErrorLevel:
		return "ERROR"
	case WarnLevel:
		return "WARN"
	case InfoLevel:
		return "INFO"
	case DebugLevel:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

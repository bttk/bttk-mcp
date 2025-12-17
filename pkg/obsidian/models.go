package obsidian

// Note represents the JSON structure of a note returned by the API.
// It corresponds to the 'NoteJson' schema in the OpenAPI spec.
type Note struct {
	Content     string                 `json:"content"`
	Frontmatter map[string]interface{} `json:"frontmatter"`
	Path        string                 `json:"path"`
	Stat        FileStat               `json:"stat"`
	Tags        []string               `json:"tags"`
}

// FileStat contains file system metadata.
type FileStat struct {
	Ctime float64 `json:"ctime"`
	Mtime float64 `json:"mtime"`
	Size  float64 `json:"size"`
}

// ErrorResponse represents an error returned by the API.
type ErrorResponse struct {
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
}

// Error implements the error interface.
func (e *ErrorResponse) Error() string {
	return e.Message
}

type PatchOperation string

const (
	PatchAppend  PatchOperation = "append"
	PatchPrepend PatchOperation = "prepend"
	PatchReplace PatchOperation = "replace"
)

type TargetType string

const (
	TargetHeading     TargetType = "heading"
	TargetBlock       TargetType = "block"
	TargetFrontmatter TargetType = "frontmatter"
)

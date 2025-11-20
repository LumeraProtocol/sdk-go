package types

// DownloadResult contains the result of a download operation
type DownloadResult struct {
	ActionID   string
	TaskID     string
	OutputPath string
}

// ActionResult contains the result of an action registration
type ActionResult struct {
	ActionID string
	TxHash   string
	Height   int64
}

// CascadeResult contains the result of a cascade operation
type CascadeResult struct {
	ActionResult
	TaskID   string
}

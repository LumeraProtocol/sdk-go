package types

// CascadeResult contains the result of a cascade operation
type CascadeResult struct {
	ActionID string
	TaskID   string
	TxHash   string
}

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

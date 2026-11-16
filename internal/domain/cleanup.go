package domain

// CleanupArea is one place with files that are safe to delete, as analysed.
type CleanupArea struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Note  string   `json:"note"` // what deleting means for the user
	Paths []string `json:"paths"`
	Files int      `json:"files"`
	Bytes uint64   `json:"bytes"`
}

// CleanupResult is what a cleanup actually did, never the estimate.
type CleanupResult struct {
	ChangeID string `json:"changeId"`
	Deleted  int    `json:"deleted"`
	Freed    uint64 `json:"freed"`
	Skipped  int    `json:"skipped"` // in use or refused; left alone
}

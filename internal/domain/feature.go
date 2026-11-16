package domain

// WindowsFeature is an optional feature with what it is for in plain words.
type WindowsFeature struct {
	Name      string `json:"name"`
	Title     string `json:"title"`
	Note      string `json:"note"`
	Enabled   bool   `json:"enabled"`
	Suggest   bool   `json:"suggest"`   // worth turning off for most people
	Protected bool   `json:"protected"` // other tools need it; never suggested
}

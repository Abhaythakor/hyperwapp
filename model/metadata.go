package model

import "time"

const (
	InputTypeOnline   = "online"
	InputTypeOffline  = "offline"
	SourceWappalyzer  = "wappalyzer"
	SourceHeadersOnly = "wappalyzer-header"
	SourceBodyOnly    = "wappalyzer-body"
)

type Meta struct {
	Tool        string    `json:"tool"`
	Version     string    `json:"version"`
	GeneratedAt time.Time `json:"generated_at"`
	Mode        string    `json:"mode"`       // all | domain
	InputType   string    `json:"input_type"` // online | offline
}

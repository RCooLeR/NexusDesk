// Package approvals owns native approval records and workspace access policy.
package approvals

import "time"

type Record struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Risk      string    `json:"risk"`
	Decision  string    `json:"decision"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type Policy struct {
	WorkspaceRoot     string    `json:"workspaceRoot"`
	FullProjectAccess bool      `json:"fullProjectAccess"`
	GrantedAt         time.Time `json:"grantedAt"`
	ExpiresAt         time.Time `json:"expiresAt"`
	Message           string    `json:"message"`
}

func (p Policy) Active(now time.Time) bool {
	return p.FullProjectAccess && !p.ExpiresAt.IsZero() && now.UTC().Before(p.ExpiresAt.UTC())
}

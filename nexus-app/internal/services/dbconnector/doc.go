// Package dbconnector owns read-only database connector services for workspace
// and saved-profile databases. UI packages request explicit inspections or
// guarded queries; connector services keep path safety, read-only mode,
// timeouts, caps, schema metadata, and future credential boundaries out of
// widgets.
package dbconnector

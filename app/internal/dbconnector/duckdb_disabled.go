//go:build !duckdb

package dbconnector

import (
	"context"
	"errors"

	"NexusAugenticStudio/internal/storage"
)

var errDuckDBBuildTagRequired = errors.New("DuckDB connector requires a CGO-enabled build with the duckdb build tag")

func TestDuckDBProfile(profile storage.ConnectorProfile) (ConnectorProfileStatus, error) {
	if err := requireDuckDBProfile(profile); err != nil {
		return ConnectorProfileStatus{}, err
	}
	return ConnectorProfileStatus{}, errDuckDBBuildTagRequired
}

func QueryDuckDBProfile(profile storage.ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	request = NormalizeConnectorQueryRequest(request)
	return QueryDuckDBProfileContext(context.Background(), profile, request)
}

func QueryDuckDBProfileContext(ctx context.Context, profile storage.ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	_ = ctx
	if err := requireDuckDBProfile(profile); err != nil {
		return ConnectorQueryResult{}, err
	}
	_, err := normalizeReadOnlyConnectorSQL(request.SQL)
	if err != nil {
		return ConnectorQueryResult{}, err
	}
	return ConnectorQueryResult{}, errDuckDBBuildTagRequired
}

func InspectDuckDBProfile(profile storage.ConnectorProfile) (ConnectorMetadata, error) {
	if err := requireDuckDBProfile(profile); err != nil {
		return ConnectorMetadata{}, err
	}
	return ConnectorMetadata{}, errDuckDBBuildTagRequired
}

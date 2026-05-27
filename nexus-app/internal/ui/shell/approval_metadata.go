package shell

import (
	approvalsSvc "nexusdesk/internal/services/approvals"
	metadataSvc "nexusdesk/internal/services/metadata"
)

type approvalMetadataRepository struct {
	store *metadataSvc.Store
}

func newApprovalMetadataRepository(store *metadataSvc.Store) approvalMetadataRepository {
	return approvalMetadataRepository{store: store}
}

func (r approvalMetadataRepository) SaveApprovalRecord(record approvalsSvc.Record) error {
	return r.store.SaveApprovalRecord(metadataSvc.ApprovalRecord{
		ID:        record.ID,
		Action:    record.Action,
		Target:    record.Target,
		Risk:      record.Risk,
		Decision:  record.Decision,
		Message:   record.Message,
		CreatedAt: record.CreatedAt,
	})
}

func (r approvalMetadataRepository) ListApprovalRecords(limit int) ([]approvalsSvc.Record, error) {
	metadataRecords, err := r.store.ListApprovalRecords(limit)
	if err != nil {
		return nil, err
	}
	records := make([]approvalsSvc.Record, 0, len(metadataRecords))
	for _, record := range metadataRecords {
		records = append(records, approvalsSvc.Record{
			ID:        record.ID,
			Action:    record.Action,
			Target:    record.Target,
			Risk:      record.Risk,
			Decision:  record.Decision,
			Message:   record.Message,
			CreatedAt: record.CreatedAt,
		})
	}
	return records, nil
}

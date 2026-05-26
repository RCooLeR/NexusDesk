import {StatusBadge} from '../../components/ui';
import type {ApprovalRecord} from '../../types';

export function ApprovalLogPanel({records}: {records: ApprovalRecord[]}) {
    return (
        <div className="approval-log-panel">
            <strong>Approval Log</strong>
            {records.length === 0 ? (
                <small>No applied actions recorded yet.</small>
            ) : records.slice(0, 5).map((record) => (
                <div className="approval-log-row" key={record.id}>
                    <span>
                        <strong>{record.action}</strong>
                        <small>{record.target}</small>
                    </span>
                    <StatusBadge tone={record.risk === 'high' ? 'warning' : 'success'}>{record.decision}</StatusBadge>
                </div>
            ))}
        </div>
    );
}

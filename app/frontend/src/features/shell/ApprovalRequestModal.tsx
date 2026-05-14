import {Button} from '../../components/ui';

export type ApprovalPrompt = {
    action: string;
    target: string;
    risk: 'low' | 'medium' | 'high';
    message: string;
    confirmLabel?: string;
};

type ApprovalRequestModalProps = {
    prompt: ApprovalPrompt | null;
    onApprove: () => void;
    onCancel: () => void;
};

export function ApprovalRequestModal({prompt, onApprove, onCancel}: ApprovalRequestModalProps) {
    if (!prompt) {
        return null;
    }

    return (
        <div className="modal-backdrop" role="presentation">
            <section aria-label="Approval request" className="approval-modal" role="dialog" aria-modal="true">
                <header>
                    <span className={`risk-dot risk-${prompt.risk}`} />
                    <div>
                        <p className="eyebrow">Approval Required</p>
                        <h2>{prompt.action}</h2>
                    </div>
                </header>
                <dl>
                    <div>
                        <dt>Risk</dt>
                        <dd>{prompt.risk}</dd>
                    </div>
                    <div>
                        <dt>Target</dt>
                        <dd>{prompt.target}</dd>
                    </div>
                </dl>
                <p>{prompt.message}</p>
                <footer>
                    <Button onClick={onCancel} variant="subtle">Cancel</Button>
                    <Button onClick={onApprove} variant="primary">{prompt.confirmLabel ?? 'Approve'}</Button>
                </footer>
            </section>
        </div>
    );
}

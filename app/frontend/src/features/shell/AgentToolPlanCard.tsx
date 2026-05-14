import {Button, StatusBadge} from '../../components/ui';
import type {AgentToolDescriptor, AgentToolPlanItem, AgentToolRunRecord} from '../../types';

type AgentToolPlanCardProps = {
    tools: AgentToolDescriptor[];
    planItems: AgentToolPlanItem[];
    runs: AgentToolRunRecord[];
    isRunning: boolean;
    onDryRun: (item: AgentToolPlanItem) => void;
    onExecute: (item: AgentToolPlanItem) => void;
    onReplayRun: (run: AgentToolRunRecord) => void;
    onCompareRunTarget: (run: AgentToolRunRecord) => void;
    onRefreshPlan: () => void;
};

export function AgentToolPlanCard({tools, planItems, runs, isRunning, onDryRun, onExecute, onReplayRun, onCompareRunTarget, onRefreshPlan}: AgentToolPlanCardProps) {
    return (
        <section className="agent-tool-plan-card">
            <div className="agent-card-header">
                <div>
                    <strong>Tool Plan</strong>
                    <small>{tools.length} registered tools</small>
                </div>
                <Button onClick={onRefreshPlan} variant="subtle">Refresh</Button>
            </div>

            {planItems.length === 0 ? (
                <p className="tool-plan-empty">Select a file, dataset, artifact, or operations config to preview grounded tool steps.</p>
            ) : (
                <div className="tool-plan-list">
                    {planItems.map((item) => (
                        <div className="tool-plan-item" key={`${item.toolName}-${item.target}`}>
                            <span>
                                <strong>{item.title}</strong>
                                <small>{item.target}</small>
                            </span>
                            <div>
                                <StatusBadge tone={item.risk === 'high' || item.risk === 'medium' ? 'warning' : 'neutral'}>
                                    {item.risk}
                                </StatusBadge>
                                {item.requiresApproval && <StatusBadge tone="warning">approval</StatusBadge>}
                            </div>
                            <small>{item.status}</small>
                            <div className="tool-plan-actions">
                                <Button disabled={isRunning} onClick={() => onDryRun(item)} variant="subtle">Dry run</Button>
                                <Button disabled={isRunning || item.toolName === 'workspace.write'} onClick={() => onExecute(item)} variant="subtle">Execute</Button>
                            </div>
                        </div>
                    ))}
                </div>
            )}
            {runs.length > 0 && (
                <div className="tool-run-list">
                    <small>Recent tool runs</small>
                    {runs.slice(0, 4).map((run) => (
                        <details className="tool-run-row" key={run.id}>
                            <summary>
                                <span>
                                    <strong>{run.title || run.toolName}</strong>
                                    <small>{run.outputSummary || run.error || run.target}</small>
                                </span>
                                <StatusBadge tone={run.status === 'failed' ? 'warning' : 'neutral'}>{run.mode}</StatusBadge>
                            </summary>
                            <div className="tool-run-detail">
                                <dl>
                                    <div><dt>Target</dt><dd>{run.target || 'workspace'}</dd></div>
                                    <div><dt>Status</dt><dd>{run.status}</dd></div>
                                    <div><dt>Approval</dt><dd>{run.approvalId || 'none'}</dd></div>
                                    <div><dt>Duration</dt><dd>{run.durationMs} ms</dd></div>
                                </dl>
                                <pre>{JSON.stringify(run.inputs ?? {}, null, 2)}</pre>
                                {run.error && <small>{run.error}</small>}
                                <div className="tool-plan-actions">
                                    <Button disabled={isRunning} onClick={() => onReplayRun(run)} variant="subtle">Replay dry run</Button>
                                    <Button disabled={isRunning || !run.target} onClick={() => onCompareRunTarget(run)} variant="subtle">Diff target</Button>
                                </div>
                            </div>
                        </details>
                    ))}
                </div>
            )}
        </section>
    );
}

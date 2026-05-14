import {Button, StatusBadge} from '../../components/ui';
import type {AgentToolDescriptor, AgentToolPlanItem} from '../../types';

type AgentToolPlanCardProps = {
    tools: AgentToolDescriptor[];
    planItems: AgentToolPlanItem[];
    onRefreshPlan: () => void;
};

export function AgentToolPlanCard({tools, planItems, onRefreshPlan}: AgentToolPlanCardProps) {
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
                        </div>
                    ))}
                </div>
            )}
        </section>
    );
}

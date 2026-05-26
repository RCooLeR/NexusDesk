import {Button, Card, InlineAlert, StatusBadge} from '../../components/ui';
import type {AccessPolicy, AccessPolicyMode, ShellAccessPolicy, WorkspaceSnapshot} from '../../types';

type AccessPolicyCardProps = {
    policy: AccessPolicy;
    status: string;
    workspace: WorkspaceSnapshot | null;
    onChange: (field: keyof AccessPolicy, value: AccessPolicy[keyof AccessPolicy]) => void;
    onReset: () => void;
};

const projectOptions: Array<{value: AccessPolicyMode; title: string; detail: string}> = [
    {
        value: 'approval',
        title: 'Ask for file writes',
        detail: 'Agent and tool writes need explicit approval before files are created or changed.',
    },
    {
        value: 'full',
        title: 'Full project access',
        detail: 'Trusted agent runs may create or update files inside the active workspace through Nexus safe write tools.',
    },
];

const dataOptions: Array<{value: AccessPolicyMode; title: string; detail: string}> = [
    {
        value: 'approval',
        title: 'Ask for data actions',
        detail: 'Connector tests, inspections, exports, and broader data workflows stay explicit and reviewable.',
    },
    {
        value: 'full',
        title: 'Full read access',
        detail: 'Trusted data workflows may inspect and query configured sources with read-only guards, caps, and timeouts.',
    },
];

const shellOptions: Array<{value: ShellAccessPolicy; title: string; detail: string}> = [
    {
        value: 'disabled',
        title: 'Shell disabled',
        detail: 'Agent chat cannot run shell commands. Use dedicated Workbench tasks or future Ops flows.',
    },
    {
        value: 'approval',
        title: 'Ask every time',
        detail: 'Shell execution remains blocked from chat today; this marks the intended policy for future Ops approval flows.',
    },
];

export function AccessPolicyCard({policy, status, workspace, onChange, onReset}: AccessPolicyCardProps) {
    const workspaceLabel = workspace ? `${workspace.name} / ${workspace.root}` : 'No workspace opened';
    return (
        <Card className="settings-card access-policy-card">
            <div className="settings-form access-policy-form">
                <div className="access-policy-heading">
                    <div>
                        <strong>Access & Approvals</strong>
                        <small>{workspaceLabel}</small>
                    </div>
                    <StatusBadge tone={policy.projectFiles === 'full' || policy.dataSources === 'full' ? 'warning' : 'neutral'}>
                        {policy.projectFiles === 'full' || policy.dataSources === 'full' ? 'trusted workspace' : 'guarded'}
                    </StatusBadge>
                </div>

                <InlineAlert tone={policy.projectFiles === 'full' || policy.dataSources === 'full' ? 'warning' : 'neutral'}>
                    Full access never bypasses workspace rooting, ignored path rules, read-only SQL guards, result caps, or audit records. It only changes whether Nexus asks again before approved classes of work.
                </InlineAlert>

                <PolicyOptionGroup
                    label="Project directory"
                    name="project-access"
                    options={projectOptions}
                    value={policy.projectFiles}
                    onChange={(value) => onChange('projectFiles', value)}
                />

                <PolicyOptionGroup
                    label="Data sources"
                    name="data-access"
                    options={dataOptions}
                    value={policy.dataSources}
                    onChange={(value) => onChange('dataSources', value)}
                />

                <PolicyOptionGroup
                    label="Shell and operations"
                    name="shell-access"
                    options={shellOptions}
                    value={policy.shellCommands}
                    onChange={(value) => onChange('shellCommands', value)}
                />

                <div className="access-policy-summary">
                    <strong>Current effect</strong>
                    <ul>
                        <li>{policy.projectFiles === 'full' ? 'Agent mode can create or update workspace files after this workspace-level trust decision.' : 'Agent mode needs a per-run approval before it can create or update files.'}</li>
                        <li>{policy.dataSources === 'full' ? 'Data workflows can use read-only source inspection/query paths without extra policy prompts.' : 'Data-source workflows stay individually reviewable.'}</li>
                        <li>{policy.shellCommands === 'approval' ? 'Shell remains explicit approval territory and is not enabled from chat yet.' : 'Shell execution is disabled for chat agent runs.'}</li>
                    </ul>
                </div>

                <div className="settings-actions">
                    <small>{status}</small>
                    <Button onClick={onReset} variant="subtle">Reset to guarded</Button>
                </div>
            </div>
        </Card>
    );
}

function PolicyOptionGroup<T extends string>({
    label,
    name,
    onChange,
    options,
    value,
}: {
    label: string;
    name: string;
    onChange: (value: T) => void;
    options: Array<{value: T; title: string; detail: string}>;
    value: T;
}) {
    return (
        <fieldset className="access-policy-group">
            <legend>{label}</legend>
            <div className="access-policy-options">
                {options.map((option) => (
                    <label className={value === option.value ? 'selected' : ''} key={option.value}>
                        <input
                            checked={value === option.value}
                            name={name}
                            onChange={() => onChange(option.value)}
                            type="radio"
                        />
                        <span>
                            <strong>{option.title}</strong>
                            <small>{option.detail}</small>
                        </span>
                    </label>
                ))}
            </div>
        </fieldset>
    );
}


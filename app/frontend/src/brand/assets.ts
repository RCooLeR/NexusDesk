import symbolSilver from '../assets/brand/nexusdesk-symbol.svg';
import symbolDark from '../assets/brand/nexusdesk-symbol-dark.svg';
import aiIcon from '../assets/brand/icons/nexusdesk-icon-ai.svg';
import codeIcon from '../assets/brand/icons/nexusdesk-icon-code.svg';
import dataIcon from '../assets/brand/icons/nexusdesk-icon-data.svg';
import documentsIcon from '../assets/brand/icons/nexusdesk-icon-documents.svg';
import opsIcon from '../assets/brand/icons/nexusdesk-icon-ops.svg';

export const brandAssets = {
    symbolSilver,
    symbolDark,
    icons: {
        ai: aiIcon,
        code: codeIcon,
        data: dataIcon,
        documents: documentsIcon,
        ops: opsIcon,
    },
};

export const railItems = [
    {label: 'Workspace', icon: codeIcon, active: true},
    {label: 'Search', icon: aiIcon, active: false},
    {label: 'Data', icon: dataIcon, active: false},
    {label: 'Documents', icon: documentsIcon, active: false},
    {label: 'Operations', icon: opsIcon, active: false},
];

export const workspaceIconByName: Record<string, string> = {
    app: codeIcon,
    docs: documentsIcon,
    services: opsIcon,
};

export const capabilityIconByTitle: Record<string, string> = {
    'Workspace browser': codeIcon,
    'Configurable LLM chat': aiIcon,
    'Artifacts and approvals': documentsIcon,
};

import {brandAssets, railItems} from '../../brand/assets';

export function WorkspaceRail() {
    return (
        <aside className="workspace-rail">
            <div className="brand-mark" aria-label="NexusDesk">
                <img src={brandAssets.symbolSilver} alt="" />
            </div>
            {railItems.map((item) => (
                <button
                    key={item.label}
                    className={item.active ? 'rail-button active' : 'rail-button'}
                    title={item.label}
                    aria-label={item.label}
                >
                    <img src={item.icon} alt="" />
                </button>
            ))}
        </aside>
    );
}

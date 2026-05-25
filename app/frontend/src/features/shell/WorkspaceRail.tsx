import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {brandAssets, implementedStudioRoutes, railItems, studioRoutePrimarySurface} from '../../brand/assets';
import type {StudioRouteId} from '../../brand/assets';

type WorkspaceRailProps = {
    activeRoute: StudioRouteId;
    onRouteChange: (route: StudioRouteId) => void;
};

export function WorkspaceRail({activeRoute, onRouteChange}: WorkspaceRailProps) {
    return (
        <aside className="workspace-rail" aria-label="Main studio menu">
            <div className="brand-mark" aria-label="Nexus">
                <img src={brandAssets.symbolSilver} alt="" />
            </div>
            {railItems.map((item) => (
                <button
                    key={item.label}
                    aria-current={activeRoute === item.id ? 'page' : undefined}
                    className={activeRoute === item.id ? 'rail-button active' : 'rail-button'}
                    data-studio-route={item.id}
                    onClick={() => onRouteChange(item.id)}
                    title={`${item.label} / ${studioRoutePrimarySurface[item.id]}`}
                    aria-label={item.label}
                >
                    <FontAwesomeIcon icon={item.icon} />
                    {!implementedStudioRoutes.includes(item.id) && <span aria-hidden="true" />}
                </button>
            ))}
        </aside>
    );
}

import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {faBars, faChevronLeft} from '@fortawesome/free-solid-svg-icons';
import {brandAssets, implementedStudioRoutes, railItems, studioRoutePrimarySurface} from '../../brand/assets';
import type {StudioRouteId} from '../../brand/assets';

type WorkspaceRailProps = {
    activeRoute: StudioRouteId;
    isExpanded: boolean;
    onRouteChange: (route: StudioRouteId) => void;
    onToggleExpanded: () => void;
};

export function WorkspaceRail({activeRoute, isExpanded, onRouteChange, onToggleExpanded}: WorkspaceRailProps) {
    return (
        <aside className={isExpanded ? 'workspace-rail expanded' : 'workspace-rail'} aria-label="Main studio menu">
            <button
                aria-expanded={isExpanded}
                aria-label={isExpanded ? 'Collapse main navigation' : 'Expand main navigation'}
                className="rail-brand"
                onClick={onToggleExpanded}
                title={isExpanded ? 'Collapse main navigation' : 'Expand main navigation'}
                type="button"
            >
                <span className="brand-mark" aria-hidden="true">
                    <img src={brandAssets.symbolSilver} alt="" />
                </span>
                <span className="rail-logo" aria-hidden={!isExpanded}>
                    <img src={brandAssets.logoHorizontalWhite} alt="" />
                </span>
                <FontAwesomeIcon className="rail-toggle-icon" icon={isExpanded ? faChevronLeft : faBars} />
            </button>
            <nav className="rail-nav" aria-label="Studio routes">
                {railItems.map((item) => (
                    <button
                        key={item.label}
                        aria-current={activeRoute === item.id ? 'page' : undefined}
                        className={activeRoute === item.id ? 'rail-button active' : 'rail-button'}
                        data-studio-route={item.id}
                        onClick={() => onRouteChange(item.id)}
                        title={`${item.label} / ${studioRoutePrimarySurface[item.id]}`}
                        aria-label={item.label}
                        type="button"
                    >
                        <FontAwesomeIcon icon={item.icon} />
                        <span className="rail-label">{item.label}</span>
                        {!implementedStudioRoutes.includes(item.id) && <span className="rail-planned-dot" aria-hidden="true" />}
                    </button>
                ))}
            </nav>
        </aside>
    );
}

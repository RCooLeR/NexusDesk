import type {CSSProperties} from 'react';
import type {EditorOutlineItem} from './editorOutline';

type EditorOutlinePanelProps = {
    items: EditorOutlineItem[];
    onSelect: (line: number) => void;
};

export function EditorOutlinePanel({items, onSelect}: EditorOutlinePanelProps) {
    return (
        <aside className="editor-outline-panel" aria-label="Editor outline">
            <div className="editor-outline-heading">
                <strong>Outline</strong>
                <small>{items.length > 0 ? `${items.length} symbols` : 'No symbols'}</small>
            </div>
            <div className="editor-outline-list">
                {items.length > 0 ? items.map((item) => (
                    <button
                        className="editor-outline-item"
                        key={item.id}
                        onClick={() => onSelect(item.line)}
                        style={{'--outline-level': String(item.level)} as CSSProperties}
                        title={`${item.kind} / line ${item.line}`}
                        type="button"
                    >
                        <span>{item.kind}</span>
                        <strong>{item.label}</strong>
                        <small>{item.line}</small>
                    </button>
                )) : (
                    <div className="editor-outline-empty">No outline symbols detected for this preview.</div>
                )}
            </div>
        </aside>
    );
}

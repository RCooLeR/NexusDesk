export type EditorOutlineItem = {
    id: string;
    kind: string;
    label: string;
    level: number;
    line: number;
};

export function buildEditorOutline(fileName: string, content: string): EditorOutlineItem[] {
    const lines = content.replace(/\r\n/g, '\n').split('\n');
    const extension = fileName.split('.').pop()?.toLowerCase() ?? '';
    const items: EditorOutlineItem[] = [];
    const add = (kind: string, label: string, line: number, level = 0) => {
        const cleanLabel = label.trim();
        if (!cleanLabel) {
            return;
        }
        items.push({
            id: `${line}-${kind}-${cleanLabel}`,
            kind,
            label: cleanLabel.slice(0, 120),
            level,
            line,
        });
    };

    lines.slice(0, 4000).forEach((line, index) => {
        const lineNumber = index + 1;
        const trimmed = line.trim();
        if (!trimmed) {
            return;
        }

        const markdownHeading = /^(#{1,6})\s+(.+)$/.exec(trimmed);
        if (markdownHeading) {
            add('heading', markdownHeading[2], lineNumber, markdownHeading[1].length - 1);
            return;
        }

        if (['ts', 'tsx', 'js', 'jsx', 'mjs', 'cjs'].includes(extension)) {
            const match = /^(?:export\s+)?(?:async\s+)?(?:function|class|interface|type|enum)\s+([A-Za-z_$][\w$]*)/.exec(trimmed) ??
                /^(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>/.exec(trimmed);
            if (match) {
                add(symbolKind(trimmed), match[1], lineNumber, 0);
            }
            return;
        }

        if (extension === 'go') {
            const match = /^func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(/.exec(trimmed) ??
                /^type\s+([A-Za-z_]\w*)\s+(?:struct|interface|func|map|\[\])/.exec(trimmed);
            if (match) {
                add(trimmed.startsWith('func') ? 'func' : 'type', match[1], lineNumber, 0);
            }
            return;
        }

        if (['css', 'scss', 'sass'].includes(extension)) {
            const match = /^([.#]?[A-Za-z_][^{]+)\s*\{/.exec(trimmed);
            if (match) {
                add('selector', match[1], lineNumber, 0);
            }
            return;
        }

        if (['json', 'jsonc'].includes(extension)) {
            const match = /^"([^"]+)"\s*:/.exec(trimmed);
            if (match && leadingSpaceCount(line) <= 8) {
                add('key', match[1], lineNumber, Math.floor(leadingSpaceCount(line) / 2));
            }
            return;
        }

        if (['yaml', 'yml'].includes(extension)) {
            const match = /^([A-Za-z0-9_.-]+)\s*:/.exec(trimmed);
            if (match && leadingSpaceCount(line) <= 8) {
                add('key', match[1], lineNumber, Math.floor(leadingSpaceCount(line) / 2));
            }
        }
    });

    return items.slice(0, 120);
}

function symbolKind(line: string) {
    if (line.includes(' class ') || line.startsWith('class ') || line.startsWith('export class ')) {
        return 'class';
    }
    if (line.includes(' interface ') || line.startsWith('interface ') || line.startsWith('export interface ')) {
        return 'interface';
    }
    if (line.includes(' type ') || line.startsWith('type ') || line.startsWith('export type ')) {
        return 'type';
    }
    if (line.includes(' enum ') || line.startsWith('enum ') || line.startsWith('export enum ')) {
        return 'enum';
    }
    return 'func';
}

function leadingSpaceCount(value: string) {
    return value.length - value.trimStart().length;
}

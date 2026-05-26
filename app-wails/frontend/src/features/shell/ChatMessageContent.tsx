import type {ReactNode} from 'react';

type MarkdownBlock =
    | {type: 'paragraph'; lines: string[]}
    | {type: 'heading'; level: number; text: string}
    | {type: 'list'; ordered: boolean; items: string[]}
    | {type: 'code'; language: string; content: string}
    | {type: 'table'; rows: string[][]};

type ChatMessageContentProps = {
    content: string;
};

export function ChatMessageContent({content}: ChatMessageContentProps) {
    const blocks = parseMarkdownBlocks(content);

    if (blocks.length === 0) {
        return <p className="chat-markdown-empty">Receiving response...</p>;
    }

    return (
        <div className="chat-markdown">
            {blocks.map((block, index) => renderBlock(block, index))}
        </div>
    );
}

function renderBlock(block: MarkdownBlock, index: number) {
    switch (block.type) {
        case 'heading': {
            const HeadingTag = (`h${Math.min(block.level + 2, 5)}` as keyof JSX.IntrinsicElements);
            return <HeadingTag key={index}>{renderInline(block.text)}</HeadingTag>;
        }
        case 'list': {
            const ListTag = block.ordered ? 'ol' : 'ul';
            return (
                <ListTag key={index}>
                    {block.items.map((item, itemIndex) => <li key={itemIndex}>{renderInline(item)}</li>)}
                </ListTag>
            );
        }
        case 'code':
            return (
                <pre className="chat-code-block" key={index}>
                    {block.language && <span>{block.language}</span>}
                    <code>{block.content}</code>
                </pre>
            );
        case 'table':
            return <ChatTable key={index} rows={block.rows} />;
        case 'paragraph':
        default:
            return <p key={index}>{renderInline(block.lines.join(' '))}</p>;
    }
}

function ChatTable({rows}: {rows: string[][]}) {
    if (rows.length === 0) {
        return null;
    }

    const [header, ...body] = rows;
    return (
        <div className="chat-table-wrap">
            <table className="chat-table">
                <thead>
                    <tr>
                        {header.map((cell, index) => <th key={index}>{renderInline(cell)}</th>)}
                    </tr>
                </thead>
                <tbody>
                    {body.map((row, rowIndex) => (
                        <tr key={rowIndex}>
                            {header.map((_, cellIndex) => <td key={cellIndex}>{renderInline(row[cellIndex] ?? '')}</td>)}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function parseMarkdownBlocks(content: string): MarkdownBlock[] {
    const lines = content.replace(/\r\n/g, '\n').split('\n');
    const blocks: MarkdownBlock[] = [];
    let index = 0;

    while (index < lines.length) {
        const line = lines[index];
        if (line.trim() === '') {
            index += 1;
            continue;
        }

        const fence = line.match(/^```(\S*)\s*$/);
        if (fence) {
            const language = fence[1] ?? '';
            const codeLines: string[] = [];
            index += 1;
            while (index < lines.length && !lines[index].startsWith('```')) {
                codeLines.push(lines[index]);
                index += 1;
            }
            if (index < lines.length) {
                index += 1;
            }
            blocks.push({type: 'code', language, content: codeLines.join('\n')});
            continue;
        }

        const heading = line.match(/^(#{1,4})\s+(.+)$/);
        if (heading) {
            blocks.push({type: 'heading', level: heading[1].length, text: heading[2].trim()});
            index += 1;
            continue;
        }

        if (isTableStart(lines, index)) {
            const rows: string[][] = [parseTableRow(lines[index])];
            index += 2;
            while (index < lines.length && isTableRow(lines[index])) {
                rows.push(parseTableRow(lines[index]));
                index += 1;
            }
            blocks.push({type: 'table', rows});
            continue;
        }

        const listMatch = line.match(/^(\s*)([-*]|\d+[.)])\s+(.+)$/);
        if (listMatch) {
            const ordered = /\d/.test(listMatch[2]);
            const items: string[] = [];
            while (index < lines.length) {
                const itemMatch = lines[index].match(/^(\s*)([-*]|\d+[.)])\s+(.+)$/);
                if (!itemMatch || /\d/.test(itemMatch[2]) !== ordered) {
                    break;
                }
                items.push(itemMatch[3].trim());
                index += 1;
            }
            blocks.push({type: 'list', ordered, items});
            continue;
        }

        const paragraphLines: string[] = [];
        while (index < lines.length && lines[index].trim() !== '' && !isSpecialBlockStart(lines, index)) {
            paragraphLines.push(lines[index].trim());
            index += 1;
        }
        if (paragraphLines.length > 0) {
            blocks.push({type: 'paragraph', lines: paragraphLines});
        }
    }

    return blocks;
}

function isSpecialBlockStart(lines: string[], index: number) {
    const line = lines[index];
    return /^```/.test(line) ||
        /^(#{1,4})\s+/.test(line) ||
        /^(\s*)([-*]|\d+[.)])\s+/.test(line) ||
        isTableStart(lines, index);
}

function isTableStart(lines: string[], index: number) {
    return index + 1 < lines.length && isTableRow(lines[index]) && isTableSeparator(lines[index + 1]);
}

function isTableRow(line: string) {
    return line.includes('|') && parseTableRow(line).length >= 2;
}

function isTableSeparator(line: string) {
    const cells = parseTableRow(line);
    return cells.length >= 2 && cells.every((cell) => /^:?-{3,}:?$/.test(cell.trim()));
}

function parseTableRow(line: string) {
    return line
        .trim()
        .replace(/^\|/, '')
        .replace(/\|$/, '')
        .split('|')
        .map((cell) => cell.trim());
}

function renderInline(value: string): ReactNode[] {
    const nodes: ReactNode[] = [];
    const pattern = /(`[^`]+`|\*\*[^*]+\*\*)/g;
    let cursor = 0;
    let match: RegExpExecArray | null;

    while ((match = pattern.exec(value)) !== null) {
        if (match.index > cursor) {
            nodes.push(value.slice(cursor, match.index));
        }

        const token = match[0];
        if (token.startsWith('`')) {
            nodes.push(<code key={`${match.index}-code`}>{token.slice(1, -1)}</code>);
        } else {
            nodes.push(<strong key={`${match.index}-strong`}>{token.slice(2, -2)}</strong>);
        }
        cursor = match.index + token.length;
    }

    if (cursor < value.length) {
        nodes.push(value.slice(cursor));
    }

    return nodes;
}

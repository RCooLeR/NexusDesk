import type {ReactNode} from 'react';

type HighlightedCodeProps = {
    content: string;
    fileName: string;
    searchQuery?: string;
};

type Token = {
    text: string;
    type: 'comment' | 'keyword' | 'number' | 'plain' | 'string';
};

const keywordByLanguage: Record<string, Set<string>> = {
    go: new Set(['break', 'case', 'chan', 'const', 'continue', 'defer', 'else', 'fallthrough', 'for', 'func', 'go', 'if', 'import', 'interface', 'map', 'package', 'range', 'return', 'select', 'struct', 'switch', 'type', 'var']),
    js: new Set(['async', 'await', 'break', 'case', 'catch', 'class', 'const', 'continue', 'default', 'else', 'export', 'extends', 'finally', 'for', 'from', 'function', 'if', 'import', 'let', 'new', 'return', 'switch', 'throw', 'try', 'var', 'while']),
    json: new Set(['false', 'null', 'true']),
    markdown: new Set([]),
    sql: new Set(['and', 'as', 'by', 'create', 'delete', 'from', 'group', 'insert', 'into', 'join', 'left', 'limit', 'not', 'null', 'on', 'or', 'order', 'right', 'select', 'table', 'update', 'values', 'where']),
};

export function HighlightedCode({content, fileName, searchQuery = ''}: HighlightedCodeProps) {
    const language = detectLanguage(fileName);
    const tokens = tokenize(content, language);

    return (
        <pre className={`highlighted-code language-${language}`}>
            {tokens.map((token, index) => (
                <span className={token.type === 'plain' ? undefined : `syntax-token syntax-${token.type}`} key={`${index}-${token.text}`}>
                    {renderTokenText(token.text, searchQuery)}
                </span>
            ))}
        </pre>
    );
}

function renderTokenText(text: string, searchQuery: string) {
    const query = searchQuery.trim();
    if (!query) {
        return text;
    }

    const queryIndex = text.toLowerCase().indexOf(query.toLowerCase());
    if (queryIndex === -1) {
        return text;
    }

    const parts: ReactNode[] = [];
    let cursor = 0;
    let matchIndex = queryIndex;
    while (matchIndex !== -1) {
        if (matchIndex > cursor) {
            parts.push(text.slice(cursor, matchIndex));
        }
        const matched = text.slice(matchIndex, matchIndex + query.length);
        parts.push(<mark className="find-highlight" key={`${matchIndex}-${matched}`}>{matched}</mark>);
        cursor = matchIndex + query.length;
        matchIndex = text.toLowerCase().indexOf(query.toLowerCase(), cursor);
    }

    if (cursor < text.length) {
        parts.push(text.slice(cursor));
    }

    return parts;
}

function tokenize(content: string, language: string) {
    const keywords = keywordByLanguage[language] ?? keywordByLanguage.js;
    const tokens: Token[] = [];
    const pattern = /(\/\/[^\n]*|#[^\n]*|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|`(?:\\.|[^`\\])*`|\b\d+(?:\.\d+)?\b|\b[A-Za-z_][A-Za-z0-9_]*\b)/g;
    let cursor = 0;

    for (const match of content.matchAll(pattern)) {
        const text = match[0];
        const index = match.index ?? 0;
        if (index > cursor) {
            tokens.push({text: content.slice(cursor, index), type: 'plain'});
        }

        tokens.push({text, type: classifyToken(text, keywords)});
        cursor = index + text.length;
    }

    if (cursor < content.length) {
        tokens.push({text: content.slice(cursor), type: 'plain'});
    }

    return tokens;
}

function classifyToken(text: string, keywords: Set<string>): Token['type'] {
    if (text.startsWith('//') || text.startsWith('#')) {
        return 'comment';
    }
    if (text.startsWith('"') || text.startsWith("'") || text.startsWith('`')) {
        return 'string';
    }
    if (/^\d/.test(text)) {
        return 'number';
    }
    if (keywords.has(text) || keywords.has(text.toLowerCase())) {
        return 'keyword';
    }
    return 'plain';
}

function detectLanguage(fileName: string) {
    const extension = fileName.split('.').pop()?.toLowerCase() ?? '';
    if (extension === 'go') {
        return 'go';
    }
    if (['js', 'jsx', 'ts', 'tsx'].includes(extension)) {
        return 'js';
    }
    if (extension === 'json') {
        return 'json';
    }
    if (['md', 'markdown'].includes(extension)) {
        return 'markdown';
    }
    if (extension === 'sql') {
        return 'sql';
    }
    return 'text';
}

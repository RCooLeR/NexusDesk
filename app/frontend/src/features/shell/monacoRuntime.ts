import type * as Monaco from 'monaco-editor/esm/vs/editor/editor.api';
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker';
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker';
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker';
import 'monaco-editor/min/vs/editor/editor.main.css';

type MonacoWorkerHost = typeof self & {
    MonacoEnvironment?: {
        getWorker: (_moduleId: string, label: string) => Worker;
    };
};

let themeDefined = false;
let monacoLoadPromise: Promise<typeof Monaco> | null = null;

(self as MonacoWorkerHost).MonacoEnvironment = {
    getWorker: (_moduleId: string, label: string) => {
        if (label === 'json') {
            return new jsonWorker();
        }
        if (label === 'css' || label === 'scss' || label === 'less') {
            return new cssWorker();
        }
        if (label === 'html' || label === 'handlebars' || label === 'razor') {
            return new htmlWorker();
        }
        if (label === 'typescript' || label === 'javascript') {
            return new tsWorker();
        }
        return new editorWorker();
    },
};

export function loadMonaco() {
    if (!monacoLoadPromise) {
        monacoLoadPromise = import('monaco-editor/esm/vs/editor/editor.api');
    }
    return monacoLoadPromise;
}

export function defineNexusDeskTheme(monaco: typeof Monaco) {
    if (themeDefined) {
        return;
    }

    monaco.editor.defineTheme('nexusdesk-light', {
        base: 'vs',
        inherit: true,
        rules: [
            {token: 'comment', foreground: '64748b'},
            {token: 'keyword', foreground: '1d4ed8', fontStyle: 'bold'},
            {token: 'number', foreground: 'b45309'},
            {token: 'string', foreground: '15803d'},
            {token: 'type', foreground: '7c3aed'},
        ],
        colors: {
            'editor.background': '#fbfdff',
            'editor.foreground': '#1f2937',
            'editor.lineHighlightBackground': '#eef6ff',
            'editorLineNumber.foreground': '#94a3b8',
            'editorLineNumber.activeForeground': '#0ea5e9',
            'editor.selectionBackground': '#bfdbfe',
            'editorCursor.foreground': '#0f172a',
        },
    });
    themeDefined = true;
}

export function languageForFile(fileName: string) {
    const lowerName = fileName.toLowerCase();
    if (/\.(ts|tsx)$/.test(lowerName)) {
        return 'typescript';
    }
    if (/\.(js|jsx|mjs|cjs)$/.test(lowerName)) {
        return 'javascript';
    }
    if (/\.jsonc?$/.test(lowerName) || lowerName.endsWith('.code-workspace')) {
        return 'json';
    }
    if (/\.(css|scss|less)$/.test(lowerName)) {
        return lowerName.endsWith('.css') ? 'css' : lowerName.endsWith('.scss') ? 'scss' : 'less';
    }
    if (/\.(html|htm|xml|svg)$/.test(lowerName)) {
        return 'html';
    }
    if (/\.mdx?$/.test(lowerName)) {
        return 'markdown';
    }
    if (/\.ya?ml$/.test(lowerName)) {
        return 'yaml';
    }
    if (/\.sql$/.test(lowerName)) {
        return 'sql';
    }
    if (/\.go$/.test(lowerName)) {
        return 'go';
    }
    if (/\.py$/.test(lowerName)) {
        return 'python';
    }
    if (/\.rs$/.test(lowerName)) {
        return 'rust';
    }
    if (/\.java$/.test(lowerName)) {
        return 'java';
    }
    if (/\.cs$/.test(lowerName)) {
        return 'csharp';
    }
    if (/\.ps1$/.test(lowerName)) {
        return 'powershell';
    }
    if (/(^|\/|\\)dockerfile$/.test(lowerName)) {
        return 'dockerfile';
    }
    return 'plaintext';
}

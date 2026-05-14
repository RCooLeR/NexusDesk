import {useEffect, useRef} from 'react';
import type * as Monaco from 'monaco-editor/esm/vs/editor/editor.api';
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker';
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker';
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker';
import 'monaco-editor/min/vs/editor/editor.main.css';

type MonacoFileEditorProps = {
    fileName: string;
    onChange: (content: string) => void;
    onSave: () => void;
    value: string;
};

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

export function MonacoFileEditor({fileName, onChange, onSave, value}: MonacoFileEditorProps) {
    const containerRef = useRef<HTMLDivElement>(null);
    const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);
    const changeHandlerRef = useRef(onChange);
    const saveHandlerRef = useRef(onSave);

    useEffect(() => {
        changeHandlerRef.current = onChange;
    }, [onChange]);

    useEffect(() => {
        saveHandlerRef.current = onSave;
    }, [onSave]);

    useEffect(() => {
        let disposed = false;
        let changeSubscription: Monaco.IDisposable | null = null;

        void loadMonaco().then((monaco) => {
            if (disposed || !containerRef.current) {
                return;
            }

            defineNexusDeskTheme(monaco);
            const editor = monaco.editor.create(containerRef.current, {
                automaticLayout: true,
                contextmenu: true,
                fontFamily: '"Cascadia Code", Consolas, monospace',
                fontLigatures: false,
                fontSize: 13,
                language: languageForFile(fileName),
                lineHeight: 21,
                minimap: {enabled: false},
                padding: {bottom: 16, top: 12},
                renderLineHighlight: 'line',
                scrollBeyondLastLine: false,
                tabSize: 4,
                theme: 'nexusdesk-light',
                value,
                wordWrap: 'on',
            });

            editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
                saveHandlerRef.current();
            });

            changeSubscription = editor.onDidChangeModelContent(() => {
                changeHandlerRef.current(editor.getValue());
            });

            editorRef.current = editor;
        });

        return () => {
            disposed = true;
            changeSubscription?.dispose();
            editorRef.current?.dispose();
            editorRef.current = null;
        };
    }, []);

    useEffect(() => {
        const editor = editorRef.current;
        if (!editor) {
            return;
        }

        void loadMonaco().then((monaco) => {
            const model = editor.getModel();
            if (!model) {
                return;
            }

            const language = languageForFile(fileName);
            if (model.getLanguageId() !== language) {
                monaco.editor.setModelLanguage(model, language);
            }
        });
    }, [fileName]);

    useEffect(() => {
        const editor = editorRef.current;
        const model = editor?.getModel();
        if (!editor || !model || model.getValue() === value) {
            return;
        }

        const selection = editor.getSelection();
        model.pushEditOperations([], [{range: model.getFullModelRange(), text: value}], () => null);
        if (selection) {
            editor.setSelection(selection);
        }
    }, [value]);

    return <div className="monaco-file-editor" ref={containerRef} />;
}

function loadMonaco() {
    if (!monacoLoadPromise) {
        monacoLoadPromise = import('monaco-editor/esm/vs/editor/editor.api');
    }
    return monacoLoadPromise;
}

function defineNexusDeskTheme(monaco: typeof Monaco) {
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

function languageForFile(fileName: string) {
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

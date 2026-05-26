import {useEffect, useRef} from 'react';
import type {MutableRefObject} from 'react';
import type * as Monaco from 'monaco-editor/esm/vs/editor/editor.api';
import {defineNexusTheme, languageForFile, loadMonaco} from './monacoRuntime';

type MonacoCodePreviewProps = {
    content: string;
    fileName: string;
    revealLine: number;
    revealNonce: number;
    searchQuery: string;
    showMinimap: boolean;
};

export function MonacoCodePreview({content, fileName, revealLine, revealNonce, searchQuery, showMinimap}: MonacoCodePreviewProps) {
    const containerRef = useRef<HTMLDivElement>(null);
    const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);
    const decorationIdsRef = useRef<string[]>([]);

    useEffect(() => {
        let disposed = false;

        void loadMonaco().then((monaco) => {
            if (disposed || !containerRef.current) {
                return;
            }

            defineNexusTheme(monaco);
            const editor = monaco.editor.create(containerRef.current, {
                automaticLayout: true,
                contextmenu: true,
                domReadOnly: true,
                fontFamily: '"Cascadia Code", Consolas, monospace',
                fontLigatures: false,
                fontSize: 13,
                language: languageForFile(fileName),
                lineHeight: 21,
                minimap: {enabled: showMinimap, maxColumn: 60, renderCharacters: false, scale: 0.7},
                padding: {bottom: 16, top: 12},
                readOnly: true,
                renderLineHighlight: 'none',
                scrollBeyondLastLine: false,
                tabSize: 4,
                theme: 'nexus-light',
                value: content,
                wordWrap: 'on',
            });

            editorRef.current = editor;
            updateSearchDecorations(monaco, editor, searchQuery, decorationIdsRef);
        });

        return () => {
            disposed = true;
            editorRef.current?.dispose();
            editorRef.current = null;
            decorationIdsRef.current = [];
        };
    }, []);

    useEffect(() => {
        editorRef.current?.updateOptions({
            minimap: {enabled: showMinimap, maxColumn: 60, renderCharacters: false, scale: 0.7},
        });
    }, [showMinimap]);

    useEffect(() => {
        const editor = editorRef.current;
        const model = editor?.getModel();
        if (!editor || !model || model.getValue() === content) {
            return;
        }

        model.setValue(content);
        decorationIdsRef.current = [];
    }, [content]);

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
        if (!editor) {
            return;
        }

        void loadMonaco().then((monaco) => {
            updateSearchDecorations(monaco, editor, searchQuery, decorationIdsRef);
        });
    }, [content, searchQuery]);

    useEffect(() => {
        const editor = editorRef.current;
        if (!editor || revealLine <= 0) {
            return;
        }
        editor.revealLineInCenter(revealLine);
        editor.setPosition({lineNumber: revealLine, column: 1});
        editor.focus();
    }, [revealLine, revealNonce]);

    return <div aria-label="Code preview editor" className="monaco-code-preview" ref={containerRef} />;
}

function updateSearchDecorations(
    monaco: typeof Monaco,
    editor: Monaco.editor.IStandaloneCodeEditor,
    searchQuery: string,
    decorationIdsRef: MutableRefObject<string[]>
) {
    const model = editor.getModel();
    const needle = searchQuery.trim();
    if (!model || !needle) {
        decorationIdsRef.current = editor.deltaDecorations(decorationIdsRef.current, []);
        return;
    }

    const ranges = findSearchRanges(monaco, model, needle);
    decorationIdsRef.current = editor.deltaDecorations(decorationIdsRef.current, ranges.map((range) => ({
        range,
        options: {
            className: 'monaco-find-highlight',
            overviewRuler: {
                color: '#facc15',
                position: monaco.editor.OverviewRulerLane.Center,
            },
        },
    })));
}

function findSearchRanges(monaco: typeof Monaco, model: Monaco.editor.ITextModel, query: string) {
    const content = model.getValue();
    const haystack = content.toLowerCase();
    const needle = query.toLowerCase();
    const ranges: Monaco.Range[] = [];
    let cursor = 0;

    while (cursor <= haystack.length && ranges.length < 2000) {
        const index = haystack.indexOf(needle, cursor);
        if (index === -1) {
            break;
        }

        const start = model.getPositionAt(index);
        const end = model.getPositionAt(index + needle.length);
        ranges.push(new monaco.Range(start.lineNumber, start.column, end.lineNumber, end.column));
        cursor = index + Math.max(needle.length, 1);
    }

    return ranges;
}

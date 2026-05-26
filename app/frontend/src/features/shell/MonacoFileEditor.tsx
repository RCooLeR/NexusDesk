import {useEffect, useRef} from 'react';
import type * as Monaco from 'monaco-editor/esm/vs/editor/editor.api';
import {defineNexusTheme, languageForFile, loadMonaco} from './monacoRuntime';

type MonacoFileEditorProps = {
    fileName: string;
    onChange: (content: string) => void;
    onSave: () => void;
    showMinimap: boolean;
    value: string;
};

export function MonacoFileEditor({fileName, onChange, onSave, showMinimap, value}: MonacoFileEditorProps) {
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

            defineNexusTheme(monaco);
            const editor = monaco.editor.create(containerRef.current, {
                automaticLayout: true,
                contextmenu: true,
                fontFamily: '"Cascadia Code", Consolas, monospace',
                fontLigatures: false,
                fontSize: 13,
                language: languageForFile(fileName),
                lineHeight: 21,
                minimap: {enabled: showMinimap, maxColumn: 60, renderCharacters: false, scale: 0.7},
                padding: {bottom: 16, top: 12},
                renderLineHighlight: 'line',
                scrollBeyondLastLine: false,
                tabSize: 4,
                theme: 'nexus-light',
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
        editorRef.current?.updateOptions({
            minimap: {enabled: showMinimap, maxColumn: 60, renderCharacters: false, scale: 0.7},
        });
    }, [showMinimap]);

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

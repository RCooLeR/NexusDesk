import {useEffect, useState} from 'react';
import {ApplyGitFileAction, ApplyGitHunkAction, GetGitFileDiff, GetGitStatus, PreviewGitFileAction, PreviewGitHunkAction} from '../../api/wailsClient';
import type {GitFileAction, GitFileActionPreview, GitFileDiff, GitHunkActionPreview, GitHunkActionRequest, GitStatus} from '../../types';

const gitNotLoadedMessage = 'Git status not loaded. Press Refresh git when you need repository status.';

export function useGitController(workspaceRoot: string | undefined, pushToolEvent: (title: string, detail: string) => void) {
    const [gitStatus, setGitStatus] = useState<GitStatus | null>(null);
    const [selectedGitChangePath, setSelectedGitChangePath] = useState('');
    const [selectedGitFileDiff, setSelectedGitFileDiff] = useState<GitFileDiff | null>(null);
    const [gitFileActionPreview, setGitFileActionPreview] = useState<GitFileActionPreview | null>(null);
    const [gitHunkActionPreview, setGitHunkActionPreview] = useState<GitHunkActionPreview | null>(null);
    const [isLoadingGitFileDiff, setIsLoadingGitFileDiff] = useState(false);
    const [isApplyingGitFileAction, setIsApplyingGitFileAction] = useState(false);
    const [isPreviewingGitFileAction, setIsPreviewingGitFileAction] = useState(false);
    const [isPreviewingGitHunkAction, setIsPreviewingGitHunkAction] = useState(false);
    const [isApplyingGitHunkAction, setIsApplyingGitHunkAction] = useState(false);

    useEffect(() => {
        resetGitStatus();
    }, [workspaceRoot]);

    useEffect(() => {
        const changes = gitStatus?.changedFiles ?? [];
        if (changes.length === 0) {
            setSelectedGitChangePath('');
            setSelectedGitFileDiff(null);
            return;
        }
        if (!selectedGitChangePath || !changes.some((change) => change.path === selectedGitChangePath)) {
            setSelectedGitChangePath(changes[0].path);
        }
    }, [gitStatus?.generatedAt, gitStatus?.changedFiles, selectedGitChangePath]);

    useEffect(() => {
        if (!selectedGitChangePath || !gitStatus?.available) {
            setSelectedGitFileDiff(null);
            setGitFileActionPreview(null);
            setGitHunkActionPreview(null);
            return;
        }
        setGitFileActionPreview(null);
        setGitHunkActionPreview(null);
        void refreshSelectedGitFileDiff(selectedGitChangePath);
    }, [gitStatus?.generatedAt, selectedGitChangePath, gitStatus?.available]);

    function resetGitStatus() {
        setGitStatus(workspaceRoot ? emptyGitStatus(gitNotLoadedMessage) : null);
        setSelectedGitChangePath('');
        setSelectedGitFileDiff(null);
        setGitFileActionPreview(null);
        setGitHunkActionPreview(null);
        setIsLoadingGitFileDiff(false);
        setIsApplyingGitFileAction(false);
        setIsPreviewingGitFileAction(false);
        setIsPreviewingGitHunkAction(false);
        setIsApplyingGitHunkAction(false);
    }

    async function refreshGitStatus() {
        try {
            const status = await GetGitStatus();
            const safeStatus = normalizeGitStatus(status);
            setGitStatus(safeStatus);
            if (safeStatus.available) {
                pushToolEvent('Git status refreshed', safeStatus.message);
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setGitStatus(emptyGitStatus(message || 'Git status is unavailable.'));
        }
    }

    async function refreshSelectedGitFileDiff(path: string) {
        setIsLoadingGitFileDiff(true);
        try {
            const diff = await GetGitFileDiff(path);
            setSelectedGitFileDiff(diff);
            if (diff.message) {
                pushToolEvent('Git file diff loaded', diff.message);
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setSelectedGitFileDiff({
                path,
                stagedDiff: '',
                stagedDiffTruncated: false,
                unstagedDiff: '',
                unstagedDiffTruncated: false,
                message: message || 'Git file diff is unavailable.',
                generatedAt: '',
            });
        } finally {
            setIsLoadingGitFileDiff(false);
        }
    }

    async function previewGitFileAction(action: GitFileAction) {
        if (!selectedGitChangePath) {
            return;
        }
        setIsPreviewingGitFileAction(true);
        try {
            const preview = await PreviewGitFileAction({path: selectedGitChangePath, action});
            const safePreview = normalizeGitFileActionPreview(preview, action, selectedGitChangePath);
            setGitFileActionPreview(safePreview);
            if (safePreview.status?.generatedAt) {
                setGitStatus(normalizeGitStatus(safePreview.status));
            }
            pushToolEvent('Git action preview ready', safePreview.message);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setGitFileActionPreview(emptyGitFileActionPreview(action, selectedGitChangePath, message || 'Git action preview is unavailable.'));
        } finally {
            setIsPreviewingGitFileAction(false);
        }
    }

    async function applyGitFileAction(action: GitFileAction) {
        if (!selectedGitChangePath) {
            return null;
        }
        setIsApplyingGitFileAction(true);
        try {
            const preview = await ApplyGitFileAction({path: selectedGitChangePath, action});
            const safePreview = normalizeGitFileActionPreview(preview, action, selectedGitChangePath);
            setGitFileActionPreview(safePreview);
            if (safePreview.status?.generatedAt) {
                setGitStatus(normalizeGitStatus(safePreview.status));
            }
            await refreshSelectedGitFileDiff(selectedGitChangePath);
            pushToolEvent('Git file action applied', safePreview.message);
            return safePreview;
        } finally {
            setIsApplyingGitFileAction(false);
        }
    }

    async function previewGitHunkAction(request: GitHunkActionRequest) {
        setIsPreviewingGitHunkAction(true);
        try {
            const preview = await PreviewGitHunkAction(request);
            const safePreview = normalizeGitHunkActionPreview(preview, request);
            setGitHunkActionPreview(safePreview);
            if (safePreview.status?.generatedAt) {
                setGitStatus(normalizeGitStatus(safePreview.status));
            }
            pushToolEvent('Git hunk preview ready', safePreview.message);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setGitHunkActionPreview(emptyGitHunkActionPreview(request, message || 'Git hunk action preview is unavailable.'));
        } finally {
            setIsPreviewingGitHunkAction(false);
        }
    }

    async function applyGitHunkAction(request: GitHunkActionRequest) {
        setIsApplyingGitHunkAction(true);
        try {
            const preview = await ApplyGitHunkAction(request);
            const safePreview = normalizeGitHunkActionPreview(preview, request);
            setGitHunkActionPreview(safePreview);
            if (safePreview.status?.generatedAt) {
                setGitStatus(normalizeGitStatus(safePreview.status));
            }
            await refreshSelectedGitFileDiff(request.path);
            pushToolEvent('Git hunk action applied', safePreview.message);
            return safePreview;
        } finally {
            setIsApplyingGitHunkAction(false);
        }
    }

    return {
        gitFileActionPreview,
        gitHunkActionPreview,
        gitStatus,
        isApplyingGitFileAction,
        isApplyingGitHunkAction,
        isLoadingGitFileDiff,
        isPreviewingGitFileAction,
        isPreviewingGitHunkAction,
        applyGitFileAction,
        applyGitHunkAction,
        previewGitFileAction,
        previewGitHunkAction,
        refreshGitStatus,
        resetGitStatus,
        selectedGitChangePath,
        selectedGitFileDiff,
        selectGitChange: setSelectedGitChangePath,
    };
}

function normalizeGitHunkActionPreview(
    preview: GitHunkActionPreview | null | undefined,
    request: GitHunkActionRequest
): GitHunkActionPreview {
    if (!preview) {
        return emptyGitHunkActionPreview(request, 'Git hunk action preview is unavailable.');
    }
    return {
        ...preview,
        action: preview.action || request.action,
        diffKind: preview.diffKind || request.diffKind,
        hunkIndex: preview.hunkIndex || request.hunkIndex,
        path: preview.path || request.path,
        command: Array.isArray(preview.command) ? preview.command : [],
        patch: typeof preview.patch === 'string' ? preview.patch : '',
        status: normalizeGitStatus(preview.status),
    };
}

function emptyGitHunkActionPreview(request: GitHunkActionRequest, message: string): GitHunkActionPreview {
    return {
        path: request.path,
        action: request.action,
        diffKind: request.diffKind,
        hunkIndex: request.hunkIndex,
        command: [],
        patch: '',
        requiresApproval: true,
        mutatesRepository: true,
        message,
        status: emptyGitStatus(message),
        generatedAt: '',
    };
}

function normalizeGitFileActionPreview(
    preview: GitFileActionPreview | null | undefined,
    action: GitFileAction,
    path: string
): GitFileActionPreview {
    if (!preview) {
        return emptyGitFileActionPreview(action, path, 'Git action preview is unavailable.');
    }
    return {
        ...preview,
        action: preview.action || action,
        path: preview.path || path,
        command: Array.isArray(preview.command) ? preview.command : [],
        status: normalizeGitStatus(preview.status),
    };
}

function emptyGitFileActionPreview(action: GitFileAction, path: string, message: string): GitFileActionPreview {
    return {
        path,
        action,
        command: [],
        requiresApproval: true,
        mutatesRepository: true,
        message,
        status: emptyGitStatus(message),
        generatedAt: '',
    };
}

function normalizeGitStatus(status: GitStatus | null | undefined): GitStatus {
    if (!status) {
        return emptyGitStatus('Git status is unavailable.');
    }
    return {
        ...status,
        changedFiles: Array.isArray(status.changedFiles) ? status.changedFiles : [],
        stagedFiles: Array.isArray(status.stagedFiles) ? status.stagedFiles : [],
        unstagedFiles: Array.isArray(status.unstagedFiles) ? status.unstagedFiles : [],
        diff: typeof status.diff === 'string' ? status.diff : '',
        stagedDiff: typeof status.stagedDiff === 'string' ? status.stagedDiff : '',
        unstagedDiff: typeof status.unstagedDiff === 'string' ? status.unstagedDiff : '',
    };
}

function emptyGitStatus(message: string): GitStatus {
    return {
        available: false,
        repoRoot: '',
        branch: '',
        head: '',
        dirty: false,
        changedFiles: [],
        stagedFiles: [],
        unstagedFiles: [],
        diff: '',
        diffTruncated: false,
        stagedDiff: '',
        stagedDiffTruncated: false,
        unstagedDiff: '',
        unstagedDiffTruncated: false,
        aheadBehind: '',
        message,
        generatedAt: '',
    };
}

type GenerateTestsPromptInput = {
    contextRelPath: string;
    diffContext: string;
};

export function buildGenerateTestsPrompt({contextRelPath, diffContext}: GenerateTestsPromptInput) {
    if (contextRelPath) {
        return [
            `Generate a focused test plan and test code suggestions for ${contextRelPath}.`,
            'Return concise Markdown with: coverage gaps, concrete test cases, suggested test file paths, and code snippets where useful.',
            'Do not claim tests were created or run. Keep suggestions grounded in the selected file.',
        ].join(' ');
    }

    return [
        'Generate focused test suggestions for this git diff.',
        'Return concise Markdown with: changed behavior, missing test coverage, concrete test cases, suggested test file paths, and code snippets where useful.',
        'Do not claim tests were created or run. Stay grounded in the diff.',
        '',
        diffContext,
    ].join('\n');
}

export function buildPatchProposalPrompt(contextRelPath: string) {
    return [
        `Propose a minimal patch for ${contextRelPath}.`,
        'Return Markdown with a short rationale and exactly one unified diff for this file.',
        'Do not say the patch was applied. The user will review it and apply it through the file-write preview boundary.',
        'Keep the patch small, reversible, and grounded in the selected file context.',
    ].join(' ');
}

export function buildDependencyGraphPrompt(contextRelPath: string) {
    return [
        contextRelPath === '.' ? 'Explain the project dependency graph from the bounded workspace context.' : `Explain the dependency graph around ${contextRelPath}.`,
        'Return concise Markdown with inbound dependencies, outbound dependencies, runtime/build/test dependencies, missing context, risks, and useful next inspection steps.',
        'Use citations from the provided context and do not invent files outside it.',
    ].join(' ');
}

export function buildPullRequestSummaryPrompt(diffContext: string) {
    return [
        'Draft a concise pull request summary from this git diff.',
        'Return Markdown with: summary, notable changes, risk areas, tests to run, and follow-up items.',
        'Do not claim tests passed unless the diff says so. Stay grounded in the diff.',
        '',
        diffContext,
    ].join('\n');
}

export function buildPullRequestDescriptionPrompt(diffContext: string) {
    return [
        'Draft a complete pull request description from this git diff.',
        'Use Markdown sections: Summary, Changes, Validation, Risks, Rollback, and Notes for reviewers.',
        'Do not claim validation was run unless the diff or prompt states it. Stay grounded in the diff.',
        '',
        diffContext,
    ].join('\n');
}

export function applyUnifiedDiffToContent(original: string, assistantAnswer: string, relPath: string) {
    const patchLines = unifiedDiffLinesForPath(assistantAnswer, relPath);
    if (!patchLines || !patchLines.some((line) => line.startsWith('@@ '))) {
        return null;
    }

    const normalized = original.replace(/\r\n/g, '\n');
    const hadFinalNewline = normalized.endsWith('\n');
    const originalLines = hadFinalNewline ? normalized.slice(0, -1).split('\n') : normalized.split('\n');
    const result: string[] = [];
    let originalIndex = 0;
    let appliedHunks = 0;

    for (let index = 0; index < patchLines.length; index += 1) {
        const header = /^@@ -(\d+)(?:,\d+)? \+\d+(?:,\d+)? @@/.exec(patchLines[index]);
        if (!header) {
            continue;
        }

        const hunkOldStart = Number(header[1]) - 1;
        if (!Number.isFinite(hunkOldStart) || hunkOldStart < originalIndex || hunkOldStart > originalLines.length) {
            return null;
        }
        while (originalIndex < hunkOldStart) {
            result.push(originalLines[originalIndex]);
            originalIndex += 1;
        }

        index += 1;
        for (; index < patchLines.length; index += 1) {
            const line = patchLines[index];
            if (line.startsWith('@@ ')) {
                index -= 1;
                break;
            }
            if (line.startsWith('diff --git ')) {
                continue;
            }
            if (line.startsWith('\\')) {
                continue;
            }
            if (line.startsWith(' ')) {
                if (originalLines[originalIndex] !== line.slice(1)) {
                    return null;
                }
                result.push(line.slice(1));
                originalIndex += 1;
                continue;
            }
            if (line.startsWith('-')) {
                if (originalLines[originalIndex] !== line.slice(1)) {
                    return null;
                }
                originalIndex += 1;
                continue;
            }
            if (line.startsWith('+')) {
                result.push(line.slice(1));
            }
        }
        appliedHunks += 1;
    }

    if (appliedHunks === 0) {
        return null;
    }
    while (originalIndex < originalLines.length) {
        result.push(originalLines[originalIndex]);
        originalIndex += 1;
    }
    return `${result.join('\n')}${hadFinalNewline ? '\n' : ''}`;
}

function unifiedDiffLinesForPath(answer: string, relPath: string) {
    const lines = stripMarkdownCodeFenceMarkers(answer.replace(/\r\n/g, '\n').split('\n'));
    const sections = splitUnifiedDiffSections(lines);
    const matchingSection = sections.find((section) => section.some((line) => line.startsWith('@@ ')) && sectionMatchesRelPath(section, relPath));
    if (matchingSection) {
        return matchingSection;
    }
    if (sections.length === 1 && sections[0].some((line) => line.startsWith('@@ '))) {
        return sections[0];
    }
    if (sections.length === 0 && lines.some((line) => line.startsWith('@@ '))) {
        return lines;
    }
    return null;
}

function stripMarkdownCodeFenceMarkers(lines: string[]) {
    return lines.filter((line) => !line.trim().startsWith('```'));
}

function splitUnifiedDiffSections(lines: string[]) {
    const sections: string[][] = [];
    let current: string[] = [];
    for (const line of lines) {
        const startsSection = line.startsWith('diff --git ') || (line.startsWith('--- ') && current.some((currentLine) => currentLine.startsWith('@@ ')));
        if (startsSection && current.length > 0) {
            sections.push(current);
            current = [];
        }
        if (line.startsWith('diff --git ') || line.startsWith('--- ') || line.startsWith('+++ ') || line.startsWith('@@ ') || /^[ +\-\\]/.test(line)) {
            current.push(line);
        }
    }
    if (current.length > 0) {
        sections.push(current);
    }
    return sections;
}

function sectionMatchesRelPath(section: string[], relPath: string) {
    const normalized = relPath.replace(/\\/g, '/');
    return section.some((line) => {
        if (!line.startsWith('diff --git ') && !line.startsWith('--- ') && !line.startsWith('+++ ')) {
            return false;
        }
        return line.includes(`a/${normalized}`) || line.includes(`b/${normalized}`) || line.includes(normalized);
    });
}

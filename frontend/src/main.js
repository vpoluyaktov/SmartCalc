import './style.css';
import { EditorView, basicSetup } from 'codemirror';
import { EditorState, RangeSetBuilder } from '@codemirror/state';
import { keymap, Decoration, ViewPlugin } from '@codemirror/view';
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
import { lineNumbers, highlightActiveLineGutter, highlightActiveLine } from '@codemirror/view';
import { Evaluate, EvaluateParallel, GetVersion, OpenFileDialog, SaveFileDialog, ReadFile, WriteFile, AddRecentFile, GetLastFile, AutoSave, AdjustReferences, CopyWithResolvedRefs, SetUnsavedState, Quit, StripLineResult, HasLineResult, EvaluateLines, StripAllResults, StripAndEvalReferencingLines, GetGitHubRepoURL, CheckForUpdates, OpenURL } from '../wailsjs/go/main/App';
import { EventsOn, ClipboardGetText, ClipboardSetText } from '../wailsjs/runtime/runtime';

let editor;
let modalEditor = null; // Editor instance for modal dialogs
let currentFile = '';
let debounceTimer = null;
let autosaveTimer = null;
let previousText = '';
let previousLineCount = 0;
let isUpdatingEditor = false; // Flag to prevent re-entry during programmatic updates
let savedContent = ''; // Content at last save, to detect unsaved changes
let lastActiveLine = 1; // Track the last active line number (1-based)
let pendingEvaluation = false; // Track if we need to evaluate when leaving line
const AUTOSAVE_DELAY = 2000; // 2 seconds after last change

// Dark theme (Tokyo Night inspired)
const darkTheme = EditorView.theme({
    '&': {
        backgroundColor: '#1a1b26',
        color: '#c0caf5',
    },
    '.cm-content': {
        caretColor: '#7aa2f7',
    },
    '.cm-cursor': {
        borderLeftColor: '#7aa2f7',
    },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
        backgroundColor: 'rgba(122, 162, 247, 0.3)',
    },
    '.cm-gutters': {
        backgroundColor: '#16161e',
        color: '#565f89',
        borderRight: '1px solid #3b4261',
    },
    '.cm-activeLineGutter': {
        backgroundColor: '#292e42',
        color: '#7aa2f7',
    },
    '.cm-activeLine': {
        backgroundColor: 'rgba(41, 46, 66, 0.5)',
    },
}, { dark: true });

// Light theme
const lightTheme = EditorView.theme({
    '&': {
        backgroundColor: '#f8f9fa',
        color: '#343a40',
    },
    '.cm-content': {
        caretColor: '#4285f4',
    },
    '.cm-cursor': {
        borderLeftColor: '#4285f4',
    },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
        backgroundColor: 'rgba(66, 133, 244, 0.25)',
    },
    '.cm-gutters': {
        backgroundColor: '#f1f3f4',
        color: '#868e96',
        borderRight: '1px solid #dee2e6',
    },
    '.cm-activeLineGutter': {
        backgroundColor: '#e9ecef',
        color: '#495057',
    },
    '.cm-activeLine': {
        backgroundColor: 'rgba(0, 0, 0, 0.04)',
    },
}, { dark: false });

// Detect system theme preference
function getSystemTheme() {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

// Get the appropriate theme based on system preference
function getCurrentTheme() {
    return getSystemTheme() === 'dark' ? darkTheme : lightTheme;
}

// Syntax highlighting decorations
const resultMark = Decoration.mark({ class: 'cm-result' });
const errorMark = Decoration.mark({ class: 'cm-error' });
const referenceMark = Decoration.mark({ class: 'cm-reference' });
const numberMark = Decoration.mark({ class: 'cm-number' });
const operatorMark = Decoration.mark({ class: 'cm-operator' });
const currencyMark = Decoration.mark({ class: 'cm-currency' });
const percentMark = Decoration.mark({ class: 'cm-percent' });
const functionMark = Decoration.mark({ class: 'cm-function' });
const keywordMark = Decoration.mark({ class: 'cm-keyword' });
const datetimeMark = Decoration.mark({ class: 'cm-datetime' });
const networkMark = Decoration.mark({ class: 'cm-network' });
const commentMark = Decoration.mark({ class: 'cm-comment' });
const outputPrefixMark = Decoration.mark({ class: 'cm-output-prefix' });
const regexMatchMark = Decoration.mark({ class: 'cm-regex-match' });
const regexMarkerMark = Decoration.mark({ class: 'cm-regex-marker' });

// Helper function to highlight regex matches marked with «» in output lines
function highlightRegexMatches(builder, text, from) {
    let pos = 0;
    let inMatch = false;
    let matchStart = 0;
    
    while (pos < text.length) {
        if (text[pos] === '«') {
            // Mark the « character
            builder.add(from + pos, from + pos + 1, regexMarkerMark);
            inMatch = true;
            matchStart = pos + 1;
            pos++;
        } else if (text[pos] === '»' && inMatch) {
            // Highlight the match content
            if (matchStart < pos) {
                builder.add(from + matchStart, from + pos, regexMatchMark);
            }
            // Mark the » character
            builder.add(from + pos, from + pos + 1, regexMarkerMark);
            inMatch = false;
            pos++;
        } else {
            pos++;
        }
    }
}

function buildDecorations(view) {
    const builder = new RangeSetBuilder();
    const doc = view.state.doc;
    
    for (let i = 1; i <= doc.lines; i++) {
        const line = doc.line(i);
        const text = line.text;
        const from = line.from;
        
        // Output lines starting with "> " or just ">"
        if (text.startsWith('> ') || text.trim() === '>') {
            builder.add(from, from + Math.min(2, text.length), outputPrefixMark);
            // Check for errors in output
            if (text.includes('ERR:') || text.includes('error')) {
                builder.add(from + 2, line.to, errorMark);
            } else if (text.length > 2) {
                // Check for regex match markers «» and highlight them
                if (text.includes('«') && text.includes('»')) {
                    highlightRegexMatches(builder, text, from);
                } else {
                    builder.add(from + 2, line.to, resultMark);
                }
            }
            continue;
        }
        
        // Comment lines starting with # or //
        if (text.trim().startsWith('#') || text.trim().startsWith('//')) {
            builder.add(from, line.to, commentMark);
            continue;
        }
        
        // Check for inline comment (# after =)
        const eqIndex = text.indexOf('=');
        const hashIndex = text.indexOf('#');
        const hasInlineComment = eqIndex >= 0 && hashIndex > eqIndex;
        
        // Process tokens in the line (up to inline comment if present)
        const tokenEnd = hasInlineComment ? hashIndex : text.length;
        let pos = 0;
        while (pos < tokenEnd) {
            const remaining = text.slice(pos, tokenEnd);
            let matched = false;
            
            // Regex match markers «» for highlighting
            if (remaining[0] === '«') {
                builder.add(from + pos, from + pos + 1, regexMarkerMark);
                pos += 1;
                matched = true;
                continue;
            }
            if (remaining[0] === '»') {
                builder.add(from + pos, from + pos + 1, regexMarkerMark);
                pos += 1;
                matched = true;
                continue;
            }
            
            // Check if we're inside a regex match (between « and »)
            const openMarker = text.lastIndexOf('«', pos);
            const closeMarker = text.lastIndexOf('»', pos);
            const inRegexMatch = openMarker >= 0 && (closeMarker < openMarker);
            
            // Line references \1, \2, etc.
            const refMatch = remaining.match(/^\\[0-9]+/);
            if (refMatch) {
                builder.add(from + pos, from + pos + refMatch[0].length, referenceMark);
                pos += refMatch[0].length;
                matched = true;
                continue;
            }
            
            // Currency with amount $1,234.56
            const currencyMatch = remaining.match(/^\$[\d,]+\.?\d*/);
            if (currencyMatch) {
                builder.add(from + pos, from + pos + currencyMatch[0].length, currencyMark);
                pos += currencyMatch[0].length;
                matched = true;
                continue;
            }
            
            // IP addresses and CIDR notation
            const ipMatch = remaining.match(/^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(\/\d{1,2})?/);
            if (ipMatch) {
                builder.add(from + pos, from + pos + ipMatch[0].length, networkMark);
                pos += ipMatch[0].length;
                matched = true;
                continue;
            }
            
            // Time patterns like 14:30, 6:00 am
            const timeMatch = remaining.match(/^\d{1,2}:\d{2}(:\d{2})?(\s*(am|pm|AM|PM))?/);
            if (timeMatch) {
                builder.add(from + pos, from + pos + timeMatch[0].length, datetimeMark);
                pos += timeMatch[0].length;
                matched = true;
                continue;
            }
            
            // Numbers (including decimals and negative)
            const numMatch = remaining.match(/^-?\d+\.?\d*/);
            if (numMatch) {
                builder.add(from + pos, from + pos + numMatch[0].length, numberMark);
                pos += numMatch[0].length;
                matched = true;
                continue;
            }
            
            // Percentage
            if (remaining[0] === '%') {
                builder.add(from + pos, from + pos + 1, percentMark);
                pos += 1;
                matched = true;
                continue;
            }
            
            // Operators
            if (['+', '-', '*', '/', '^', '×', '÷', '='].includes(remaining[0])) {
                builder.add(from + pos, from + pos + 1, operatorMark);
                pos += 1;
                matched = true;
                continue;
            }
            
            // Functions
            const funcMatch = remaining.match(/^(sin|cos|tan|sqrt|abs|log|ln|exp|floor|ceil|round|min|max|avg|average|mean|median|sum|stddev|stdev|variance|var|count|range)\s*\(/i);
            if (funcMatch) {
                builder.add(from + pos, from + pos + funcMatch[1].length, functionMark);
                pos += funcMatch[1].length;
                matched = true;
                continue;
            }
            
            // Keywords
            const kwMatch = remaining.match(/^(jwt|cert|ssl|chmod|umask|permissions?|symbolic|octal|setuid|setgid|sticky|regex|test|match|against|now|today|yesterday|tomorrow|in|to|till|from|split|subnets?|networks?|hosts?|mask|wildcard|how\s+many|is|Range|Broadcast|what|percent|percentage|increase|decrease|tip|loan|mortgage|compound|simple|interest|invest|avg|average|mean|median|sum|stddev|stdev|variance|count|range|ascii|char|uuid|md5|sha1|sha256|base64|encode|decode|random|and|or|xor|not|speed\s+of\s+light|gravity|pi|avogadro|planck|golden\s+ratio|value\s+of|dig|nslookup|dns|lookup|resolve|whois)\b/i);
            if (kwMatch) {
                builder.add(from + pos, from + pos + kwMatch[0].length, keywordMark);
                pos += kwMatch[0].length;
                matched = true;
                continue;
            }
            
            // Date keywords
            const dateKwMatch = remaining.match(/^(days?|weeks?|months?|years?|hours?|minutes?|seconds?|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|January|February|March|April|June|July|August|September|October|November|December|Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\b/i);
            if (dateKwMatch) {
                builder.add(from + pos, from + pos + dateKwMatch[0].length, datetimeMark);
                pos += dateKwMatch[0].length;
                matched = true;
                continue;
            }
            
            if (!matched) {
                pos++;
            }
        }
        
        // Add inline comment decoration after all tokens (must be in order)
        if (hasInlineComment) {
            builder.add(from + hashIndex, line.to, commentMark);
        }
    }
    
    return builder.finish();
}

const syntaxHighlighter = ViewPlugin.fromClass(class {
    constructor(view) {
        this.decorations = buildDecorations(view);
    }
    update(update) {
        if (update.docChanged || update.viewportChanged) {
            this.decorations = buildDecorations(update.view);
        }
    }
}, {
    decorations: v => v.decorations
});

// Find the position of the result '=' in a line (not comparison operators)
function findResultEqualsPos(lineText) {
    for (let i = lineText.length - 1; i >= 0; i--) {
        if (lineText[i] === '=') {
            // Check if this '=' is part of >=, <=, ==, or !=
            if (i > 0) {
                const prev = lineText[i - 1];
                if (prev === '>' || prev === '<' || prev === '=' || prev === '!') {
                    continue; // Skip this '=', it's part of a comparison operator
                }
            }
            return i;
        }
    }
    return -1;
}

// Custom Enter key handler - handles evaluation and line splitting
// Must be synchronous for CodeMirror keymap to work correctly
function handleEnterKey(view) {
    const state = view.state;
    const pos = state.selection.main.head;
    const line = state.doc.lineAt(pos);
    const lineText = line.text;
    const lineNumber = line.number;
    const cursorColumn = pos - line.from;
    
    // Skip empty lines and comments - just insert newline
    if (lineText.trim().length === 0 || 
        lineText.trim().startsWith('#') || 
        lineText.trim().startsWith('//')) {
        return false; // Let default handler insert newline
    }
    
    const eqPos = findResultEqualsPos(lineText);
    
    // Case 1: No '=' in line - add '=' and evaluate, then add newline
    if (eqPos < 0) {
        // Add ' =' at end first
        view.dispatch({
            changes: { from: line.to, insert: ' =' },
        });
        
        // Then evaluate and add newline asynchronously
        handleEnterKeyAsync(view, lineNumber, true);
        return true;
    }
    
    // Case 2: Cursor is before or at '=' - split the line
    if (cursorColumn <= eqPos) {
        // Split line at cursor position like a regular text editor
        return false; // Let default handler do the split
    }
    
    // Case 3: Cursor is after '=' - evaluate current line and go to new line
    handleEnterKeyAsync(view, lineNumber, false);
    return true;
}

// Async helper for Enter key - evaluates and adds newline
async function handleEnterKeyAsync(view, lineNumber, alreadyAddedEquals) {
    isUpdatingEditor = true;
    try {
        // Evaluate the line
        await evaluateAndUpdate(lineNumber);
        
        // Insert newline after the updated line (and any multi-line output lines)
        const newDoc = view.state.doc;
        let lastLine = newDoc.line(lineNumber);
        // Skip past any output lines (starting with ">") that follow the expression
        while (lastLine.number < newDoc.lines) {
            const nextLine = newDoc.line(lastLine.number + 1);
            if (nextLine.text.startsWith('>')) {
                lastLine = nextLine;
            } else {
                break;
            }
        }
        view.dispatch({
            changes: { from: lastLine.to, insert: '\n' },
            selection: { anchor: lastLine.to + 1 },
        });
    } finally {
        isUpdatingEditor = false;
    }
}

// Evaluate a specific line and update the editor
async function evaluateAndUpdate(lineNumber) {
    const text = editor.state.doc.toString();
    try {
        // First, add progress indicator to the line being evaluated
        const lines = text.split('\n');
        const lineIdx = lineNumber - 1; // Convert to 0-based
        if (lineIdx >= 0 && lineIdx < lines.length) {
            const line = lines[lineIdx];
            const trimmed = line.trim();
            // Add progress indicator if line ends with = (and it's not a comparison operator)
            if (trimmed.endsWith('=') && !trimmed.endsWith('==') && !trimmed.endsWith('!=') && 
                !trimmed.endsWith('<=') && !trimmed.endsWith('>=')) {
                lines[lineIdx] = line + ' ⏳';
                const progressText = lines.join('\n');
                
                // Show progress indicator immediately
                const scrollTop = editor.scrollDOM.scrollTop;
                const scrollLeft = editor.scrollDOM.scrollLeft;
                
                editor.dispatch({
                    changes: { from: 0, to: editor.state.doc.length, insert: progressText },
                });
                
                // Restore scroll and force a render
                requestAnimationFrame(() => {
                    editor.scrollDOM.scrollTop = scrollTop;
                    editor.scrollDOM.scrollLeft = scrollLeft;
                });
                
                // Small delay to ensure progress indicator is visible
                await new Promise(resolve => setTimeout(resolve, 10));
            }
        }
        
        // Now evaluate the line
        const results = await EvaluateLines(text, lineNumber);
        
        // Build new content from results
        const newLines = results.map(r => r.output);
        const newText = newLines.join('\n');
        
        if (newText !== text) {
            const scrollTop = editor.scrollDOM.scrollTop;
            const scrollLeft = editor.scrollDOM.scrollLeft;
            
            editor.dispatch({
                changes: { from: 0, to: editor.state.doc.length, insert: newText },
            });
            
            // Restore scroll
            requestAnimationFrame(() => {
                editor.scrollDOM.scrollTop = scrollTop;
                editor.scrollDOM.scrollLeft = scrollLeft;
            });
            
            previousText = newText;
            previousLineCount = newText.split('\n').length;
        }
    } catch (err) {
        console.error('Evaluation error:', err);
    }
}

// Custom keymap for Enter key
const customKeymap = keymap.of([
    {
        key: 'Enter',
        run: handleEnterKey,
    },
]);

// Initialize editor
function initEditor() {
    const container = document.getElementById('editor-container');
    
    const state = EditorState.create({
        doc: '',
        extensions: [
            lineNumbers(),
            highlightActiveLineGutter(),
            highlightActiveLine(),
            history(),
            customKeymap,
            keymap.of([...defaultKeymap, ...historyKeymap]),
            getCurrentTheme(),
            syntaxHighlighter,
            EditorView.updateListener.of((update) => {
                if (update.docChanged) {
                    onTextChanged();
                } else if (update.selectionSet) {
                    // Selection changed without doc change (cursor movement, mouse click)
                    onSelectionChanged();
                }
            }),
            EditorView.lineWrapping,
        ],
    });

    editor = new EditorView({
        state,
        parent: container,
    });

    // Load version and set up version link
    GetVersion().then(version => {
        const versionLink = document.getElementById('version-link');
        versionLink.textContent = `Version ${version}`;
        
        // Set up click handler to open GitHub repo
        versionLink.addEventListener('click', (e) => {
            e.preventDefault();
            GetGitHubRepoURL().then(url => {
                OpenURL(url);
            });
        });
    });

    // Check for updates (on startup and every hour)
    function checkForUpdates() {
        CheckForUpdates().then(release => {
            if (release) {
                const updateNotification = document.getElementById('update-notification');
                const updateLink = document.getElementById('update-link');
                
                updateNotification.classList.remove('hidden');
                updateLink.textContent = `New version ${release.tag_name} available!`;
                
                updateLink.addEventListener('click', (e) => {
                    e.preventDefault();
                    OpenURL(release.html_url);
                });
            }
        }).catch(err => {
            // Silently ignore update check errors (e.g., no network)
            console.log('Update check failed:', err);
        });
    }
    checkForUpdates();
    setInterval(checkForUpdates, 60 * 60 * 1000); // Re-check every hour

    // Set up keyboard shortcuts
    document.addEventListener('keydown', handleKeyboard);
    
    // Listen for system theme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
        // Recreate editor with new theme
        const content = editor.state.doc.toString();
        const cursorPos = editor.state.selection.main.head;
        
        editor.destroy();
        
        const newState = EditorState.create({
            doc: content,
            extensions: [
                lineNumbers(),
                highlightActiveLineGutter(),
                highlightActiveLine(),
                history(),
                customKeymap,
                keymap.of([...defaultKeymap, ...historyKeymap]),
                e.matches ? darkTheme : lightTheme,
                syntaxHighlighter,
                EditorView.updateListener.of((update) => {
                    if (update.docChanged) {
                        onTextChanged();
                    } else if (update.selectionSet) {
                        onSelectionChanged();
                    }
                }),
                EditorView.lineWrapping,
            ],
        });
        
        editor = new EditorView({
            state: newState,
            parent: container,
        });
        
        // Restore cursor position
        editor.dispatch({
            selection: { anchor: Math.min(cursorPos, content.length) },
        });
    });
}

// Handle selection changes (cursor movement without text change)
function onSelectionChanged() {
    if (isUpdatingEditor) {
        return;
    }
    
    const cursorPos = editor.state.selection.main.head;
    const currentLine = editor.state.doc.lineAt(cursorPos);
    const currentLineNum = currentLine.number;
    
    // Check if we moved to a different line
    if (currentLineNum !== lastActiveLine) {
        // Evaluate the previous line if it has pending changes
        if (pendingEvaluation && lastActiveLine > 0) {
            evaluateLineAndDependents(lastActiveLine);
        }
        lastActiveLine = currentLineNum;
        pendingEvaluation = false;
    }
}

// Handle text changes with debounce
function onTextChanged() {
    // Skip if we're programmatically updating the editor
    if (isUpdatingEditor) {
        return;
    }
    
    // Capture the "before" state immediately, before any async operations
    // This is critical for reference adjustment to work correctly
    const snapshotPreviousText = previousText;
    
    // Get current cursor position and line
    const cursorPos = editor.state.selection.main.head;
    const currentLine = editor.state.doc.lineAt(cursorPos);
    const currentLineNum = currentLine.number;
    
    // Check if we changed lines
    if (currentLineNum !== lastActiveLine) {
        // Line changed - evaluate the previous line if it has pending changes
        if (pendingEvaluation && lastActiveLine > 0) {
            evaluateLineAndDependents(lastActiveLine);
        }
        lastActiveLine = currentLineNum;
        pendingEvaluation = false;
    }
    
    // Mark that we have pending changes on current line
    pendingEvaluation = true;
    
    // Strip result from current line if it has one (while user is editing)
    stripCurrentLineResult();
    
    if (debounceTimer) {
        clearTimeout(debounceTimer);
    }
    debounceTimer = setTimeout(() => {
        // Capture current text AFTER debounce to get the latest state (after any stripping)
        const currentText = editor.state.doc.toString();
        const currentLineCount = currentText.split('\n').length;
        // Pass the snapshot of previousText captured at the moment of change
        checkAndAdjustReferences(currentText, currentLineCount, snapshotPreviousText);
    }, 150);
    
    // Schedule autosave
    scheduleAutosave();
}

// Strip the result from the current line and all dependent lines while editing
async function stripCurrentLineResult() {
    const cursorPos = editor.state.selection.main.head;
    const line = editor.state.doc.lineAt(cursorPos);
    const lineText = line.text;
    const lineNumber = line.number;
    
    // Check if line has a result to strip
    const hasResult = await HasLineResult(lineText);
    if (!hasResult) {
        return;
    }
    
    // Strip the result from current line
    const strippedLine = await StripLineResult(lineText);
    if (strippedLine === lineText) {
        return;
    }
    
    // Calculate new cursor position
    const cursorColumn = cursorPos - line.from;
    const eqPos = findResultEqualsPos(lineText);
    
    // If cursor was in the result area, move it to after '='
    let newCursorColumn = cursorColumn;
    if (eqPos >= 0 && cursorColumn > eqPos + 1) {
        // Cursor was after the result, put it right after '='
        newCursorColumn = eqPos + 1;
    }
    
    // Find all dependent lines and strip their results too
    const text = editor.state.doc.toString();
    const dependentLines = await FindDependentLines(text, lineNumber);
    
    // Build changes array - current line first, then dependents
    const changes = [];
    changes.push({ from: line.from, to: line.to, insert: strippedLine });
    
    // Strip results from dependent lines (process in reverse order to maintain positions)
    const doc = editor.state.doc;
    for (let i = dependentLines.length - 1; i >= 0; i--) {
        const depLineNum = dependentLines[i];
        if (depLineNum <= doc.lines) {
            const depLine = doc.line(depLineNum);
            const depHasResult = await HasLineResult(depLine.text);
            if (depHasResult) {
                const strippedDepLine = await StripLineResult(depLine.text);
                if (strippedDepLine !== depLine.text) {
                    changes.push({ from: depLine.from, to: depLine.to, insert: strippedDepLine });
                }
            }
        }
    }
    
    isUpdatingEditor = true;
    try {
        editor.dispatch({
            changes: changes,
            selection: { anchor: line.from + Math.min(newCursorColumn, strippedLine.length) },
        });
    } finally {
        isUpdatingEditor = false;
    }
}

// Evaluate a line and all lines that depend on it
async function evaluateLineAndDependents(lineNumber) {
    const text = editor.state.doc.toString();
    try {
        // Add progress indicator to the line being evaluated
        const lines = text.split('\n');
        const lineIdx = lineNumber - 1; // Convert to 0-based
        if (lineIdx >= 0 && lineIdx < lines.length) {
            const line = lines[lineIdx];
            const trimmed = line.trim();
            // Add progress indicator if line ends with = (and it's not a comparison operator)
            if (trimmed.endsWith('=') && !trimmed.endsWith('==') && !trimmed.endsWith('!=') && 
                !trimmed.endsWith('<=') && !trimmed.endsWith('>=')) {
                lines[lineIdx] = line + ' ⏳';
                const progressText = lines.join('\n');
                
                // Show progress indicator immediately
                const scrollTop = editor.scrollDOM.scrollTop;
                const scrollLeft = editor.scrollDOM.scrollLeft;
                
                isUpdatingEditor = true;
                editor.dispatch({
                    changes: { from: 0, to: editor.state.doc.length, insert: progressText },
                });
                isUpdatingEditor = false;
                
                // Restore scroll
                requestAnimationFrame(() => {
                    editor.scrollDOM.scrollTop = scrollTop;
                    editor.scrollDOM.scrollLeft = scrollLeft;
                });
                
                // Small delay to ensure progress indicator is visible
                await new Promise(resolve => setTimeout(resolve, 10));
            }
        }
        
        const results = await EvaluateLines(text, lineNumber);
        
        // Build new content from results
        const newLines = results.map(r => r.output);
        const newText = newLines.join('\n');
        
        if (newText !== text) {
            const scrollTop = editor.scrollDOM.scrollTop;
            const scrollLeft = editor.scrollDOM.scrollLeft;
            const cursorPos = editor.state.selection.main.head;
            const cursorLine = editor.state.doc.lineAt(cursorPos);
            const lineNumber = cursorLine.number;
            const columnOffset = cursorPos - cursorLine.from;
            
            isUpdatingEditor = true;
            try {
                editor.dispatch({
                    changes: { from: 0, to: editor.state.doc.length, insert: newText },
                });
                
                // Restore cursor position
                const newDoc = editor.state.doc;
                if (lineNumber <= newDoc.lines) {
                    const newLine = newDoc.line(lineNumber);
                    const newPos = newLine.from + Math.min(columnOffset, newLine.length);
                    editor.dispatch({
                        selection: { anchor: newPos },
                    });
                }
                
                // Restore scroll
                requestAnimationFrame(() => {
                    editor.scrollDOM.scrollTop = scrollTop;
                    editor.scrollDOM.scrollLeft = scrollLeft;
                });
                
                previousText = newText;
                previousLineCount = newText.split('\n').length;
            } finally {
                isUpdatingEditor = false;
            }
        }
    } catch (err) {
        console.error('Evaluation error:', err);
    }
}

// Check if line count changed and adjust references
// currentText/currentLineCount are the current state
// snapshotPreviousText is the state BEFORE the change (captured at moment of change)
async function checkAndAdjustReferences(currentText, currentLineCount, snapshotPreviousText) {
    // Prevent re-entry when we dispatch programmatically
    if (isUpdatingEditor) {
        return;
    }
    
    // Skip if the editor content has changed since the snapshot was taken
    // This prevents processing stale snapshots after evaluation has already updated the content
    if (editor.state.doc.toString() !== currentText) {
        return;
    }
    
    // Use the snapshot of previousText if provided, otherwise use global previousText
    const oldText = snapshotPreviousText !== undefined ? snapshotPreviousText : previousText;
    
    // Skip if oldText is empty (first change)
    if (oldText === '') {
        previousText = currentText;
        previousLineCount = currentLineCount;
        updateUnsavedState();
        return;
    }
    
    // Count total lines for reference adjustment
    const prevLineCount = oldText.split('\n').length;
    const currLineCount = currentText.split('\n').length;
    
    // If line count changed, adjust references
    if (prevLineCount !== currLineCount) {
        try {
            const adjusted = await AdjustReferences(oldText, currentText);
            if (adjusted !== currentText) {
                // After adjusting references, strip results from referencing lines and re-evaluate
                // This prevents showing ERR temporarily when references are adjusted
                const reEvaluated = await StripAndEvalReferencingLines(adjusted);
                
                // Set flag to prevent re-entry
                isUpdatingEditor = true;
                
                // Save cursor and scroll position
                const scrollTop = editor.scrollDOM.scrollTop;
                const scrollLeft = editor.scrollDOM.scrollLeft;
                const cursorPos = editor.state.selection.main.head;
                
                // Update with re-evaluated text
                editor.dispatch({
                    changes: { from: 0, to: editor.state.doc.length, insert: reEvaluated },
                    selection: { anchor: Math.min(cursorPos, reEvaluated.length) },
                });
                
                // Restore scroll
                requestAnimationFrame(() => {
                    editor.scrollDOM.scrollTop = scrollTop;
                    editor.scrollDOM.scrollLeft = scrollLeft;
                });
                
                // Update previous text to re-evaluated version
                previousText = reEvaluated;
                previousLineCount = reEvaluated.split('\n').length;
                
                // Clear flag after adjustment is done
                isUpdatingEditor = false;
                return;
            }
        } catch (err) {
            console.error('Adjust references error:', err);
            isUpdatingEditor = false;
        }
    }
    
    // Update previous text tracking
    previousText = currentText;
    previousLineCount = currentLineCount;
    
    // Update unsaved state
    updateUnsavedState();
    
    // Note: We no longer evaluate on every change - lazy evaluation happens on line change or Enter
}

// Update unsaved state and notify backend
function updateUnsavedState() {
    const currentContent = editor.state.doc.toString();
    const hasUnsaved = currentContent !== savedContent;
    SetUnsavedState(hasUnsaved, currentFile);
}

// Schedule autosave after delay
function scheduleAutosave() {
    if (autosaveTimer) {
        clearTimeout(autosaveTimer);
    }
    autosaveTimer = setTimeout(() => {
        performAutosave();
    }, AUTOSAVE_DELAY);
}

// Perform autosave if we have a current file
async function performAutosave() {
    if (currentFile) {
        try {
            await AutoSave(currentFile, editor.state.doc.toString());
        } catch (err) {
            console.error('Autosave error:', err);
        }
    }
}

// Evaluate content and update display (sets flag to prevent re-entry)
async function evaluateContent() {
    isUpdatingEditor = true;
    try {
        await evaluateContentInternal();
    } finally {
        isUpdatingEditor = false;
    }
}

// Recalculate all expressions - strips all results and re-evaluates everything in parallel
async function recalculateAll() {
    isUpdatingEditor = true;
    try {
        const text = editor.state.doc.toString();
        const scrollTop = editor.scrollDOM.scrollTop;
        const scrollLeft = editor.scrollDOM.scrollLeft;
        const cursorPos = editor.state.selection.main.head;
        const cursorLine = editor.state.doc.lineAt(cursorPos);
        const lineNumber = cursorLine.number;
        const columnOffset = cursorPos - cursorLine.from;
        
        // Strip all results and output lines
        const strippedText = await StripAllResults(text);
        
        // Add progress indicators (= ⏳) to all expression lines
        const strippedLines = strippedText.split('\n');
        const withProgress = strippedLines.map(line => {
            const trimmed = line.trim();
            // Add progress indicator to lines with = but no result yet
            if (trimmed.endsWith('=') && !trimmed.endsWith('==') && !trimmed.endsWith('!=') && 
                !trimmed.endsWith('<=') && !trimmed.endsWith('>=')) {
                return line + ' ⏳';
            }
            return line;
        });
        const progressText = withProgress.join('\n');
        
        // Show progress indicators immediately
        editor.dispatch({
            changes: { from: 0, to: editor.state.doc.length, insert: progressText },
        });
        
        // Force a render to show progress indicators
        await new Promise(resolve => requestAnimationFrame(resolve));
        
        // Now evaluate all expressions in parallel
        const results = await EvaluateParallel(strippedText);
        const newLines = results.map(r => r.output);
        const newText = newLines.join('\n');
        
        editor.dispatch({
            changes: { from: 0, to: editor.state.doc.length, insert: newText },
        });
        
        // Restore cursor position
        const newDoc = editor.state.doc;
        if (lineNumber <= newDoc.lines) {
            const newLine = newDoc.line(lineNumber);
            const newPos = newLine.from + Math.min(columnOffset, newLine.length);
            editor.dispatch({
                selection: { anchor: newPos },
            });
        }
        
        // Restore scroll
        requestAnimationFrame(() => {
            editor.scrollDOM.scrollTop = scrollTop;
            editor.scrollDOM.scrollLeft = scrollLeft;
        });
        
        previousText = newText;
        previousLineCount = newText.split('\n').length;
    } catch (err) {
        console.error('Recalculate error:', err);
    } finally {
        isUpdatingEditor = false;
    }
}

// Internal evaluate function (does not manage flag)
async function evaluateContentInternal() {
    const text = editor.state.doc.toString();
    try {
        // Get current cursor line number (1-based) to skip formatting for active line
        const cursorPos = editor.state.selection.main.head;
        const activeLineNum = editor.state.doc.lineAt(cursorPos).number;
        
        const results = await Evaluate(text, activeLineNum);
        
        // Build new content from results
        const newLines = results.map(r => r.output);
        const newText = newLines.join('\n');
        
        // Only update if different (to avoid cursor jump)
        if (newText !== text) {
            // Save scroll position and cursor line/column
            const scrollTop = editor.scrollDOM.scrollTop;
            const scrollLeft = editor.scrollDOM.scrollLeft;
            const cursorPos = editor.state.selection.main.head;
            const cursorLine = editor.state.doc.lineAt(cursorPos);
            const lineNumber = cursorLine.number;
            const columnOffset = cursorPos - cursorLine.from;
            
            // Detect if multi-line output was added (new lines starting with ">")
            const oldOutputLines = text.split('\n').filter(l => l.startsWith('>')).length;
            const newOutputLines = newText.split('\n').filter(l => l.startsWith('>')).length;
            const hasNewMultiLineOutput = newOutputLines > oldOutputLines && newOutputLines > 1;
            
            editor.dispatch({
                changes: { from: 0, to: editor.state.doc.length, insert: newText },
            });
            
            // Restore cursor position based on line number
            const newDoc = editor.state.doc;
            if (lineNumber <= newDoc.lines) {
                const newLine = newDoc.line(lineNumber);
                const newPos = newLine.from + Math.min(columnOffset, newLine.length);
                editor.dispatch({
                    selection: { anchor: newPos },
                });
            } else {
                // If line doesn't exist, go to end
                editor.dispatch({
                    selection: { anchor: newText.length },
                });
            }
            
            // Scroll handling after update
            requestAnimationFrame(() => {
                if (hasNewMultiLineOutput) {
                    // Scroll to show the last output line
                    const lastLineNum = newDoc.lines;
                    const lastLine = newDoc.line(lastLineNum);
                    // Use scrollIntoView effect to scroll the last line into view
                    editor.dispatch({
                        effects: EditorView.scrollIntoView(lastLine.from, { y: 'end' })
                    });
                } else {
                    // Restore previous scroll position
                    editor.scrollDOM.scrollTop = scrollTop;
                    editor.scrollDOM.scrollLeft = scrollLeft;
                }
            });
            
            // Update previous text to the evaluated result
            previousText = newText;
            previousLineCount = newText.split('\n').length;
        }
    } catch (err) {
        console.error('Evaluation error:', err);
    }
}

// Keyboard shortcuts
function handleKeyboard(e) {
    // Ctrl+O - Open
    if (e.ctrlKey && e.key === 'o') {
        e.preventDefault();
        openFile();
    }
    // Ctrl+S - Save
    if (e.ctrlKey && e.key === 's') {
        e.preventDefault();
        saveFile();
    }
    // Ctrl+Shift+S - Save As
    if (e.ctrlKey && e.shiftKey && e.key === 'S') {
        e.preventDefault();
        saveFileAs();
    }
    // Ctrl+N - New
    if (e.ctrlKey && e.key === 'n') {
        e.preventDefault();
        newFile();
    }
    // Ctrl+R - Recalculate all expressions
    if (e.ctrlKey && e.key === 'r') {
        e.preventDefault();
        recalculateAll();
    }
    // Ctrl+C - Smart Copy (replace refs with values)
    if (e.ctrlKey && e.key === 'c') {
        e.preventDefault();
        smartCopy();
    }
    // Ctrl+V - Paste
    if (e.ctrlKey && e.key === 'v') {
        e.preventDefault();
        smartPaste();
    }
}

// File operations
async function openFile() {
    try {
        const path = await OpenFileDialog();
        if (path) {
            await openFilePath(path);
        }
    } catch (err) {
        console.error('Open error:', err);
    }
}

async function openFilePath(path) {
    try {
        const content = await ReadFile(path);
        editor.dispatch({
            changes: { from: 0, to: editor.state.doc.length, insert: content },
        });
        currentFile = path;
        savedContent = content; // Mark as saved
        updateFileName();
        evaluateContent();
        await AddRecentFile(path);
        SetUnsavedState(false, currentFile);
    } catch (err) {
        console.error('Open error:', err);
    }
}

async function saveFile() {
    if (currentFile) {
        try {
            const content = editor.state.doc.toString();
            await WriteFile(currentFile, content);
            savedContent = content; // Mark as saved
            await AddRecentFile(currentFile);
            SetUnsavedState(false, currentFile);
        } catch (err) {
            console.error('Save error:', err);
        }
    } else {
        saveFileAs();
    }
}

async function saveFileAs() {
    try {
        const path = await SaveFileDialog();
        if (path) {
            const content = editor.state.doc.toString();
            await WriteFile(path, content);
            currentFile = path;
            savedContent = content; // Mark as saved
            updateFileName();
            await AddRecentFile(path);
            SetUnsavedState(false, currentFile);
        }
    } catch (err) {
        console.error('Save As error:', err);
    }
}

// Welcome message for new documents
const WELCOME_MESSAGE = `# Welcome to SmartCalc!
# Check out the Snippets menu to explore features.
# Type an expression and press Enter to calculate.

`;

function newFile() {
    editor.dispatch({
        changes: { from: 0, to: editor.state.doc.length, insert: WELCOME_MESSAGE },
    });
    // Move cursor to end
    editor.dispatch({
        selection: { anchor: WELCOME_MESSAGE.length },
    });
    currentFile = '';
    savedContent = WELCOME_MESSAGE; // New file starts as "saved"
    updateFileName();
    SetUnsavedState(false, '');
}

function updateFileName() {
    const name = currentFile ? currentFile : 'Untitled';
    document.getElementById('file-name').textContent = name;
}

// Adjust line references in snippet based on insertion line
function adjustSnippetReferences(snippet, insertionLine) {
    // Replace \N references with adjusted line numbers
    // \1 -> \(insertionLine), \2 -> \(insertionLine+1), etc.
    return snippet.replace(/\\(\d+)/g, (match, num) => {
        const originalRef = parseInt(num, 10);
        const adjustedRef = originalRef + insertionLine - 1;
        return '\\' + adjustedRef;
    });
}

// Insert snippet at end of document
async function insertSnippet(snippet) {
    // Always insert at the end of the document
    const docLength = editor.state.doc.length;
    const lastLine = editor.state.doc.lines;
    
    // Move cursor to end of document first
    editor.dispatch({
        selection: { anchor: docLength },
        scrollIntoView: true,
    });
    
    // Adjust line references in the snippet based on insertion line
    const adjustedSnippet = adjustSnippetReferences(snippet, lastLine);
    
    // Insert the snippet at end of document
    editor.dispatch({
        changes: { from: docLength, insert: adjustedSnippet },
        scrollIntoView: true,
    });
    
    // Evaluate the content
    await evaluateContent();
    
    // Use requestAnimationFrame to ensure cursor positioning happens after
    // evaluateContent's internal cursor restoration completes
    requestAnimationFrame(() => {
        const newDocLength = editor.state.doc.length;
        editor.dispatch({
            selection: { anchor: newDocLength },
        });
        // Use scrollIntoView effect to ensure the last line is visible at the bottom
        editor.dispatch({
            effects: EditorView.scrollIntoView(newDocLength, { y: 'end' })
        });
        editor.focus();
    });
}

// Modal dialog functions
function showModal(title, content, iconSrc = null) {
    const overlay = document.getElementById('modal-overlay');
    const titleEl = document.getElementById('modal-title');
    const contentEl = document.getElementById('modal-content');
    const iconEl = document.getElementById('modal-icon');
    const okBtn = document.getElementById('modal-ok-btn');

    // Set title
    titleEl.textContent = title;

    // Set icon
    if (iconSrc) {
        iconEl.src = iconSrc;
        iconEl.classList.remove('hidden');
    } else {
        iconEl.classList.add('hidden');
    }

    // Clear previous content
    contentEl.innerHTML = '';

    // Destroy previous modal editor if exists
    if (modalEditor) {
        modalEditor.destroy();
        modalEditor = null;
    }

    // Create read-only CodeMirror editor for content
    const modalState = EditorState.create({
        doc: content,
        extensions: [
            EditorState.readOnly.of(true),
            EditorView.editable.of(false),
            getCurrentTheme(),
            syntaxHighlighter,
            EditorView.lineWrapping,
        ],
    });

    modalEditor = new EditorView({
        state: modalState,
        parent: contentEl,
    });

    // Show modal
    overlay.classList.remove('hidden');

    // Focus OK button
    okBtn.focus();

    // Close handlers
    const closeModal = () => {
        overlay.classList.add('hidden');
        if (modalEditor) {
            modalEditor.destroy();
            modalEditor = null;
        }
        editor.focus();
    };

    okBtn.onclick = closeModal;
    overlay.onclick = (e) => {
        if (e.target === overlay) {
            closeModal();
        }
    };

    // ESC key to close
    const escHandler = (e) => {
        if (e.key === 'Escape') {
            closeModal();
            document.removeEventListener('keydown', escHandler);
        }
    };
    document.addEventListener('keydown', escHandler);
}

// Show manual dialog
function showManual() {
    const manual = `# SmartCalc Manual

SmartCalc is a powerful multi-purpose calculator that goes
beyond basic arithmetic. It understands natural language
expressions for dates, units, percentages, finances, and more.

Simply type your expression followed by = and SmartCalc
will calculate the result. Use line references (\\1, \\2)
to build on previous calculations. Add comments with #.

## Basic Math
10 + 20 * 3 = 70
$1,500.00 + $250.50 = $1,750.50
$1,000 x 12 - 15% + $500 = $10,700.00
sin(45) + cos(30) = 1.57
sqrt(144) = 12
abs(-50) = 50
25 > 2.5 = true
100 >= 100 = true

## Line References
100 = 100
\\1 * 2 = 200

## Base Conversion
255 in hex = 0xFF
0xFF in dec = 255
25 in bin = 0b11001

## Constants
pi = 3.14159265359
e = 2.71828182846
speed of light = 299,792,458 m/s
gravity = 9.80665 m/s²

## Date & Time
now = (current time)
today = (current date)
now in Seattle = (Seattle time)
today + 30 days = (future date)
6:00 am Seattle in Kiev = (converted time)

## Network/IP
10.100.0.0/24 = 254 hosts
mask for /24 = 255.255.255.0
wildcard for /24 = 0.0.0.255
broadcast for 10.100.0.0/24 = 10.100.0.255
is 10.100.0.50 in 10.100.0.0/24 = yes
10.100.0.0/16 / 4 subnets = (subnet list)

## Unit Conversions
5 miles in km = 8.05 km
100 f to c = 37.78 °C
10 kg in lbs = 22.05 lbs
5 gallons in liters = 18.93 L
60 mph to kph = 96.56 kph
1 acre to sqft = 43,560 sqft

## Percentage
$100 - 20% = $80.00
$100 + 15% = $115.00
what is 15% of 200 = 30
50 is what % of 200 = 25%
increase 100 by 20% = 120
percent change from 50 to 75 = +50%
tip 20% on $85.50 = Tip: $17.10
$150 split 4 ways = $37.50/person

## Financial
loan $250000 at 6.5% for 30 years = $1,580.17/month
mortgage $350000 at 7% for 30 years = $2,328.56/month
$10000 at 5% for 10 years compounded monthly = $16,470.09
simple interest $5000 at 3% for 2 years = $300.00
invest $1000 at 7% for 20 years = $3,869.68

## Statistics
avg(10, 20, 30, 40) = 25
median(1, 2, 3, 4, 5) = 3
sum(10, 20, 30) = 60
count(1, 2, 3, 4, 5) = 5
min(10, 5, 20, 3) = 3
max(10, 5, 20, 3) = 20
stddev(2, 4, 4, 4, 5, 5, 7, 9) = 2
range(1, 5, 10, 3) = 9

## Programmer
0xFF AND 0x0F = 15 (0xF)
0xF0 OR 0x0F = 255 (0xFF)
0xFF XOR 0x0F = 240 (0xF0)
1 << 8 = 256 (0x100)
256 >> 4 = 16 (0x10)
ascii A = 65
char 65 = A
md5 hello = 5d41402abc4b2a76...
base64 encode hello = aGVsbG8=
base64 decode aGVsbG8= = hello
uuid = (random UUID)
random 1 to 100 = (random number)

# Check the Snippets menu for more examples!`;
    showModal("SmartCalc Manual", manual);
}

// Show about dialog
function showAbout() {
    GetVersion().then(version => {
        const about = `# About SmartCalc

Version: ${version}

A powerful multi-purpose calculator with support for:

• Mathematical expressions & functions
• Unit conversions (length, weight, temperature, etc.)
• Percentage & financial calculations
• Statistics (avg, median, stddev, etc.)
• Date/Time calculations & time zones
• Network/IP subnet calculations
• Programmer utilities (bitwise, ASCII, hashing)
• Physical & mathematical constants

© Vladimir Poluyaktov 2025
https://github.com/vpoluyaktov/smartcalc

# Keyboard Shortcuts

Ctrl+N    New file
Ctrl+O    Open file
Ctrl+S    Save file
Ctrl+R    Recalculate all expressions
Ctrl+C    Copy (with resolved references)
Ctrl+V    Paste
Enter     Auto-append = and calculate`;
        showModal("About SmartCalc", about);
    });
}

// Smart copy - replace line references with actual values
async function smartCopy() {
    const selection = editor.state.selection.main;
    let textToCopy;
    
    if (selection.empty) {
        // No selection - copy entire document
        textToCopy = editor.state.doc.toString();
    } else {
        // Copy selected text
        textToCopy = editor.state.sliceDoc(selection.from, selection.to);
    }
    
    // Replace line references with actual values
    const resolvedText = await CopyWithResolvedRefs(textToCopy);
    
    // Copy to clipboard using Wails runtime
    ClipboardSetText(resolvedText);
}

// Paste from clipboard using Wails runtime
async function smartPaste() {
    try {
        const text = await ClipboardGetText();
        if (text) {
            const selection = editor.state.selection.main;
            editor.dispatch({
                changes: { from: selection.from, to: selection.to, insert: text },
                selection: { anchor: selection.from + text.length },
            });
        }
    } catch (err) {
        console.error('Paste error:', err);
    }
}

// Set up menu event listeners
function setupMenuEvents() {
    EventsOn('menu:new', newFile);
    EventsOn('menu:open', openFile);
    EventsOn('menu:save', saveFile);
    EventsOn('menu:saveAs', saveFileAs);
    EventsOn('menu:openRecent', openFilePath);
    EventsOn('menu:cut', () => document.execCommand('cut'));
    EventsOn('menu:copy', smartCopy);
    EventsOn('menu:paste', smartPaste);
    EventsOn('menu:snippet', insertSnippet);
    EventsOn('menu:manual', showManual);
    EventsOn('menu:about', showAbout);
    EventsOn('app:saveAndQuit', saveAndQuit);
}

// Save file and quit - called when user clicks Save on unsaved unnamed file close
async function saveAndQuit() {
    try {
        const path = await SaveFileDialog();
        if (path) {
            const content = editor.state.doc.toString();
            await WriteFile(path, content);
            currentFile = path;
            savedContent = content;
            await AddRecentFile(path);
            // Clear unsaved state BEFORE quitting to prevent loop
            await SetUnsavedState(false, path);
            // Now quit the app
            Quit();
        }
        // If user cancelled the save dialog, don't quit (they already cancelled the close)
    } catch (err) {
        console.error('Save and quit error:', err);
    }
}

// Load last file on startup, or show welcome message if no file
async function loadLastFile() {
    try {
        const lastFile = await GetLastFile();
        if (lastFile) {
            await openFilePath(lastFile);
        } else {
            // No last file - show welcome message
            newFile();
        }
    } catch (err) {
        console.error('Load last file error:', err);
        // On error, show welcome message
        newFile();
    }
}

// Context menu functionality
function setupContextMenu() {
    const contextMenu = document.getElementById('context-menu');
    const editorContainer = document.getElementById('editor-container');

    // Show context menu on right-click
    editorContainer.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        
        // Position the menu at click location
        const x = e.clientX;
        const y = e.clientY;
        
        // Ensure menu doesn't go off screen
        const menuWidth = 180;
        const menuHeight = 160;
        const windowWidth = window.innerWidth;
        const windowHeight = window.innerHeight;
        
        const posX = x + menuWidth > windowWidth ? windowWidth - menuWidth - 10 : x;
        const posY = y + menuHeight > windowHeight ? windowHeight - menuHeight - 10 : y;
        
        contextMenu.style.left = posX + 'px';
        contextMenu.style.top = posY + 'px';
        contextMenu.classList.remove('hidden');
    });

    // Hide context menu on click elsewhere
    document.addEventListener('click', (e) => {
        if (!contextMenu.contains(e.target)) {
            contextMenu.classList.add('hidden');
        }
    });

    // Hide context menu on scroll
    document.addEventListener('scroll', () => {
        contextMenu.classList.add('hidden');
    }, true);

    // Hide context menu on Escape
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            contextMenu.classList.add('hidden');
        }
    });

    // Handle context menu item clicks
    contextMenu.querySelectorAll('.context-menu-item').forEach(item => {
        item.addEventListener('click', async () => {
            const action = item.dataset.action;
            contextMenu.classList.add('hidden');
            
            // Focus editor before executing action
            editor.focus();
            
            switch (action) {
                case 'cut':
                    await performCut();
                    break;
                case 'copy':
                    await smartCopy();
                    break;
                case 'paste':
                    await smartPaste();
                    break;
                case 'selectall':
                    selectAll();
                    break;
            }
        });
    });
}

// Perform cut operation
async function performCut() {
    const selection = editor.state.selection.main;
    if (selection.empty) return;
    
    const selectedText = editor.state.sliceDoc(selection.from, selection.to);
    
    try {
        await ClipboardSetText(selectedText);
        
        // Delete the selected text
        editor.dispatch({
            changes: { from: selection.from, to: selection.to, insert: '' },
            selection: { anchor: selection.from }
        });
    } catch (err) {
        console.error('Cut error:', err);
    }
}

// Select all text in editor
function selectAll() {
    const docLength = editor.state.doc.length;
    editor.dispatch({
        selection: { anchor: 0, head: docLength }
    });
}

// Initialize on load
document.addEventListener('DOMContentLoaded', () => {
    initEditor();
    setupMenuEvents();
    setupContextMenu();
    loadLastFile();
});

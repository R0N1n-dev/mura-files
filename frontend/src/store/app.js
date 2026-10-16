/**
 * Application store.
 *
 * A single reactive object holds every piece of UI state. Keeping it in one
 * place (rather than scattering refs across components) makes the interactions
 * that matter for a file manager - tabs, selection, clipboard, operations -
 * easy to reason about.
 */

import { reactive, computed, watch } from 'vue';
import { Events } from '@wailsio/runtime';
import {
  AppService, FileService, OperationService, SearchService, WatchService,
  listOptions, searchQuery, transferOptions, deleteOptions, describeError,
} from '../lib/fs.js';
import { clamp } from '../lib/format.js';

/* ------------------------------------------------------------------------ */
/* State                                                                     */
/* ------------------------------------------------------------------------ */

const DEFAULT_SETTINGS = {
  theme: 'system',
  accentColour: '',
  viewMode: 'list',
  gridSize: 112,
  sortBy: 'name',
  sortDesc: false,
  dirsFirst: true,
  groupByKind: false,
  showHidden: false,
  showExtensions: true,
  showThumbnails: true,
  showPreview: false,
  showSidebar: true,
  showStatusBar: true,
  confirmDelete: true,
  confirmTrash: false,
  useTrash: true,
  confirmMove: false,
  singleClickOpen: false,
  restoreTabs: true,
  openTabs: [],
  activeTab: 0,
  dateFormat: 'auto',
  sizeFormat: 'binary',
  terminalCommand: '',
  editorCommand: '',
  thumbnailSize: 128,
  watchEnabled: true,
  sidebarWidth: 244,
  previewWidth: 320,
  hiddenFolders: [],
  confirmCloseTabs: false,
  followLinks: false,
  autoExpandTree: true,
  showFolderSize: false,
};

export const state = reactive({
  ready: false,
  bootError: '',
  info: null,
  settings: { ...DEFAULT_SETTINGS },
  places: [],
  drives: [],
  bookmarks: [],

  tabs: [],
  activeTabId: '',

  ops: [],
  toasts: [],
  clipboard: { mode: '', paths: [] },
  contextMenu: { open: false, x: 0, y: 0, items: [] },
  dialogs: {
    prompt: null,
    confirm: null,
    properties: null,
    compress: null,
    appPicker: null,
    settings: false,
    shortcuts: false,
    about: false,
    trash: false,
  },
  theme: { dark: true, accent: '', source: 'default' },
  trash: [],
  trashLoading: false,
  palette: { open: false, query: '', index: 0 },
  renaming: { path: '', value: '' },
  tree: { expanded: {}, children: {}, loading: {} },
  drag: { active: false, paths: [], over: '', external: false },
  conflict: null,
  notify: { drop: [] },
});

/* ------------------------------------------------------------------------ */
/* Tab management                                                            */
/* ------------------------------------------------------------------------ */

let tabSequence = 0;

export function createTab(path = '', overrides = {}) {
  tabSequence += 1;
  const settings = state.settings;
  return reactive({
    id: `tab-${tabSequence}`,
    title: 'Home',
    path,
    history: [],
    historyIndex: -1,
    mode: 'browse',
    entries: [],
    listing: null,
    loading: false,
    error: '',
    selected: new Set(),
    anchor: '',
    cursor: '',
    filter: '',
    sortBy: settings.sortBy || 'name',
    sortDesc: !!settings.sortDesc,
    dirsFirst: settings.dirsFirst !== false,
    groupByKind: !!settings.groupByKind,
    showHidden: !!settings.showHidden,
    viewMode: settings.viewMode || 'list',
    searchPattern: '',
    searchRunning: false,
    searchId: '',
    searchScanned: 0,
    searchFound: 0,
    searchElapsed: 0,
    searchDone: false,
    searchOptions: { includeContent: false, includeHidden: false, mode: 'glob', matchCase: false },
    archivePath: '',
    archiveInfo: null,
    requestId: 0,
    ...overrides,
  });
}

/** The tab currently in focus. */
export const activeTab = computed(() =>
  state.tabs.find((tab) => tab.id === state.activeTabId) || state.tabs[0] || null);

/** Entries visible in the active tab, honouring the quick filter. */
export const visibleEntries = computed(() => {
  const tab = activeTab.value;
  if (!tab) return [];
  return tab.entries;
});

/** Paths selected in the active tab, as an array. */
export const selectedPaths = computed(() => {
  const tab = activeTab.value;
  if (!tab) return [];
  return [...tab.selected];
});

export const selectedEntries = computed(() => {
  const tab = activeTab.value;
  if (!tab) return [];
  return tab.entries.filter((entry) => tab.selected.has(entry.path));
});

/** Aggregate size of the current selection (files only). */
export const selectionSize = computed(() =>
  selectedEntries.value.reduce((total, entry) => total + (entry.isDir ? 0 : entry.size || 0), 0));

/** The entry the preview pane should display. */
export const previewEntry = computed(() => {
  const entries = selectedEntries.value;
  if (entries.length === 1) return entries[0];
  return null;
});

export function tabById(id) {
  return state.tabs.find((tab) => tab.id === id) || null;
}

/* ------------------------------------------------------------------------ */
/* Toasts                                                                    */
/* ------------------------------------------------------------------------ */

let toastSequence = 0;

export function notify(message, kind = 'info', title = '', timeout = 4200) {
  const id = `toast-${++toastSequence}`;
  state.toasts.push({ id, message, kind, title });
  if (timeout > 0) {
    setTimeout(() => dismissToast(id), timeout);
  }
  return id;
}

export function dismissToast(id) {
  const index = state.toasts.findIndex((toast) => toast.id === id);
  if (index >= 0) state.toasts.splice(index, 1);
}

export const toast = {
  info: (message, title) => notify(message, 'info', title),
  success: (message, title) => notify(message, 'success', title),
  warn: (message, title) => notify(message, 'warning', title),
  error: (message, title) => notify(message, 'error', title, 7000),
};

/* ------------------------------------------------------------------------ */
/* Bootstrap                                                                 */
/* ------------------------------------------------------------------------ */

export async function bootstrap() {
  try {
    const [info, settings, bookmarks, places, drives] = await Promise.all([
      AppService.Info(),
      AppService.Settings(),
      AppService.Bookmarks(),
      FileService.WellKnown(),
      FileService.Drives(),
    ]);

    state.info = info;
    state.settings = Object.assign({}, DEFAULT_SETTINGS, settings || {});
    state.bookmarks = bookmarks || [];
    state.places = places || [];
    state.drives = drives || [];
    state.theme = { dark: !!info?.darkMode, accent: info?.accent || '', source: 'system' };

    applyTheme();
    restoreTabs();
    wireBackendEvents();
    state.ready = true;

    const tab = activeTab.value;
    if (tab) await navigate(tab, tab.path, { replace: true });
  } catch (error) {
    state.bootError = describeError(error);
    state.ready = true;
  }
}

function restoreTabs() {
  const settings = state.settings;
  const paths = settings.restoreTabs && Array.isArray(settings.openTabs) ? settings.openTabs : [];
  const home = state.info?.os?.home || '';
  if (paths.length === 0) {
    const tab = createTab(home);
    state.tabs.push(tab);
    state.activeTabId = tab.id;
    return;
  }
  paths.forEach((path) => state.tabs.push(createTab(path)));
  const index = clamp(Number(settings.activeTab) || 0, 0, state.tabs.length - 1);
  state.activeTabId = state.tabs[index].id;
}

/* ------------------------------------------------------------------------ */
/* Backend event wiring                                                      */
/* ------------------------------------------------------------------------ */

let eventsWired = false;

function wireBackendEvents() {
  if (eventsWired) return;
  eventsWired = true;

  Events.On('op:progress', (event) => upsertOperation(event.data));
  Events.On('op:finished', (event) => {
    const progress = event.data;
    if (!progress) return;
    if (progress.status === 'error') {
      toast.error(progress.message || 'Operation failed', progress.title);
    } else if (progress.status === 'cancelled') {
      toast.warn('Cancelled', progress.title);
    }
    refreshActiveAfterOperation(progress);
  });

  Events.On('search:results', (event) => handleSearchBatch(event.data));

  Events.On('fs:changed', (event) => {
    const payload = event.data;
    if (!payload || !payload.dir) return;
    state.tabs.forEach((tab) => {
      if (tab.mode === 'browse' && tab.path === payload.dir && !tab.loading) {
        scheduleReload(tab);
      }
    });
    // Invalidate cached sidebar tree children for the changed folder.
    if (state.tree.children[payload.dir]) {
      delete state.tree.children[payload.dir];
    }
  });

  Events.On('app:notify', (event) => {
    const payload = event.data;
    if (!payload) return;
    if (payload.kind === 'drop') {
      const paths = String(payload.message || '').split('\n').filter(Boolean);
      if (paths.length) handleExternalDrop(paths, payload.title || '');
      return;
    }
    if (payload.kind === 'theme') {
      refreshTheme();
    }
  });

  Events.On('app:open-path', (event) => {
    const raw = event.data;
    const paths = String(raw || '').split('\n').map((p) => p.trim()).filter(Boolean);
    if (paths.length) openPathsInNewTab(paths);
  });
}

/* ------------------------------------------------------------------------ */
/* Theme                                                                     */
/* ------------------------------------------------------------------------ */

const ACCENTS = {
  indigo: 'accent-indigo',
  violet: 'accent-violet',
  teal: 'accent-teal',
  rose: 'accent-rose',
  amber: 'accent-amber',
  emerald: 'accent-emerald',
  sky: 'accent-sky',
  slate: 'accent-slate',
};

export function resolveDark() {
  const mode = state.settings.theme || 'system';
  if (mode === 'dark') return true;
  if (mode === 'light') return false;
  return !!state.theme.dark;
}

export function applyTheme() {
  const root = document.documentElement;
  const dark = resolveDark();
  root.classList.toggle('theme-dark', dark);
  root.classList.toggle('theme-light', !dark);

  const accent = state.settings.accentColour || '';
  Object.values(ACCENTS).forEach((cls) => root.classList.remove(cls));
  if (ACCENTS[accent]) root.classList.add(ACCENTS[accent]);
}

export async function refreshTheme() {
  try {
    const theme = await AppService.Theme();
    if (theme) state.theme = theme;
  } catch {
    /* the desktop may not expose a colour scheme; keep the current one */
  }
  applyTheme();
}

/* ------------------------------------------------------------------------ */
/* Settings persistence                                                      */
/* ------------------------------------------------------------------------ */

const SETTING_KEYS = Object.keys(DEFAULT_SETTINGS);
let settingsTimer = null;

/** Persist a subset of the settings, debounced so typing stays smooth. */
export function updateSettings(patch, immediate = false) {
  Object.assign(state.settings, patch);
  if ('theme' in patch || 'accentColour' in patch) applyTheme();

  const payload = {};
  Object.keys(patch).forEach((key) => {
    if (SETTING_KEYS.includes(key)) payload[key] = state.settings[key];
  });
  if (!Object.keys(payload).length) return;

  if (settingsTimer) clearTimeout(settingsTimer);
  const flush = () => {
    AppService.PatchSettings(payload).catch(() => {});
    settingsTimer = null;
  };
  if (immediate) flush();
  else settingsTimer = setTimeout(flush, 450);
}

/** Remember the open tabs so the session can be restored. */
export function persistSession() {
  const paths = state.tabs.map((tab) => tab.path).filter(Boolean);
  const index = state.tabs.findIndex((tab) => tab.id === state.activeTabId);
  updateSettings({ openTabs: paths, activeTab: Math.max(0, index) });
}

/* ------------------------------------------------------------------------ */
/* Navigation and directory loading                                          */
/* ------------------------------------------------------------------------ */

const reloadTimers = new Map();

function scheduleReload(tab) {
  if (reloadTimers.has(tab.id)) clearTimeout(reloadTimers.get(tab.id));
  reloadTimers.set(tab.id, setTimeout(() => {
    reloadTimers.delete(tab.id);
    if (!tab.loading) loadDirectory(tab);
  }, 180));
}

/**
 * Navigate a tab to a path. History is updated unless `replace` is set.
 *
 * @param {object} tab
 * @param {string} path
 * @param {{ replace?: boolean, skipHistory?: boolean, select?: string }} [opts]
 */
export async function navigate(tab, path, opts = {}) {
  if (!tab) return;
  const target = path || state.info?.os?.home || '';
  tab.mode = 'browse';
  tab.archivePath = '';
  tab.archiveInfo = null;
  tab.error = '';

  if (!opts.skipHistory) {
    if (opts.replace || tab.historyIndex < 0) {
      tab.history = [target];
      tab.historyIndex = 0;
    } else if (tab.history[tab.historyIndex] !== target) {
      tab.history = tab.history.slice(0, tab.historyIndex + 1);
      tab.history.push(target);
      tab.historyIndex = tab.history.length - 1;
    }
  }

  tab.path = target;
  await loadDirectory(tab, opts.select);
  if (tab.id === state.activeTabId) {
    AppService.SetTitle(`${tab.listing?.name || 'Mura Files'} — Mura Files`).catch(() => {});
    persistSession();
    startWatching(target);
  }
}

/** Load the directory the tab points at. */
export async function loadDirectory(tab, selectPath = '') {
  if (!tab) return;
  const requestId = ++tab.requestId;
  tab.loading = true;
  tab.error = '';

  try {
    const listing = await FileService.List(tab.path, listOptions(tab));
    if (requestId !== tab.requestId) return; // a newer navigation won
    tab.listing = listing;
    tab.entries = listing?.entries || [];
    tab.title = listing?.name || tab.path || 'Home';
    tab.error = listing?.error || '';
    pruneSelection(tab);
    if (selectPath) {
      const match = tab.entries.find((entry) => entry.path === selectPath);
      if (match) {
        tab.selected.clear();
        tab.selected.add(match.path);
        tab.cursor = match.path;
        tab.anchor = match.path;
      }
    } else if (tab.selected.size === 0 && tab.entries.length && !tab.cursor) {
      tab.cursor = tab.entries[0].path;
    }
  } catch (error) {
    if (requestId !== tab.requestId) return;
    tab.error = describeError(error);
    tab.entries = [];
  } finally {
    if (requestId === tab.requestId) tab.loading = false;
  }
}

/** Re-read the current directory of a tab, keeping the selection. */
export async function refreshTab(tab = activeTab.value) {
  if (!tab) return;
  if (tab.mode === 'trash') return loadTrash();
  if (tab.mode === 'archive') return loadArchive(tab, tab.archivePath);
  await loadDirectory(tab);
}

function pruneSelection(tab) {
  const present = new Set(tab.entries.map((entry) => entry.path));
  [...tab.selected].forEach((path) => {
    if (!present.has(path)) tab.selected.delete(path);
  });
}

export function canGoBack(tab) {
  return !!tab && tab.historyIndex > 0;
}

export function canGoForward(tab) {
  return !!tab && tab.historyIndex >= 0 && tab.historyIndex < tab.history.length - 1;
}

export function goBack(tab) {
  if (!canGoBack(tab)) return;
  tab.historyIndex -= 1;
  navigate(tab, tab.history[tab.historyIndex], { skipHistory: true });
}

export function goForward(tab) {
  if (!canGoForward(tab)) return;
  tab.historyIndex += 1;
  navigate(tab, tab.history[tab.historyIndex], { skipHistory: true });
}

export async function goUp(tab = activeTab.value) {
  if (!tab) return;
  const parent = await FileService.Parent(tab.path);
  if (parent) await navigate(tab, parent, { select: tab.path });
}

/* ------------------------------------------------------------------------ */
/* Live folder watching                                                      */
/* ------------------------------------------------------------------------ */

let watchTarget = '';

async function startWatching(path) {
  if (!state.settings.watchEnabled) return;
  if (watchTarget === path) return;
  watchTarget = path;
  try {
    await WatchService.Watch(path);
  } catch {
    watchTarget = '';
  }
}

/* ------------------------------------------------------------------------ */
/* Tabs                                                                      */
/* ------------------------------------------------------------------------ */

export function newTab(path = '', focus = true) {
  const target = path || activeTab.value?.path || state.info?.os?.home || '';
  const tab = createTab(target);
  const index = state.tabs.findIndex((item) => item.id === state.activeTabId);
  state.tabs.splice(index + 1, 0, tab);
  if (focus) selectTab(tab.id);
  navigate(tab, target, { replace: true });
  return tab;
}

export function selectTab(id) {
  state.activeTabId = id;
  const tab = tabById(id);
  if (tab) {
    AppService.SetTitle(`${tab.listing?.name || tab.path || 'Mura Files'} — Mura Files`).catch(() => {});
    if (tab.mode === 'browse') startWatching(tab.path);
  }
  persistSession();
}

export function closeTab(id) {
  const index = state.tabs.findIndex((tab) => tab.id === id);
  if (index < 0) return;
  const wasActive = state.activeTabId === id;
  state.tabs.splice(index, 1);

  if (state.tabs.length === 0) {
    const tab = createTab(state.info?.os?.home || '');
    state.tabs.push(tab);
    state.activeTabId = tab.id;
    navigate(tab, tab.path, { replace: true });
    persistSession();
    return;
  }
  if (wasActive) {
    const next = state.tabs[Math.min(index, state.tabs.length - 1)];
    selectTab(next.id);
  }
  persistSession();
}

export function closeOtherTabs(id) {
  const keep = tabById(id);
  if (!keep) return;
  state.tabs = [keep];
  selectTab(keep.id);
}

export function closeTabsToRight(id) {
  const index = state.tabs.findIndex((tab) => tab.id === id);
  if (index < 0) return;
  state.tabs.splice(index + 1);
  if (!tabById(state.activeTabId)) selectTab(id);
  persistSession();
}

export function duplicateTab(id) {
  const tab = tabById(id);
  if (!tab) return;
  newTab(tab.path);
}

export function reorderTabs(fromId, toIndex) {
  const from = state.tabs.findIndex((tab) => tab.id === fromId);
  if (from < 0) return;
  const [moved] = state.tabs.splice(from, 1);
  state.tabs.splice(clamp(toIndex, 0, state.tabs.length), 0, moved);
  persistSession();
}

/* ------------------------------------------------------------------------ */
/* Selection                                                                 */
/* ------------------------------------------------------------------------ */

/**
 * Apply a click selection honouring the usual modifier keys.
 *
 * @param {object} tab
 * @param {object} entry
 * @param {{ ctrl?: boolean, shift?: boolean, meta?: boolean }} mods
 */
export function selectEntry(tab, entry, mods = {}) {
  if (!tab || !entry) return;
  const additive = mods.ctrl || mods.meta;

  if (mods.shift && tab.anchor) {
    const anchorIndex = tab.entries.findIndex((item) => item.path === tab.anchor);
    const targetIndex = tab.entries.findIndex((item) => item.path === entry.path);
    if (anchorIndex >= 0 && targetIndex >= 0) {
      const [start, end] = anchorIndex < targetIndex
        ? [anchorIndex, targetIndex]
        : [targetIndex, anchorIndex];
      if (!additive) tab.selected.clear();
      for (let i = start; i <= end; i += 1) tab.selected.add(tab.entries[i].path);
      tab.cursor = entry.path;
      return;
    }
  }

  if (additive) {
    if (tab.selected.has(entry.path)) tab.selected.delete(entry.path);
    else tab.selected.add(entry.path);
    tab.anchor = entry.path;
    tab.cursor = entry.path;
    return;
  }

  if (tab.selected.size === 1 && tab.selected.has(entry.path)) return;
  tab.selected.clear();
  tab.selected.add(entry.path);
  tab.anchor = entry.path;
  tab.cursor = entry.path;
}

export function setSelection(tab, paths) {
  if (!tab) return;
  tab.selected.clear();
  paths.forEach((path) => tab.selected.add(path));
  tab.cursor = paths[paths.length - 1] || '';
}

export function selectAll(tab = activeTab.value) {
  if (!tab) return;
  tab.entries.forEach((entry) => tab.selected.add(entry.path));
}

export function clearSelection(tab = activeTab.value) {
  if (!tab) return;
  tab.selected.clear();
}

export function invertSelection(tab = activeTab.value) {
  if (!tab) return;
  tab.entries.forEach((entry) => {
    if (tab.selected.has(entry.path)) tab.selected.delete(entry.path);
    else tab.selected.add(entry.path);
  });
}

/** Extend the selection using the keyboard cursor. */
export function moveCursor(tab, delta, extend = false) {
  if (!tab || !tab.entries.length) return;
  const current = tab.entries.findIndex((entry) => entry.path === tab.cursor);
  const next = clamp((current < 0 ? 0 : current + delta), 0, tab.entries.length - 1);
  const entry = tab.entries[next];
  if (!entry) return;
  if (extend) {
    if (!tab.anchor) tab.anchor = tab.cursor || entry.path;
    selectEntry(tab, entry, { shift: true });
  } else {
    selectEntry(tab, entry, {});
  }
  tab.cursor = entry.path;
  scrollEntryIntoView(entry.path);
}

function scrollEntryIntoView(path) {
  requestAnimationFrame(() => {
    const element = document.querySelector(`[data-path="${cssEscape(path)}"]`);
    if (element && element.scrollIntoView) {
      element.scrollIntoView({ block: 'nearest' });
    }
  });
}

function cssEscape(value) {
  if (window.CSS && window.CSS.escape) return window.CSS.escape(value);
  return String(value).replace(/(["\\])/g, '\\$1');
}

/* ------------------------------------------------------------------------ */
/* Opening entries                                                           */
/* ------------------------------------------------------------------------ */

export async function openEntry(tab, entry) {
  if (!tab || !entry) return;
  if (entry.isDir) {
    if (entry.isSymlink && entry.linkBroken) {
      toast.warn(`${entry.name} points to a missing location`);
      return;
    }
    await navigate(tab, entry.path);
    return;
  }
  if (entry.isSymlink && entry.linkBroken) {
    toast.warn(`${entry.name} is a broken link`);
    return;
  }
  if (entry.ext === '.desktop') {
    const result = await FileService.LaunchDesktop(entry.path);
    if (!result?.ok) toast.error(result?.message || 'Could not launch the application');
    return;
  }
  const result = await AppService.Open(entry.path);
  if (!result?.ok) toast.error(result?.message || 'Could not open the file');
}

/** Open a path or select it when it points at a file. */
export async function openPathsInNewTab(paths) {
  const first = paths[0];
  if (!first) return;
  const entry = await FileService.Stat(first);
  if (entry?.isDir) {
    newTab(entry.path);
  } else {
    const tab = newTab(await FileService.Parent(first), false);
    navigate(tab, await FileService.Parent(first), { replace: true, select: first });
  }
}

/* ------------------------------------------------------------------------ */
/* External drag and drop                                                    */
/* ------------------------------------------------------------------------ */

async function handleExternalDrop(paths, targetId) {
  const tab = activeTab.value;
  if (!tab) return;

  // A drop onto a folder row moves the files there; otherwise it copies them
  // into the folder being browsed.
  let destination = tab.path;
  const element = targetId ? document.querySelector(`[data-path="${cssEscape(targetId)}"]`) : null;
  if (element) destination = element.dataset.dropPath || tab.path;

  const op = await OperationService.Copy(paths, destination, transferOptions({ conflict: 'rename' }));
  if (op) toast.info(`Copying ${paths.length} item(s) into ${destination.split('/').pop() || destination}`);
}

/* ------------------------------------------------------------------------ */
/* Clipboard                                                                 */
/* ------------------------------------------------------------------------ */

export function clipboardCopy(tab = activeTab.value) {
  const paths = [...(tab?.selected || [])];
  if (!paths.length) return;
  state.clipboard = { mode: 'copy', paths };
  toast.info(`${paths.length} item(s) copied`);
}

export function clipboardCut(tab = activeTab.value) {
  const paths = [...(tab?.selected || [])];
  if (!paths.length) return;
  state.clipboard = { mode: 'cut', paths };
  toast.info(`${paths.length} item(s) cut`);
}

export function clipboardClear() {
  state.clipboard = { mode: '', paths: [] };
}

export const clipboardHasItems = computed(() => state.clipboard.paths.length > 0);

export async function clipboardPaste(tab = activeTab.value) {
  if (!tab) return;
  const { mode, paths } = state.clipboard;
  if (!paths.length) return;

  const options = transferOptions({ conflict: 'rename', preserveAttrs: true });
  let opId = '';
  try {
    opId = mode === 'cut'
      ? await OperationService.Move(paths, tab.path, options)
      : await OperationService.Copy(paths, tab.path, options);
  } catch (error) {
    toast.error(describeError(error));
    return;
  }
  if (!opId) return;
  if (mode === 'cut') clipboardClear();
  toast.info(`${mode === 'cut' ? 'Moving' : 'Copying'} ${paths.length} item(s)`);
}

/* ------------------------------------------------------------------------ */
/* File operations                                                           */
/* ------------------------------------------------------------------------ */

function upsertOperation(progress) {
  if (!progress) return;
  const index = state.ops.findIndex((op) => op.id === progress.id);
  if (index >= 0) state.ops.splice(index, 1, progress);
  else state.ops.push(progress);
  // Keep the list bounded so long sessions do not grow forever.
  if (state.ops.length > 40) state.ops.splice(0, state.ops.length - 40);
}

function refreshActiveAfterOperation(progress) {
  if (progress.status === 'running') return;
  if (progress.kind === 'trash' || progress.kind === 'delete') {
    state.trash = state.trash.filter((entry) => !progress.current?.includes(entry.name));
  }
  state.tabs.forEach((tab) => {
    if (tab.mode === 'browse' && !tab.loading) scheduleReload(tab);
  });
}

export async function cancelOperation(id) {
  await OperationService.Cancel(id);
}

export async function clearFinishedOperations() {
  await OperationService.ClearFinished();
  state.ops = state.ops.filter((op) => op.status === 'running' || op.status === 'queued');
}

export async function createFolder(tab = activeTab.value) {
  if (!tab) return;
  openPrompt({
    title: 'New Folder',
    label: 'Folder name',
    value: 'Untitled Folder',
    confirmLabel: 'Create',
    selectStem: true,
    onConfirm: async (name) => {
      try {
        const entry = await FileService.CreateDir(tab.path, name);
        await loadDirectory(tab);
        if (entry?.path) {
          tab.selected.clear();
          tab.selected.add(entry.path);
          startRename(entry.path, entry.name);
        }
      } catch (error) {
        toast.error(describeError(error));
      }
    },
  });
}

export async function createFile(tab = activeTab.value) {
  if (!tab) return;
  openPrompt({
    title: 'New File',
    label: 'File name',
    value: 'untitled.txt',
    confirmLabel: 'Create',
    selectStem: true,
    onConfirm: async (name) => {
      try {
        const entry = await FileService.CreateFile(tab.path, name, '');
        await loadDirectory(tab);
        if (entry?.path) {
          tab.selected.clear();
          tab.selected.add(entry.path);
          startRename(entry.path, entry.name);
        }
      } catch (error) {
        toast.error(describeError(error));
      }
    },
  });
}

export function startRename(path, value) {
  state.renaming = { path, value: value ?? (path.split('/').pop() || '') };
}

export function cancelRename() {
  state.renaming = { path: '', value: '' };
}

export async function commitRename(newName) {
  const { path } = state.renaming;
  if (!path) return;
  const current = path.split('/').pop();
  state.renaming = { path: '', value: '' };
  if (!newName || newName === current) return;

  try {
    const entry = await FileService.Rename(path, newName);
    const tab = activeTab.value;
    if (tab) {
      await loadDirectory(tab, entry?.path);
    }
  } catch (error) {
    toast.error(describeError(error));
  }
}

export async function duplicateSelection(tab = activeTab.value) {
  const paths = [...(tab?.selected || [])];
  if (!paths.length) return;
  try {
    await OperationService.Duplicate(paths);
    toast.info(`Duplicating ${paths.length} item(s)`);
  } catch (error) {
    toast.error(describeError(error));
  }
}

/** Delete the selection, routing through the trash unless forced. */
export async function deleteSelection(tab = activeTab.value, force = false) {
  const paths = [...(tab?.selected || [])];
  if (!paths.length) return;
  const useTrash = !force && state.settings.useTrash;
  const needsConfirm = force
    ? state.settings.confirmDelete
    : (state.settings.confirmDelete || state.settings.confirmTrash);

  const run = async () => {
    try {
      await OperationService.Delete(paths, deleteOptions({ permanent: !useTrash, skipErrors: true }));
      if (tab) tab.selected.clear();
      toast.info(`${useTrash ? 'Moving to Trash' : 'Deleting'} ${paths.length} item(s)`);
    } catch (error) {
      toast.error(describeError(error));
    }
  };

  if (!needsConfirm) {
    await run();
    return;
  }
  openConfirm({
    title: useTrash ? 'Move to Trash?' : 'Delete permanently?',
    message: paths.length === 1
      ? `\u201c${paths[0].split('/').pop()}\u201d will be ${useTrash ? 'moved to the Trash' : 'deleted permanently'}.`
      : `${paths.length} items will be ${useTrash ? 'moved to the Trash' : 'deleted permanently'}.`,
    detail: useTrash ? 'You can restore them from the Trash later.' : 'This cannot be undone.',
    confirmLabel: useTrash ? 'Move to Trash' : 'Delete',
    danger: !useTrash,
    onConfirm: run,
  });
}

/** Copy or move the selection into a folder chosen through a native dialog. */
export async function transferSelectionTo(tab = activeTab.value, move = false) {
  const paths = [...(tab?.selected || [])];
  if (!paths.length) return;
  const destination = await AppService.ChooseFolder(
    move ? 'Move to folder' : 'Copy to folder',
    tab?.path || '',
  );
  if (!destination) return;
  try {
    const options = transferOptions({ conflict: 'rename' });
    if (move) await OperationService.Move(paths, destination, options);
    else await OperationService.Copy(paths, destination, options);
    toast.info(`${move ? 'Moving' : 'Copying'} ${paths.length} item(s)`);
  } catch (error) {
    toast.error(describeError(error));
  }
}

/** Move the selection into a folder inside the current directory. */
export async function dropInto(destination, paths) {
  if (!destination || !paths?.length) return;
  const sources = paths.filter((path) => {
    if (path === destination) return false;
    return !destination.startsWith(`${path}/`);
  });
  if (!sources.length) return;
  try {
    await OperationService.Move(sources, destination, transferOptions({ conflict: 'rename' }));
    toast.info(`Moving ${sources.length} item(s) to ${destination.split('/').pop()}`);
  } catch (error) {
    toast.error(describeError(error));
  }
}

export async function createArchive(tab = activeTab.value, format = 'zip') {
  const paths = [...(tab?.selected || [])];
  if (!paths.length) return;
  const base = paths.length === 1
    ? paths[0].split('/').pop().replace(/\.[^.]+$/, '')
    : 'archive';
  openPrompt({
    title: 'Create Archive',
    label: 'Archive name',
    value: `${base}.${format}`,
    confirmLabel: 'Create',
    selectStem: true,
    hint: 'Supported: zip, tar, tar.gz, tar.bz2',
    onConfirm: async (name) => {
      const destination = `${tab.path}/${name}`;
      try {
        await OperationService.Compress(paths, destination, format);
        toast.info(`Creating ${name}`);
      } catch (error) {
        toast.error(describeError(error));
      }
    },
  });
}

export async function extractSelection(tab = activeTab.value) {
  const entries = tab?.entries.filter((entry) => tab.selected.has(entry.path)) || [];
  const archives = entries.filter((entry) => !entry.isDir && isArchiveName(entry.name));
  if (!archives.length) {
    toast.warn('Select an archive to extract');
    return;
  }
  try {
    await OperationService.Extract(archives.map((entry) => entry.path), '');
    toast.info(`Extracting ${archives.length} archive(s)`);
  } catch (error) {
    toast.error(describeError(error));
  }
}

export function isArchiveName(name) {
  return /\.(zip|tar|tar\.gz|tgz|tar\.bz2|tbz2|tar\.xz|txz|tar\.zst|gz|7z|rar|jar|war|deb|rpm|iso|apk)$/i.test(name);
}

/* ------------------------------------------------------------------------ */
/* Search                                                                    */
/* ------------------------------------------------------------------------ */

export async function startSearch(tab = activeTab.value, pattern) {
  if (!tab) return;
  const query = pattern ?? tab.searchPattern;
  if (!String(query || '').trim()) {
    await stopSearch(tab);
    return;
  }
  if (tab.searchRunning && tab.searchId) {
    await SearchService.Cancel(tab.searchId).catch(() => {});
  }

  tab.mode = 'search';
  tab.searchPattern = query;
  tab.entries = [];
  tab.selected.clear();
  tab.searchRunning = true;
  tab.searchDone = false;
  tab.searchScanned = 0;
  tab.searchFound = 0;
  tab.searchElapsed = 0;
  tab.error = '';

  const options = tab.searchOptions || {};
  const request = searchQuery({
    roots: [tab.path],
    pattern: String(query),
    mode: options.mode || 'glob',
    matchCase: !!options.matchCase,
    includeHidden: !!options.includeHidden || !!tab.showHidden,
    includeFiles: true,
    includeDirs: true,
    content: !!options.includeContent,
    followLinks: !!state.settings.followLinks,
  });

  try {
    tab.searchId = await SearchService.Start(request);
  } catch (error) {
    tab.searchRunning = false;
    tab.error = describeError(error);
  }
}

export async function stopSearch(tab = activeTab.value) {
  if (!tab) return;
  if (tab.searchId) await SearchService.Cancel(tab.searchId).catch(() => {});
  tab.searchRunning = false;
  tab.searchId = '';
  tab.mode = 'browse';
  await loadDirectory(tab);
}

function handleSearchBatch(batch) {
  if (!batch) return;
  const tab = state.tabs.find((item) => item.searchId === batch.id);
  if (!tab) return;

  if (batch.entries?.length) {
    tab.entries = tab.entries.concat(batch.entries);
  }
  tab.searchScanned = batch.scanned || tab.searchScanned;
  tab.searchFound = batch.found || tab.searchFound;
  tab.searchElapsed = batch.elapsed || tab.searchElapsed;

  if (batch.done) {
    tab.searchRunning = false;
    tab.searchDone = true;
    if (batch.error && batch.error !== 'Search cancelled') {
      tab.error = batch.error;
    }
  }
}

export async function searchHere(pattern, options = {}) {
  const tab = activeTab.value;
  if (!tab) return;
  tab.searchOptions = Object.assign({}, tab.searchOptions, options);
  await startSearch(tab, pattern);
}

/* ------------------------------------------------------------------------ */
/* Archives                                                                  */
/* ------------------------------------------------------------------------ */

export async function openArchive(tab, path) {
  if (!tab) return;
  tab.mode = 'archive';
  tab.archivePath = path;
  tab.loading = true;
  try {
    const info = await FileService.ArchiveInfo(path);
    tab.archiveInfo = info;
    tab.title = path.split('/').pop();
    tab.entries = (info?.entries || []).map((entry) => ({
      name: entry.name,
      path: `${path}::${entry.path}`,
      dir: path,
      ext: '',
      isDir: entry.isDir,
      isSymlink: false,
      isHidden: entry.name.startsWith('.'),
      size: entry.size,
      modTime: entry.modTime,
      kind: entry.isDir ? 'directory' : 'other',
      icon: entry.isDir ? 'folder' : 'file-archive',
      thumb: false,
      mime: '',
      perm: entry.mode || '',
      owner: '',
      group: '',
      itemCount: 0,
      archiveMember: entry.path,
    }));
    tab.error = '';
  } catch (error) {
    tab.error = describeError(error);
    tab.entries = [];
  } finally {
    tab.loading = false;
  }
}

async function loadArchive(tab, path) {
  await openArchive(tab, path);
}

/* ------------------------------------------------------------------------ */
/* Trash                                                                     */
/* ------------------------------------------------------------------------ */

export async function loadTrash() {
  state.trashLoading = true;
  try {
    state.trash = (await AppService.ListTrash()) || [];
  } catch (error) {
    toast.error(describeError(error));
    state.trash = [];
  } finally {
    state.trashLoading = false;
  }
}

export async function showTrash() {
  const tab = activeTab.value;
  if (!tab) return;
  tab.mode = 'trash';
  tab.title = 'Trash';
  tab.selected.clear();
  tab.entries = [];
  await loadTrash();
}

export async function restoreTrashItems(infoPaths) {
  if (!infoPaths.length) return;
  let restored = 0;
  for (const infoPath of infoPaths) {
    try {
      await AppService.RestoreTrash(infoPath);
      restored += 1;
    } catch (error) {
      toast.error(describeError(error));
    }
  }
  await loadTrash();
  if (restored) toast.success(`Restored ${restored} item(s)`);
}

export async function deleteTrashItems(infoPaths, permanent = true) {
  if (!infoPaths.length) return;
  const run = async () => {
    for (const infoPath of infoPaths) {
      try {
        await AppService.DeleteTrashEntry(infoPath);
      } catch (error) {
        toast.error(describeError(error));
      }
    }
    await loadTrash();
  };
  if (!permanent) {
    await run();
    return;
  }
  openConfirm({
    title: 'Delete permanently?',
    message: infoPaths.length === 1
      ? 'This item will be removed from the Trash permanently.'
      : `${infoPaths.length} items will be removed from the Trash permanently.`,
    detail: 'This cannot be undone.',
    confirmLabel: 'Delete',
    danger: true,
    onConfirm: run,
  });
}

export async function emptyTrash() {
  openConfirm({
    title: 'Empty Trash?',
    message: 'Every item in the Trash will be deleted permanently.',
    detail: 'This cannot be undone.',
    confirmLabel: 'Empty Trash',
    danger: true,
    onConfirm: async () => {
      try {
        const count = await AppService.EmptyTrash();
        await loadTrash();
        toast.success(`Removed ${count} item(s) from the Trash`);
      } catch (error) {
        toast.error(describeError(error));
      }
    },
  });
}

/* ------------------------------------------------------------------------ */
/* Dialogs                                                                   */
/* ------------------------------------------------------------------------ */

export function openPrompt(config) {
  state.dialogs.prompt = Object.assign({
    title: 'Rename',
    label: 'Name',
    value: '',
    hint: '',
    confirmLabel: 'OK',
    selectStem: false,
    onConfirm: null,
  }, config);
}

export function closePrompt() {
  state.dialogs.prompt = null;
}

export function openConfirm(config) {
  state.dialogs.confirm = Object.assign({
    title: 'Are you sure?',
    message: '',
    detail: '',
    confirmLabel: 'Confirm',
    cancelLabel: 'Cancel',
    danger: false,
    onConfirm: null,
  }, config);
}

export function closeConfirm() {
  state.dialogs.confirm = null;
}

export async function showProperties(paths) {
  const list = Array.isArray(paths) ? paths : [paths];
  if (!list.length) return;
  state.dialogs.properties = { loading: true, paths: list, entries: [], count: null, usage: null, checksum: '' };
  try {
    const [entries, count] = await Promise.all([
      FileService.StatMany(list),
      FileService.Count(list, 200000),
    ]);
    const first = entries[0];
    const usage = first ? await FileService.Usage(first.path) : null;
    state.dialogs.properties = { loading: false, paths: list, entries, count, usage, checksum: '' };
  } catch (error) {
    toast.error(describeError(error));
    state.dialogs.properties = null;
  }
}

export function closeProperties() {
  state.dialogs.properties = null;
}

export async function computeChecksum(path, algorithm) {
  if (!state.dialogs.properties) return;
  try {
    state.dialogs.properties.checksum = await FileService.Checksum(path, algorithm);
  } catch (error) {
    toast.error(describeError(error));
  }
}

export function showAppPicker(entry) {
  state.dialogs.appPicker = { entry, apps: [], loading: true };
  FileService.Applications(entry?.path || '').then((apps) => {
    if (state.dialogs.appPicker) {
      state.dialogs.appPicker.apps = apps || [];
      state.dialogs.appPicker.loading = false;
    }
  }).catch(() => {
    if (state.dialogs.appPicker) state.dialogs.appPicker.loading = false;
  });
}

export function closeAppPicker() {
  state.dialogs.appPicker = null;
}

export async function launchApp(app, entry) {
  const result = await AppService.OpenWith(app.path || app.name, entry.path);
  if (!result?.ok) toast.error(result?.message || 'Could not launch the application');
  closeAppPicker();
}

/* ------------------------------------------------------------------------ */
/* Context menu                                                              */
/* ------------------------------------------------------------------------ */

const MENU_MARGIN = 10;

export function openContextMenu(event, items) {
  const x = Math.min(event.clientX, window.innerWidth - 240 - MENU_MARGIN);
  const y = Math.min(event.clientY, window.innerHeight - Math.min(items.length * 31 + 16, 420) - MENU_MARGIN);
  state.contextMenu = { open: true, x: Math.max(MENU_MARGIN, x), y: Math.max(MENU_MARGIN, y), items };
}

export function closeContextMenu() {
  if (state.contextMenu.open) state.contextMenu = { open: false, x: 0, y: 0, items: [] };
}

/* ------------------------------------------------------------------------ */
/* Sidebar                                                                   */
/* ------------------------------------------------------------------------ */

export async function addBookmark(path = activeTab.value?.path) {
  if (!path) return;
  try {
    state.bookmarks = await AppService.AddBookmark('', path, 'bookmark') || [];
    toast.success('Added to bookmarks');
  } catch (error) {
    toast.error(describeError(error));
  }
}

export async function removeBookmark(path) {
  try {
    state.bookmarks = await AppService.RemoveBookmark(path) || [];
  } catch (error) {
    toast.error(describeError(error));
  }
}

export async function renameBookmark(path, name) {
  try {
    state.bookmarks = await AppService.RenameBookmark(path, name) || [];
  } catch (error) {
    toast.error(describeError(error));
  }
}

export async function reorderBookmarks(paths) {
  try {
    state.bookmarks = await AppService.ReorderBookmarks(paths) || [];
  } catch (error) {
    toast.error(describeError(error));
  }
}

export async function refreshDrives() {
  try {
    state.drives = (await FileService.Drives()) || [];
  } catch {
    state.drives = [];
  }
}

export async function refreshPlaces() {
  try {
    state.places = (await FileService.WellKnown()) || [];
  } catch {
    state.places = [];
  }
}

export async function ejectDrive(drive) {
  try {
    await AppService.Eject(drive.device || drive.path);
    await refreshDrives();
    toast.success(`Ejected ${drive.name}`);
  } catch (error) {
    toast.error(describeError(error));
  }
}

/* ------------------------------------------------------------------------ */
/* Folder tree (sidebar)                                                     */
/* ------------------------------------------------------------------------ */

export async function toggleTreeNode(path) {
  const expanded = state.tree.expanded;
  if (expanded[path]) {
    delete expanded[path];
    return;
  }
  expanded[path] = true;
  await loadTreeNode(path);
}

export async function loadTreeNode(path) {
  if (state.tree.children[path]) return;
  state.tree.loading[path] = true;
  try {
    const listing = await FileService.List(path, listOptions({
      ...state.settings,
      showHidden: state.settings.showHidden,
    }));
    const dirs = (listing?.entries || [])
      .filter((entry) => entry.isDir && !entry.linkBroken)
      .slice(0, 400);
    state.tree.children[path] = dirs;
  } catch {
    state.tree.children[path] = [];
  } finally {
    delete state.tree.loading[path];
  }
}

export function expandTreeTo(path) {
  if (!state.settings.autoExpandTree || !path) return;
  const parts = path.split('/').filter(Boolean);
  let current = '';
  parts.slice(0, -1).forEach((part) => {
    current += `/${part}`;
    state.tree.expanded[current] = true;
    loadTreeNode(current);
  });
}

/* ------------------------------------------------------------------------ */
/* Command palette                                                           */
/* ------------------------------------------------------------------------ */

export function togglePalette(open) {
  state.palette.open = open === undefined ? !state.palette.open : !!open;
  if (state.palette.open) {
    state.palette.query = '';
    state.palette.index = 0;
  }
}

export function openDialog(name) {
  if (name in state.dialogs) state.dialogs[name] = true;
}

export function closeDialog(name) {
  if (name in state.dialogs) state.dialogs[name] = false;
}

/* ------------------------------------------------------------------------ */
/* Sorting and view helpers                                                  */
/* ------------------------------------------------------------------------ */

export async function setSort(tab, sortBy, keepDirection = false) {
  if (!tab) return;
  if (tab.sortBy === sortBy && !keepDirection) tab.sortDesc = !tab.sortDesc;
  else tab.sortBy = sortBy;
  if (tab.mode === 'browse') await loadDirectory(tab);
  else tab.entries = await FileService.SortEntries(tab.entries, listOptions(tab));
}

export async function setViewMode(tab, mode) {
  if (!tab) return;
  tab.viewMode = mode;
  updateSettings({ viewMode: mode });
}

export async function toggleHidden(tab = activeTab.value) {
  if (!tab) return;
  tab.showHidden = !tab.showHidden;
  updateSettings({ showHidden: tab.showHidden });
  if (tab.mode === 'browse') await loadDirectory(tab);
}

/** Applies the current tab's sort preferences as the global default. */
export function saveViewDefaults(tab = activeTab.value) {
  if (!tab) return;
  updateSettings({
    sortBy: tab.sortBy,
    sortDesc: tab.sortDesc,
    dirsFirst: tab.dirsFirst,
    groupByKind: tab.groupByKind,
    viewMode: tab.viewMode,
    showHidden: tab.showHidden,
  }, true);
  toast.success('View settings saved as defaults');
}

/* ------------------------------------------------------------------------ */
/* Miscellaneous                                                             */
/* ------------------------------------------------------------------------ */

export async function openInTerminal(path) {
  const result = await AppService.OpenTerminal(path);
  if (!result?.ok) toast.error(result?.message || 'No terminal emulator found');
}

export async function revealInFileManager(path) {
  await AppService.RevealInFileManager(path);
}

export async function copyPathToClipboard(path) {
  await AppService.CopyText(path);
  toast.success('Path copied');
}

export async function copyNameToClipboard(name) {
  await AppService.CopyText(name);
  toast.success('Name copied');
}

export async function copyFilesToSystemClipboard(paths) {
  try {
    await AppService.CopyFilesToClipboard(paths, false);
  } catch {
    /* the desktop clipboard may be unavailable; the internal one still works */
  }
}

export function focusPath(path) {
  const tab = activeTab.value;
  if (!tab) return;
  if (tab.path === path) {
    refreshTab(tab);
    return;
  }
  navigate(tab, path);
}

export function quit() {
  AppService.Quit().catch(() => {});
}

// Persist the session whenever the tab set changes.
watch(() => state.tabs.length, persistSession);

/* ------------------------------------------------------------------------ */
/* Context menu construction                                                 */
/* ------------------------------------------------------------------------ */

/**
 * Build the context menu for a file or folder row.
 *
 * Entries adapt to what is selected, so the menu never offers an action that
 * cannot apply - which is the main usability difference between a good file
 * manager and a frustrating one.
 */
export function buildEntryMenu(entry) {
  const tab = activeTab.value;
  const selection = [...(tab?.selected || [])];
  const multiple = selection.length > 1;
  const isArchive = !entry.isDir && isArchiveName(entry.name);
  const items = [];

  items.push({ label: 'Open', icon: 'external-link', run: () => openEntry(tab, entry) });
  items.push({ label: 'Open With…', icon: 'sparkles', run: () => showAppPicker(entry) });

  if (entry.isDir) {
    items.push({ label: 'Open in New Tab', icon: 'plus', run: () => newTab(entry.path) });
    items.push({ label: 'Open in Terminal', icon: 'terminal', run: () => openInTerminal(entry.path) });
  }

  if (isArchive) {
    items.push({ label: 'Extract Here', icon: 'package', run: () => extractSelection(tab) });
    items.push({ label: 'Show Contents', icon: 'layout-list', run: () => openArchive(tab, entry.path) });
  }

  items.push({ separator: true });

  items.push({ label: 'Cut', icon: 'scissors', shortcut: 'Ctrl+X', disabled: !selection.length, run: () => clipboardCut(tab) });
  items.push({ label: 'Copy', icon: 'copy', shortcut: 'Ctrl+C', disabled: !selection.length, run: () => clipboardCopy(tab) });
  items.push({
    label: entry.isDir && !multiple ? 'Paste Into Folder' : 'Paste',
    icon: 'paste',
    shortcut: 'Ctrl+V',
    disabled: !clipboardHasItems.value,
    run: () => (entry.isDir && !multiple ? clipboardPasteInto(entry.path) : clipboardPaste(tab)),
  });
  items.push({ label: 'Duplicate', icon: 'copy-plus', shortcut: 'Ctrl+D', disabled: !selection.length, run: () => duplicateSelection(tab) });
  items.push({ label: 'Rename…', icon: 'pencil', shortcut: 'F2', disabled: multiple, run: () => startRename(entry.path, entry.name) });

  items.push({ separator: true });

  items.push({ label: 'Move To…', icon: 'folder-plus', disabled: !selection.length, run: () => transferSelectionTo(tab, true) });
  items.push({ label: 'Copy To…', icon: 'folder-plus', disabled: !selection.length, run: () => transferSelectionTo(tab, false) });
  items.push({
    label: 'Compress…',
    icon: 'file-archive',
    disabled: !selection.length,
    run: () => createArchive(tab, 'zip'),
  });

  items.push({ separator: true });

  items.push({ label: 'Copy Path', icon: 'link', disabled: multiple, run: () => copyPathToClipboard(entry.path) });
  items.push({ label: 'Copy Name', icon: 'tag', disabled: multiple, run: () => copyNameToClipboard(entry.name) });
  items.push({
    label: 'Copy to System Clipboard',
    icon: 'clipboard',
    disabled: !selection.length,
    run: () => copyFilesToSystemClipboard(selection),
  });
  if (entry.isDir) {
    items.push({ label: 'Add to Bookmarks', icon: 'bookmark', disabled: multiple, run: () => addBookmark(entry.path) });
  }

  items.push({ separator: true });

  items.push({ label: 'Move to Trash', icon: 'trash', shortcut: 'Del', danger: true, disabled: !selection.length, run: () => deleteSelection(tab) });
  items.push({ label: 'Delete Permanently', icon: 'trash-2', shortcut: 'Shift+Del', danger: true, disabled: !selection.length, run: () => deleteSelection(tab, true) });

  items.push({ separator: true });

  items.push({ label: 'Properties', icon: 'info', shortcut: 'Alt+Enter', run: () => showProperties(selection.length ? selection : [entry.path]) });

  return items;
}

/** Paste the clipboard into a specific folder rather than the current one. */
export async function clipboardPasteInto(destination) {
  const { mode, paths } = state.clipboard;
  if (!paths.length) return;
  try {
    const options = transferOptions({ conflict: 'rename' });
    if (mode === 'cut') await OperationService.Move(paths, destination, options);
    else await OperationService.Copy(paths, destination, options);
    if (mode === 'cut') clipboardClear();
    toast.info(`${mode === 'cut' ? 'Moving' : 'Copying'} ${paths.length} item(s)`);
  } catch (error) {
    toast.error(describeError(error));
  }
}

/** Context menu for the empty area of the file view. */
export function buildBackgroundMenu() {
  const tab = activeTab.value;
  return [
    { label: 'New Folder', icon: 'folder-plus', shortcut: 'Ctrl+Shift+N', run: () => createFolder(tab) },
    { label: 'New File', icon: 'file-plus', shortcut: 'Ctrl+Alt+N', run: () => createFile(tab) },
    { separator: true },
    { label: 'Paste', icon: 'paste', shortcut: 'Ctrl+V', disabled: !clipboardHasItems.value, run: () => clipboardPaste(tab) },
    { label: 'Select All', icon: 'check', shortcut: 'Ctrl+A', run: () => selectAll(tab) },
    { separator: true },
    { label: 'Open in Terminal', icon: 'terminal', run: () => openInTerminal(tab?.path) },
    { label: 'Add to Bookmarks', icon: 'bookmark', run: () => addBookmark(tab?.path) },
    { label: 'Refresh', icon: 'refresh', shortcut: 'F5', run: () => refreshTab(tab) },
    { separator: true },
    { label: 'Properties', icon: 'info', run: () => showProperties([tab?.path]) },
  ];
}

/* ------------------------------------------------------------------------ */
/* Sidebar search helpers                                                    */
/* ------------------------------------------------------------------------ */

/** Quick lookup of the entry for a path inside the active tab. */
export function entryForPath(path) {
  return activeTab.value?.entries.find((entry) => entry.path === path) || null;
}

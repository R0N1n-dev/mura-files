<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue';
import Icon from './Icon.vue';
import TitleBar from './TitleBar.vue';
import TabBar from './TabBar.vue';
import Toolbar from './Toolbar.vue';
import Sidebar from './Sidebar.vue';
import AppContent from './AppContent.vue';
import PreviewPane from './PreviewPane.vue';
import StatusBar from './StatusBar.vue';
import ContextMenu from './ContextMenu.vue';
import CommandPalette from './CommandPalette.vue';
import Toasts from './Toasts.vue';
import PromptDialog from './PromptDialog.vue';
import ConfirmDialog from './ConfirmDialog.vue';
import PropertiesDialog from './PropertiesDialog.vue';
import SettingsDialog from './SettingsDialog.vue';
import ShortcutsDialog from './ShortcutsDialog.vue';
import AboutDialog from './AboutDialog.vue';
import AppPickerDialog from './AppPickerDialog.vue';
import {
  state, activeTab, updateSettings, togglePalette, openDialog, closeDialog,
  closeContextMenu, newTab, closeTab, selectTab, goBack, goForward, goUp,
  refreshTab, createFolder, createFile, clipboardCopy, clipboardCut,
  clipboardPaste, selectAll, invertSelection, clearSelection, deleteSelection,
  duplicateSelection, startRename, cancelRename, moveCursor, openEntry,
  toggleHidden, setViewMode, showProperties, openInTerminal, showTrash,
  addBookmark, saveViewDefaults, commitRename, openPathsInNewTab,
} from '../store/app.js';
import { AppService } from '../lib/fs.js';

const toolbar = ref(null);
const sidebarEl = ref(null);
const previewEl = ref(null);

const tab = computed(() => activeTab.value);

/* ------------------------------- resizing -------------------------------- */

function startResize(target, event) {
  event.preventDefault();
  const startX = event.clientX;
  const startValue = target === 'sidebar' ? state.settings.sidebarWidth : state.settings.previewWidth;

  const move = (moveEvent) => {
    const delta = moveEvent.clientX - startX;
    if (target === 'sidebar') {
      updateSettings({ sidebarWidth: Math.min(420, Math.max(170, startValue + delta)) });
    } else {
      updateSettings({ previewWidth: Math.min(560, Math.max(240, startValue - delta)) });
    }
  };
  const up = () => {
    window.removeEventListener('mousemove', move);
    window.removeEventListener('mouseup', up);
    document.body.style.cursor = '';
    persistWidths();
  };
  document.body.style.cursor = 'col-resize';
  window.addEventListener('mousemove', move);
  window.addEventListener('mouseup', up);
}

function persistWidths() {
  updateSettings({
    sidebarWidth: state.settings.sidebarWidth,
    previewWidth: state.settings.previewWidth,
  }, true);
}

/* ---------------------------- keyboard handling -------------------------- */

const EDITABLE = new Set(['INPUT', 'TEXTAREA', 'SELECT']);

function isEditable(target) {
  return EDITABLE.has(target?.tagName) || target?.isContentEditable;
}

async function onKeydown(event) {
  const current = tab.value;
  const mod = event.ctrlKey || event.metaKey;
  const target = event.target;

  // Global shortcuts that work even while typing.
  if (event.key === 'Escape') {
    closeContextMenu();
    if (state.dialogs.prompt) return; // handled by the dialog
    if (state.palette.open) { togglePalette(false); return; }
    if (isEditable(target) && target.blur) target.blur();
    return;
  }

  if (mod && event.key === ',') { event.preventDefault(); openDialog('settings'); return; }
  if (mod && event.key === '/') { event.preventDefault(); openDialog('shortcuts'); return; }
  if (mod && event.key.toLowerCase() === 'p' && !event.shiftKey) {
    event.preventDefault();
    togglePalette(true);
    return;
  }
  if (event.key === 'F12') { event.preventDefault(); AppService.OpenDevTools().catch(() => {}); return; }

  if (state.palette.open || state.dialogs.prompt || state.dialogs.confirm ||
      state.dialogs.properties || state.dialogs.settings || state.dialogs.shortcuts ||
      state.dialogs.about || state.dialogs.appPicker) {
    return;
  }

  // Tab management works from anywhere.
  if (mod && event.key.toLowerCase() === 't' && !event.shiftKey) { event.preventDefault(); newTab(); return; }
  if (mod && event.key.toLowerCase() === 'w') { event.preventDefault(); if (current) closeTab(current.id); return; }
  if (mod && event.key === 'Tab') {
    event.preventDefault();
    const index = state.tabs.findIndex((item) => item.id === state.activeTabId);
    const next = (index + (event.shiftKey ? -1 : 1) + state.tabs.length) % state.tabs.length;
    selectTab(state.tabs[next].id);
    return;
  }
  if (mod && ['1', '2', '3'].includes(event.key)) {
    event.preventDefault();
    setViewMode(current, ['list', 'grid', 'columns'][Number(event.key) - 1]);
    return;
  }

  if (isEditable(target)) return;

  if (mod && event.key.toLowerCase() === 'l') { event.preventDefault(); toolbar.value?.focusPath(); return; }
  if (mod && event.key.toLowerCase() === 'f') {
    event.preventDefault();
    document.querySelector('.search-input')?.focus();
    return;
  }
  if (mod && event.key.toLowerCase() === 'a' && event.shiftKey) { event.preventDefault(); invertSelection(current); return; }
  if (mod && event.key.toLowerCase() === 'a') { event.preventDefault(); selectAll(current); return; }
  if (mod && event.key.toLowerCase() === 'c') { event.preventDefault(); clipboardCopy(current); return; }
  if (mod && event.key.toLowerCase() === 'x') { event.preventDefault(); clipboardCut(current); return; }
  if (mod && event.key.toLowerCase() === 'v') { event.preventDefault(); clipboardPaste(current); return; }
  if (mod && event.key.toLowerCase() === 'd') { event.preventDefault(); duplicateSelection(current); return; }
  if (mod && event.key.toLowerCase() === 'h') { event.preventDefault(); toggleHidden(current); return; }
  if (mod && event.shiftKey && event.key.toLowerCase() === 'n') { event.preventDefault(); createFolder(current); return; }
  if (mod && event.altKey && event.key.toLowerCase() === 'n') { event.preventDefault(); createFile(current); return; }
  if (mod && event.shiftKey && event.key.toLowerCase() === 's') { event.preventDefault(); saveViewDefaults(current); return; }
  if (mod && event.shiftKey && event.key.toLowerCase() === 'f') {
    event.preventDefault();
    AppService.ToggleFullscreen().catch(() => {});
    return;
  }

  if (event.altKey && event.key === 'ArrowLeft') { event.preventDefault(); goBack(current); return; }
  if (event.altKey && event.key === 'ArrowRight') { event.preventDefault(); goForward(current); return; }
  if (event.altKey && event.key === 'ArrowUp') { event.preventDefault(); goUp(current); return; }
  if (event.altKey && event.key === 'Enter') {
    event.preventDefault();
    const paths = [...(current?.selected || [])];
    showProperties(paths.length ? paths : [current?.path]);
    return;
  }

  if (event.key === 'F5') { event.preventDefault(); refreshTab(current); return; }
  if (event.key === 'F2') {
    event.preventDefault();
    const entry = current?.entries.find((item) => item.path === current.cursor) ||
      current?.entries.find((item) => current.selected.has(item.path));
    if (entry) startRename(entry.path, entry.name);
    return;
  }
  if (event.key === 'F11') {
    event.preventDefault();
    updateSettings({ showPreview: !state.settings.showPreview }, true);
    return;
  }
  if (event.key === 'Delete') {
    event.preventDefault();
    deleteSelection(current, event.shiftKey);
    return;
  }
  if (event.key === 'Enter') {
    event.preventDefault();
    const entries = current?.entries.filter((item) => current.selected.has(item.path)) || [];
    if (entries.length) entries.forEach((entry) => openEntry(current, entry));
    return;
  }
  if (event.key === 'Backspace') { event.preventDefault(); goUp(current); return; }
  if (event.key === 'ArrowDown') { event.preventDefault(); moveCursor(current, 1, event.shiftKey); return; }
  if (event.key === 'ArrowUp') { event.preventDefault(); moveCursor(current, -1, event.shiftKey); return; }
  if (event.key === 'PageDown') { event.preventDefault(); moveCursor(current, 12, event.shiftKey); return; }
  if (event.key === 'PageUp') { event.preventDefault(); moveCursor(current, -12, event.shiftKey); return; }
  if (event.key === 'Home') { event.preventDefault(); moveCursor(current, -100000, event.shiftKey); return; }
  if (event.key === 'End') { event.preventDefault(); moveCursor(current, 100000, event.shiftKey); return; }
  if (event.key === ' ') {
    event.preventDefault();
    updateSettings({ showPreview: !state.settings.showPreview }, true);
    return;
  }
}

/* ------------------------------- lifecycle ------------------------------- */

function onWindowFocus() {
  AppService.Show().catch(() => {});
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown);
  window.addEventListener('focus', onWindowFocus);
});

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown);
  window.removeEventListener('focus', onWindowFocus);
});
</script>

<template>
  <div
    class="shell"
    :class="{ 'no-sidebar': !state.settings.showSidebar, 'no-preview': !state.settings.showPreview }"
    :style="{
      '--sidebar-w': `${state.settings.sidebarWidth}px`,
      '--preview-w': `${state.settings.previewWidth}px`,
    }"
  >
    <TitleBar />
    <TabBar />
    <Toolbar ref="toolbar" />

    <div class="body">
      <Sidebar v-if="state.settings.showSidebar" ref="sidebarEl" />
      <div
        v-if="state.settings.showSidebar"
        class="resizer"
        title="Drag to resize"
        @mousedown="startResize('sidebar', $event)"
      />

      <AppContent />

      <div
        v-if="state.settings.showPreview"
        class="resizer is-right"
        title="Drag to resize"
        @mousedown="startResize('preview', $event)"
      />
      <PreviewPane v-if="state.settings.showPreview" ref="previewEl" />
    </div>

    <StatusBar v-if="state.settings.showStatusBar" />

    <ContextMenu />
    <CommandPalette />
    <Toasts />
    <PromptDialog />
    <ConfirmDialog />
    <PropertiesDialog />
    <SettingsDialog />
    <ShortcutsDialog />
    <AboutDialog />
    <AppPickerDialog />
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  background: var(--bg);
}

.body {
  flex: 1 1 auto;
  display: flex;
  min-height: 0;
  overflow: hidden;
}

.resizer {
  width: 4px;
  flex: 0 0 auto;
  cursor: col-resize;
  background: transparent;
  transition: background var(--t-fast);
  position: relative;
  z-index: 6;
}
.resizer::after {
  content: "";
  position: absolute;
  inset: 0 1px;
  background: transparent;
  transition: background var(--t-fast);
}
.resizer:hover::after { background: var(--accent); }
</style>

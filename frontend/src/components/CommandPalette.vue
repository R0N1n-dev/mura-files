<script setup>
import { ref, computed, watch, nextTick } from 'vue';
import Icon from './Icon.vue';
import {
  state, togglePalette, activeTab, navigate, newTab, refreshTab, goUp,
  createFolder, createFile, clipboardPaste, clipboardCopy, clipboardCut,
  selectAll, toggleHidden, setViewMode, setSort, openDialog, showTrash,
  addBookmark, deleteSelection, duplicateSelection, saveViewDefaults,
  openInTerminal, emptyTrash, quit, openPathsInNewTab,
} from '../store/app.js';

const query = ref('');
const index = ref(0);
const input = ref(null);

const tab = computed(() => activeTab.value);

/** Static commands plus dynamic entries for folders and bookmarks. */
const commands = computed(() => {
  const list = [
    { id: 'new-tab', label: 'New Tab', hint: 'Ctrl+T', icon: 'plus', run: () => newTab() },
    { id: 'new-folder', label: 'New Folder', hint: 'Ctrl+Shift+N', icon: 'folder-plus', run: () => createFolder(tab.value) },
    { id: 'new-file', label: 'New File', hint: 'Ctrl+Alt+N', icon: 'file-plus', run: () => createFile(tab.value) },
    { id: 'refresh', label: 'Refresh Folder', hint: 'F5', icon: 'refresh', run: () => refreshTab(tab.value) },
    { id: 'up', label: 'Go to Parent Folder', hint: 'Alt+Up', icon: 'arrow-up', run: () => goUp(tab.value) },
    { id: 'paste', label: 'Paste', hint: 'Ctrl+V', icon: 'paste', run: () => clipboardPaste(tab.value) },
    { id: 'copy', label: 'Copy Selection', hint: 'Ctrl+C', icon: 'copy', run: () => clipboardCopy(tab.value) },
    { id: 'cut', label: 'Cut Selection', hint: 'Ctrl+X', icon: 'scissors', run: () => clipboardCut(tab.value) },
    { id: 'duplicate', label: 'Duplicate Selection', hint: 'Ctrl+D', icon: 'copy', run: () => duplicateSelection(tab.value) },
    { id: 'select-all', label: 'Select All', hint: 'Ctrl+A', icon: 'check', run: () => selectAll(tab.value) },
    { id: 'trash-selection', label: 'Move Selection to Trash', hint: 'Del', icon: 'trash', run: () => deleteSelection(tab.value) },
    { id: 'view-list', label: 'List View', hint: 'Ctrl+1', icon: 'layout-list', run: () => setViewMode(tab.value, 'list') },
    { id: 'view-grid', label: 'Grid View', hint: 'Ctrl+2', icon: 'layout-grid', run: () => setViewMode(tab.value, 'grid') },
    { id: 'view-columns', label: 'Compact View', hint: 'Ctrl+3', icon: 'layout-columns', run: () => setViewMode(tab.value, 'columns') },
    { id: 'sort-name', label: 'Sort by Name', icon: 'sort-asc', run: () => setSort(tab.value, 'name') },
    { id: 'sort-size', label: 'Sort by Size', icon: 'sort-asc', run: () => setSort(tab.value, 'size') },
    { id: 'sort-modified', label: 'Sort by Modified Date', icon: 'sort-asc', run: () => setSort(tab.value, 'modified') },
    { id: 'hidden', label: 'Toggle Hidden Files', hint: 'Ctrl+H', icon: 'eye', run: () => toggleHidden(tab.value) },
    { id: 'bookmark', label: 'Bookmark This Folder', icon: 'bookmark', run: () => addBookmark(tab.value?.path) },
    { id: 'terminal', label: 'Open in Terminal', icon: 'terminal', run: () => openInTerminal(tab.value?.path) },
    { id: 'trash', label: 'Open Trash', icon: 'trash', run: () => showTrash() },
    { id: 'empty-trash', label: 'Empty Trash', icon: 'trash-2', run: () => emptyTrash() },
    { id: 'save-view', label: 'Save View Settings as Default', icon: 'save', run: () => saveViewDefaults(tab.value) },
    { id: 'settings', label: 'Open Settings', hint: 'Ctrl+,', icon: 'settings', run: () => openDialog('settings') },
    { id: 'shortcuts', label: 'Keyboard Shortcuts', hint: 'Ctrl+/', icon: 'keyboard', run: () => openDialog('shortcuts') },
    { id: 'about', label: 'About Mura Files', icon: 'info', run: () => openDialog('about') },
    { id: 'quit', label: 'Quit Mura Files', icon: 'x-circle', run: () => quit() },
  ];

  state.places.forEach((place) => {
    if (!place.present) return;
    list.push({
      id: `place-${place.path}`,
      label: `Go to ${place.name}`,
      hint: place.path,
      icon: 'folder',
      run: () => navigate(tab.value, place.path),
    });
  });

  state.bookmarks.forEach((bookmark) => {
    list.push({
      id: `bookmark-${bookmark.path}`,
      label: `Bookmark: ${bookmark.name}`,
      hint: bookmark.path,
      icon: 'bookmark',
      run: () => navigate(tab.value, bookmark.path),
    });
  });

  state.drives.forEach((drive) => {
    list.push({
      id: `drive-${drive.path}`,
      label: `Open ${drive.name}`,
      hint: drive.path,
      icon: 'hard-drive',
      run: () => navigate(tab.value, drive.path),
    });
  });

  return list;
});

/** A trailing path-looking query offers to open it directly. */
const results = computed(() => {
  const needle = query.value.trim().toLowerCase();
  const base = commands.value;
  if (!needle) return base.slice(0, 12);

  const scored = [];
  base.forEach((command) => {
    const haystack = `${command.label} ${command.hint || ''}`.toLowerCase();
    const position = haystack.indexOf(needle);
    if (position >= 0) scored.push({ command, score: position });
  });
  scored.sort((a, b) => a.score - b.score);

  const list = scored.slice(0, 12).map((item) => item.command);
  if (needle.startsWith('/') || needle.startsWith('~')) {
    list.unshift({
      id: 'open-path',
      label: `Open “${query.value.trim()}”`,
      hint: 'path',
      icon: 'external-link',
      run: () => openPathsInNewTab([query.value.trim()]),
    });
  }
  return list;
});

watch(results, () => { index.value = 0; });

watch(() => state.palette.open, async (open) => {
  if (!open) return;
  query.value = '';
  index.value = 0;
  await nextTick();
  input.value?.focus();
});

function run(item) {
  if (!item) return;
  togglePalette(false);
  item.run?.();
}

function onKeydown(event) {
  if (event.key === 'ArrowDown') {
    event.preventDefault();
    index.value = Math.min(index.value + 1, results.value.length - 1);
  } else if (event.key === 'ArrowUp') {
    event.preventDefault();
    index.value = Math.max(index.value - 1, 0);
  } else if (event.key === 'Enter') {
    event.preventDefault();
    run(results.value[index.value]);
  } else if (event.key === 'Escape') {
    event.preventDefault();
    togglePalette(false);
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="state.palette.open" class="palette-backdrop" @mousedown.self="togglePalette(false)">
      <div class="palette panel">
        <div class="palette-input">
          <Icon name="search" :size="16" />
          <input
            ref="input"
            v-model="query"
            placeholder="Type a command, folder or path…"
            spellcheck="false"
            autocomplete="off"
            @keydown="onKeydown"
          />
          <kbd class="kbd">Esc</kbd>
        </div>

        <div v-if="results.length" class="palette-list">
          <button
            v-for="(item, position) in results"
            :key="item.id"
            class="palette-item"
            :class="{ 'is-active': position === index }"
            type="button"
            @mouseenter="index = position"
            @click="run(item)"
          >
            <Icon :name="item.icon || 'zap'" :size="15" class="palette-icon" />
            <span class="truncate">{{ item.label }}</span>
            <span v-if="item.hint" class="palette-hint faint truncate">{{ item.hint }}</span>
          </button>
        </div>
        <div v-else class="palette-empty faint">No matching commands.</div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.palette-backdrop {
  position: fixed;
  inset: 0;
  z-index: 550;
  display: flex;
  justify-content: center;
  padding-top: 14vh;
  background: var(--overlay);
  backdrop-filter: blur(2px);
}

.palette {
  width: min(620px, calc(100vw - 2 * var(--s-6)));
  max-height: 62vh;
  display: flex;
  flex-direction: column;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-lg);
  box-shadow: var(--shadow-3);
  overflow: hidden;
  animation: palette-in var(--t-base) cubic-bezier(0.32, 0.72, 0, 1);
}
@keyframes palette-in {
  from { opacity: 0; transform: translateY(-8px) scale(0.985); }
  to { opacity: 1; transform: none; }
}

.palette-input {
  display: flex;
  align-items: center;
  gap: var(--s-3);
  padding: 0 var(--s-4);
  height: 50px;
  border-bottom: 1px solid var(--border);
  color: var(--text-faint);
}
.palette-input input {
  flex: 1 1 auto;
  height: 100%;
  border: 0;
  background: none;
  color: var(--text);
  font-size: var(--fs-lg);
  user-select: text;
}
.palette-input input:focus { outline: none; }
.palette-input input::placeholder { color: var(--text-faint); }

.palette-list { overflow-y: auto; padding: 5px; }
.palette-item {
  display: flex;
  align-items: center;
  gap: var(--s-3);
  width: 100%;
  height: 36px;
  padding: 0 var(--s-3);
  border-radius: var(--r-sm);
  color: var(--text);
  font-size: var(--fs-md);
  text-align: left;
}
.palette-item.is-active { background: var(--accent-soft); color: var(--accent); }
.palette-icon { flex: 0 0 auto; opacity: 0.8; }
.palette-hint { margin-left: auto; font-size: var(--fs-xs); max-width: 220px; }
.palette-empty { padding: var(--s-5); text-align: center; font-size: var(--fs-sm); }
</style>

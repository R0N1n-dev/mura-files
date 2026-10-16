<script setup>
import { computed, ref, nextTick, watch } from 'vue';
import Icon from './Icon.vue';
import {
  state, activeTab, selectEntry, openEntry, startRename, commitRename, cancelRename,
  openContextMenu, buildEntryMenu, dropInto, setSelection,
} from '../store/app.js';
import { formatSize, formatDate, splitName } from '../lib/format.js';
import { thumbUrl, iconUrl } from '../lib/fs.js';

const props = defineProps({
  entries: { type: Array, default: null },
  size: { type: Number, default: 0 },
});

const gridEl = ref(null);
const renameValue = ref('');
const dragOverPath = ref('');

const tab = computed(() => activeTab.value);
const rows = computed(() => props.entries || tab.value?.entries || []);
const cellSize = computed(() => props.size || state.settings.gridSize || 112);

function onCellClick(event, entry) {
  selectEntry(tab.value, entry, { ctrl: event.ctrlKey, shift: event.shiftKey, meta: event.metaKey });
  if (state.settings.singleClickOpen && !event.ctrlKey && !event.shiftKey && !event.metaKey) {
    openEntry(tab.value, entry);
  }
}

function onCellDoubleClick(entry) {
  if (state.settings.singleClickOpen) return;
  openEntry(tab.value, entry);
}

function onCellMenu(event, entry) {
  event.preventDefault();
  if (!tab.value.selected.has(entry.path)) selectEntry(tab.value, entry, {});
  openContextMenu(event, buildEntryMenu(entry));
}

function onCellDragStart(event, entry) {
  const selected = tab.value.selected.has(entry.path) ? [...tab.value.selected] : [entry.path];
  state.drag = { active: true, paths: selected, over: '', external: false };
  event.dataTransfer.effectAllowed = 'copyMove';
  event.dataTransfer.setData('application/x-mura-paths', selected.join('\n'));
  event.dataTransfer.setData('text/plain', selected.join('\n'));
}

function onCellDragOver(event, entry) {
  if (!entry.isDir) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = 'move';
  dragOverPath.value = entry.path;
}

function onCellDrop(event, entry) {
  event.preventDefault();
  dragOverPath.value = '';
  if (!entry.isDir) return;
  const raw = event.dataTransfer.getData('application/x-mura-paths') ||
    event.dataTransfer.getData('text/plain');
  const paths = raw.split('\n').map((value) => value.trim()).filter(Boolean);
  if (paths.length) dropInto(entry.path, paths);
  state.drag = { active: false, paths: [], over: '', external: false };
}

function onBackgroundMouseDown(event) {
  if (event.button !== 0) return;
  if (event.target.closest('.tile')) return;
  if (!event.ctrlKey && !event.metaKey) tab.value.selected.clear();
}

const renamingPath = computed(() => state.renaming.path);

watch(() => state.renaming.path, async (path) => {
  renameValue.value = state.renaming.value;
  if (!path) return;
  await nextTick();
  const input = gridEl.value?.querySelector('.tile .rename-input');
  if (input) {
    input.focus();
    const dot = renameValue.value.lastIndexOf('.');
    if (dot > 0) input.setSelectionRange(0, dot);
    else input.select();
  }
});

function onRenameKeydown(event) {
  if (event.key === 'Enter') {
    event.preventDefault();
    commitRename(renameValue.value.trim());
  } else if (event.key === 'Escape') {
    event.preventDefault();
    cancelRename();
  }
}

function tileIcon(entry) {
  if (entry.isDir) return entry.isSymlink ? 'folder-symlink' : 'folder';
  return entry.icon || 'file';
}

function showThumb(entry) {
  return state.settings.showThumbnails && entry.thumb && !entry.isDir;
}

function thumbPixels() {
  return Math.round(cellSize.value * 0.62 * (window.devicePixelRatio || 1));
}

function displayName(entry) {
  return entry.desktopName || entry.name;
}

function nameParts(entry) {
  if (!state.settings.showExtensions || entry.isDir || entry.desktopName) {
    return { stem: displayName(entry), ext: '' };
  }
  return splitName(displayName(entry));
}

function metaLabel(entry) {
  if (entry.isDir) return formatDate(entry.modTime, 'auto');
  return formatSize(entry.size, { base: state.settings.sizeFormat === 'decimal' ? 'decimal' : 'binary' });
}
</script>

<template>
  <div
    ref="gridEl"
    class="grid"
    :style="{ '--cell': `${cellSize}px` }"
    @mousedown="onBackgroundMouseDown"
  >
    <button
      v-for="entry in rows"
      :key="entry.path"
      class="tile"
      :class="{
        'is-selected': tab?.selected.has(entry.path),
        'is-cursor': tab?.cursor === entry.path,
        'is-dropping': dragOverPath === entry.path,
        'is-cut': state.clipboard.mode === 'cut' && state.clipboard.paths.includes(entry.path),
      }"
      type="button"
      :data-path="entry.path"
      :data-drop-path="entry.isDir ? entry.path : ''"
      draggable="true"
      :title="entry.path"
      @click="onCellClick($event, entry)"
      @dblclick="onCellDoubleClick(entry)"
      @contextmenu="onCellMenu($event, entry)"
      @dragstart="onCellDragStart($event, entry)"
      @dragend="state.drag.active = false"
      @dragover="onCellDragOver($event, entry)"
      @dragleave="dragOverPath = ''"
      @drop="onCellDrop($event, entry)"
    >
      <span class="tile-art" :class="{ 'is-photo': showThumb(entry) || entry.appIcon }">
        <img
          v-if="showThumb(entry)"
          :src="thumbUrl(entry.path, thumbPixels())"
          :alt="entry.name"
          loading="lazy"
          decoding="async"
          draggable="false"
        />
        <img
          v-else-if="entry.appIcon"
          :src="iconUrl(entry.appIcon)"
          :alt="entry.name"
          loading="lazy"
          draggable="false"
        />
        <Icon v-else :name="tileIcon(entry)" :size="Math.round(cellSize * 0.36)" :stroke="1.4" />
        <span v-if="entry.isSymlink" class="tile-badge">
          <Icon name="link" :size="10" :stroke="2.2" />
        </span>
        <span v-if="entry.linkBroken" class="tile-badge is-danger">
          <Icon name="alert-triangle" :size="10" :stroke="2.2" />
        </span>
      </span>

      <input
        v-if="renamingPath === entry.path"
        v-model="renameValue"
        class="input rename-input"
        spellcheck="false"
        @keydown="onRenameKeydown"
        @blur="commitRename(renameValue.trim())"
        @click.stop
      />
      <span v-else class="tile-name" :class="{ 'is-broken': entry.linkBroken }">
        <span class="tile-stem">{{ nameParts(entry).stem }}</span>
        <span class="tile-ext">{{ nameParts(entry).ext }}</span>
      </span>
      <span class="tile-meta faint">{{ metaLabel(entry) }}</span>
    </button>
  </div>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(var(--cell), 1fr));
  gap: var(--s-2);
  align-content: start;
  height: 100%;
  overflow: auto;
  padding: var(--s-3) var(--s-3) var(--s-6);
}

.tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  padding: 10px 6px 9px;
  border-radius: var(--r-md);
  color: var(--text);
  transition: background var(--t-fast), box-shadow var(--t-fast), transform var(--t-fast);
  content-visibility: auto;
  contain-intrinsic-size: auto var(--cell);
  min-width: 0;
}
.tile:hover { background: var(--hover); }
.tile:active { transform: scale(0.985); }
.tile.is-selected {
  background: var(--selection);
  box-shadow: inset 0 0 0 1px var(--accent);
}
.tile.is-cursor { box-shadow: inset 0 0 0 1px var(--border-strong); }
.tile.is-selected.is-cursor { box-shadow: inset 0 0 0 1px var(--accent); }
.tile.is-cut { opacity: 0.5; }
.tile.is-dropping {
  background: var(--accent-soft);
  box-shadow: inset 0 0 0 2px var(--accent);
}

.tile-art {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: calc(var(--cell) * 0.6);
  color: var(--text-muted);
  border-radius: var(--r-sm);
  overflow: hidden;
}
.tile.is-selected .tile-art { color: var(--accent); }
.tile-art img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: var(--r-sm);
  box-shadow: var(--shadow-1);
}
.tile-art.is-photo img { background: var(--surface-2); }

.tile-badge {
  position: absolute;
  right: 2px;
  bottom: 2px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 5px;
  background: var(--surface-3);
  color: var(--text-muted);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}
.tile-badge.is-danger { background: var(--danger); color: #fff; }

.tile-name {
  width: 100%;
  font-size: var(--fs-sm);
  line-height: 1.3;
  text-align: center;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.tile-name.is-broken { color: var(--danger); }
.tile-ext { color: var(--text-faint); }
.tile-meta { font-size: var(--fs-xs); }

.rename-input {
  height: 24px;
  padding: 0 6px;
  font-size: var(--fs-sm);
  text-align: center;
}
</style>

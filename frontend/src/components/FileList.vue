<script setup>
import { computed, ref, nextTick, watch } from 'vue';
import Icon from './Icon.vue';
import {
  state, activeTab, selectEntry, openEntry, setSort, startRename, commitRename,
  cancelRename, openContextMenu, dropInto, buildEntryMenu, showProperties,
} from '../store/app.js';
import { formatSize, formatDate, describeKind, splitName } from '../lib/format.js';
import { thumbUrl, iconUrl } from '../lib/fs.js';

const props = defineProps({
  /** Optional explicit entries; defaults to the active tab. */
  entries: { type: Array, default: null },
  /** Column set to render. */
  columns: { type: Array, default: () => ['name', 'size', 'kind', 'modified'] },
});

const listEl = ref(null);
const renameInput = ref(null);

const tab = computed(() => activeTab.value);
const rows = computed(() => props.entries || tab.value?.entries || []);

const allSelected = computed(() =>
  rows.value.length > 0 && rows.value.every((entry) => tab.value?.selected.has(entry.path)));

function onRowClick(event, entry) {
  selectEntry(tab.value, entry, { ctrl: event.ctrlKey, shift: event.shiftKey, meta: event.metaKey });
  if (state.settings.singleClickOpen && !event.ctrlKey && !event.shiftKey && !event.metaKey) {
    openEntry(tab.value, entry);
  }
}

function onRowDoubleClick(entry) {
  if (state.settings.singleClickOpen) return;
  openEntry(tab.value, entry);
}

function onRowMenu(event, entry) {
  event.preventDefault();
  if (!tab.value.selected.has(entry.path)) {
    selectEntry(tab.value, entry, {});
  }
  openContextMenu(event, buildEntryMenu(entry));
}

function onHeaderMenu(event, column) {
  event.preventDefault();
  openContextMenu(event, [
    { label: 'Sort ascending', icon: 'sort-asc', run: () => setSort(tab.value, column) },
    { label: 'Sort descending', icon: 'sort-desc', run: () => setSort(tab.value, column, true) },
  ]);
}

function toggleAll() {
  if (allSelected.value) tab.value.selected.clear();
  else rows.value.forEach((entry) => tab.value.selected.add(entry.path));
}

/* --------------------------- inline renaming ----------------------------- */

const renamingPath = computed(() => state.renaming.path);
const renameValue = ref('');

watch(() => state.renaming.path, async (path) => {
  renameValue.value = state.renaming.value;
  if (!path) return;
  await nextTick();
  const input = document.querySelector('.rename-input');
  if (input) {
    input.focus();
    const dot = state.settings.showExtensions ? renameValue.value.lastIndexOf('.') : -1;
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

/* ------------------------------ drag and drop ---------------------------- */

const dragOverPath = ref('');

function onRowDragStart(event, entry) {
  const selected = tab.value.selected.has(entry.path)
    ? [...tab.value.selected]
    : [entry.path];
  state.drag = { active: true, paths: selected, over: '', external: false };
  event.dataTransfer.effectAllowed = 'copyMove';
  event.dataTransfer.setData('text/plain', selected.join('\n'));
  event.dataTransfer.setData('application/x-mura-paths', selected.join('\n'));
}

function onRowDragEnd() {
  state.drag = { active: false, paths: [], over: '', external: false };
  dragOverPath.value = '';
}

function onRowDragOver(event, entry) {
  if (!entry.isDir || !state.drag.active) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = 'move';
  dragOverPath.value = entry.path;
}

function onRowDragLeave(entry) {
  if (dragOverPath.value === entry.path) dragOverPath.value = '';
}

function onRowDrop(event, entry) {
  event.preventDefault();
  dragOverPath.value = '';
  if (!entry.isDir) return;
  const raw = event.dataTransfer.getData('application/x-mura-paths') ||
    event.dataTransfer.getData('text/plain');
  const paths = raw.split('\n').map((value) => value.trim()).filter(Boolean);
  if (!paths.length) return;
  dropInto(entry.path, paths);
  state.drag = { active: false, paths: [], over: '', external: false };
}

/* ------------------------------ marquee select --------------------------- */

const marquee = ref(null);

function onBackgroundMouseDown(event) {
  if (event.button !== 0) return;
  if (event.target.closest('.row, .list-head')) return;
  const container = listEl.value;
  if (!container) return;

  const bounds = container.getBoundingClientRect();
  const startX = event.clientX - bounds.left + container.scrollLeft;
  const startY = event.clientY - bounds.top + container.scrollTop;
  const additive = event.ctrlKey || event.metaKey;
  if (!additive) tab.value.selected.clear();

  const move = (moveEvent) => {
    const x = moveEvent.clientX - bounds.left + container.scrollLeft;
    const y = moveEvent.clientY - bounds.top + container.scrollTop;
    marquee.value = {
      left: Math.min(startX, x),
      top: Math.min(startY, y),
      width: Math.abs(x - startX),
      height: Math.abs(y - startY),
    };
  };

  const up = () => {
    window.removeEventListener('mousemove', move);
    window.removeEventListener('mouseup', up);
    const box = marquee.value;
    marquee.value = null;
    if (!box || box.width < 4 || box.height < 4) return;
    container.querySelectorAll('.row').forEach((element) => {
      const rect = element.getBoundingClientRect();
      const top = rect.top - bounds.top + container.scrollTop;
      const left = rect.left - bounds.left + container.scrollLeft;
      const overlaps = left < box.left + box.width && left + rect.width > box.left &&
        top < box.top + box.height && top + rect.height > box.top;
      if (overlaps) tab.value.selected.add(element.dataset.path);
    });
  };

  window.addEventListener('mousemove', move);
  window.addEventListener('mouseup', up);
}

/* ------------------------------- rendering ------------------------------- */

function rowIcon(entry) {
  if (entry.isDir) return entry.isSymlink ? 'folder-symlink' : 'folder';
  return entry.icon || 'file';
}

function showThumb(entry) {
  return state.settings.showThumbnails && entry.thumb && !entry.isDir;
}

function thumbSize() {
  return 2 * (window.devicePixelRatio || 1) * 24;
}

function displayName(entry) {
  if (entry.desktopName) return entry.desktopName;
  return entry.name;
}

function nameParts(entry) {
  const name = displayName(entry);
  if (!state.settings.showExtensions || entry.isDir || entry.desktopName) {
    return { stem: name, ext: '' };
  }
  return splitName(name);
}

function sizeLabel(entry) {
  if (entry.isDir) return '--';
  if (entry.isSymlink) return 'link';
  return formatSize(entry.size, { base: state.settings.sizeFormat === 'decimal' ? 'decimal' : 'binary' });
}

function dateLabel(entry) {
  return formatDate(entry.modTime, state.settings.dateFormat || 'auto');
}

const COLUMN_LABELS = {
  name: 'Name', size: 'Size', kind: 'Type', modified: 'Modified',
  created: 'Created', perm: 'Permissions', owner: 'Owner',
};

defineExpose({ listEl });
</script>

<template>
  <div
    ref="listEl"
    class="list"
    :class="`cols-${columns.length}`"
    role="grid"
    @mousedown="onBackgroundMouseDown"
  >
    <div class="list-head" role="row">
      <div class="cell cell-check">
        <button
          class="check"
          :class="{ 'is-on': allSelected }"
          type="button"
          :aria-label="allSelected ? 'Deselect all' : 'Select all'"
          @click="toggleAll"
        >
          <Icon name="check" :size="11" :stroke="2.6" />
        </button>
      </div>
      <button
        v-for="column in columns"
        :key="column"
        class="cell head-cell"
        :class="`cell-${column}`"
        type="button"
        @click="setSort(tab, column)"
        @contextmenu="onHeaderMenu($event, column)"
      >
        <span>{{ COLUMN_LABELS[column] || column }}</span>
        <Icon
          v-if="tab?.sortBy === column"
          :name="tab.sortDesc ? 'chevron-down' : 'chevron-up'"
          :size="12"
          class="sort-arrow"
        />
      </button>
    </div>

    <div class="list-body" role="rowgroup">
      <div
        v-for="entry in rows"
        :key="entry.path"
        class="row"
        :class="{
          'is-selected': tab?.selected.has(entry.path),
          'is-cursor': tab?.cursor === entry.path,
          'is-hidden': entry.isHidden,
          'is-dropping': dragOverPath === entry.path,
          'is-cut': state.clipboard.mode === 'cut' && state.clipboard.paths.includes(entry.path),
        }"
        role="row"
        :data-path="entry.path"
        :data-drop-path="entry.isDir ? entry.path : ''"
        draggable="true"
        :title="entry.path"
        @click="onRowClick($event, entry)"
        @dblclick="onRowDoubleClick(entry)"
        @contextmenu="onRowMenu($event, entry)"
        @dragstart="onRowDragStart($event, entry)"
        @dragend="onRowDragEnd"
        @dragover="onRowDragOver($event, entry)"
        @dragleave="onRowDragLeave(entry)"
        @drop="onRowDrop($event, entry)"
      >
        <div class="cell cell-check">
          <button
            class="check"
            :class="{ 'is-on': tab?.selected.has(entry.path) }"
            type="button"
            :aria-label="`Select ${entry.name}`"
            @click.stop="selectEntry(tab, entry, { ctrl: true })"
          >
            <Icon name="check" :size="11" :stroke="2.6" />
          </button>
        </div>

        <div class="cell cell-name">
          <span class="thumb" :class="{ 'is-square': !entry.isDir }">
            <img
              v-if="showThumb(entry)"
              :src="thumbUrl(entry.path, thumbSize())"
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
            <Icon v-else :name="rowIcon(entry)" :size="18" />
            <span v-if="entry.isSymlink" class="link-badge" title="Symbolic link">
              <Icon name="link" :size="9" :stroke="2.2" />
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
          <span v-else class="name-text truncate" :class="{ 'is-broken': entry.linkBroken }">
            <span>{{ nameParts(entry).stem }}</span><span class="ext">{{ nameParts(entry).ext }}</span>
          </span>

          <span v-if="entry.linkBroken" class="chip danger">broken</span>
          <span v-else-if="entry.isHidden" class="chip">hidden</span>
        </div>

        <div v-if="columns.includes('size')" class="cell cell-size mono tnum">{{ sizeLabel(entry) }}</div>
        <div v-if="columns.includes('kind')" class="cell cell-kind truncate muted">{{ describeKind(entry) }}</div>
        <div v-if="columns.includes('modified')" class="cell cell-modified muted">{{ dateLabel(entry) }}</div>
        <div v-if="columns.includes('created')" class="cell cell-created muted">{{ formatDate(entry.changeTime, 'auto') }}</div>
        <div v-if="columns.includes('perm')" class="cell cell-perm mono faint">{{ entry.perm }}</div>
        <div v-if="columns.includes('owner')" class="cell cell-owner truncate muted">{{ entry.owner }}</div>
      </div>
    </div>

    <div
      v-if="marquee"
      class="marquee"
      :style="{
        left: `${marquee.left}px`,
        top: `${marquee.top}px`,
        width: `${marquee.width}px`,
        height: `${marquee.height}px`,
      }"
    />
  </div>
</template>

<style scoped>
.list {
  position: relative;
  height: 100%;
  overflow: auto;
  padding-bottom: var(--s-6);
}

.list-head,
.row {
  display: grid;
  align-items: center;
  gap: 0;
  grid-template-columns: 30px minmax(180px, 3fr) 92px 132px 148px;
  padding: 0 var(--s-3);
}
.cols-3 { grid-template-columns: 30px minmax(180px, 3fr) 100px 148px !important; }
.cols-4 { grid-template-columns: 30px minmax(180px, 3fr) 92px 132px 148px !important; }
.cols-5 { grid-template-columns: 30px minmax(180px, 3fr) 92px 132px 148px 148px !important; }

.list-head {
  position: sticky;
  top: 0;
  z-index: 5;
  height: 30px;
  background: var(--bg);
  border-bottom: 1px solid var(--border);
  color: var(--text-faint);
  font-size: var(--fs-xs);
  font-weight: 600;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}

.head-cell {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 100%;
  color: inherit;
  font: inherit;
  letter-spacing: inherit;
  text-transform: inherit;
  text-align: left;
  transition: color var(--t-fast);
}
.head-cell:hover { color: var(--text); }
.head-cell.cell-size { justify-content: flex-end; }
.sort-arrow { color: var(--accent); }

.row {
  height: var(--row-h);
  border-radius: var(--r-sm);
  color: var(--text);
  /* Contain paint work for off-screen rows so huge folders stay smooth. */
  content-visibility: auto;
  contain-intrinsic-size: auto var(--row-h);
  transition: background var(--t-fast);
}
.row:hover { background: var(--hover); }
.row.is-selected { background: var(--selection); }
.row.is-selected:hover { background: var(--selection-hover); }
.row.is-cursor { box-shadow: inset 0 0 0 1px var(--border-strong); }
.row.is-hidden .name-text { opacity: 0.62; }
.row.is-cut .name-text { opacity: 0.5; }
.row.is-dropping {
  background: var(--accent-soft);
  box-shadow: inset 0 0 0 1.5px var(--accent);
}

.cell { display: flex; align-items: center; min-width: 0; padding-right: var(--s-3); }
.cell-check { justify-content: center; padding-right: 0; }
.cell-size { justify-content: flex-end; }
.cell-perm { font-size: var(--fs-xs); }
.cell-modified, .cell-created, .cell-kind, .cell-owner { font-size: var(--fs-sm); }

.cell-name { gap: 9px; }

.thumb {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  color: var(--text-muted);
  position: relative;
}
.thumb img {
  width: 20px;
  height: 20px;
  object-fit: cover;
  border-radius: 3px;
}
.thumb.is-square img { border-radius: 4px; }
.thumb.is-square { width: 20px; height: 20px; }

.link-badge {
  position: absolute;
  right: -4px;
  bottom: -3px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 12px;
  height: 12px;
  border-radius: 3px;
  background: var(--surface-3);
  color: var(--text-muted);
  box-shadow: 0 0 0 1.5px var(--bg);
}
.row.is-selected .link-badge { box-shadow: 0 0 0 1.5px transparent; }

.name-text { flex: 1 1 auto; min-width: 0; font-size: var(--fs-md); }
.name-text .ext { color: var(--text-faint); }
.name-text.is-broken { color: var(--danger); text-decoration: line-through; }

.rename-input {
  height: 24px;
  padding: 0 7px;
  font-size: var(--fs-md);
}

.marquee {
  position: absolute;
  z-index: 3;
  background: var(--accent-soft);
  border: 1px solid var(--accent);
  border-radius: 3px;
  pointer-events: none;
}
</style>

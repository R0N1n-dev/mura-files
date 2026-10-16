<script setup>
import { computed, ref } from 'vue';
import Icon from './Icon.vue';
import {
  state, activeTab, loadTrash, restoreTrashItems, deleteTrashItems, emptyTrash,
  openContextMenu, copyPathToClipboard, openInTerminal, showProperties, toast,
} from '../store/app.js';
import { formatSize, formatDate, pluralise } from '../lib/format.js';
import { AppService, OperationService } from '../lib/fs.js';

const tab = computed(() => activeTab.value);
const selected = ref(new Set());
const sortBy = ref('deleted');
const sortDesc = ref(true);
const filter = ref('');

const rows = computed(() => {
  const needle = filter.value.trim().toLowerCase();
  let list = state.trash.filter((entry) =>
    !needle || entry.name.toLowerCase().includes(needle) || entry.path.toLowerCase().includes(needle));
  list = list.slice().sort((a, b) => {
    let result = 0;
    switch (sortBy.value) {
      case 'name': result = a.name.localeCompare(b.name, undefined, { numeric: true }); break;
      case 'size': result = (a.size || 0) - (b.size || 0); break;
      case 'origin': result = a.path.localeCompare(b.path); break;
      default: result = (a.deletedAt || 0) - (b.deletedAt || 0);
    }
    return sortDesc.value ? -result : result;
  });
  return list;
});

const totalSize = computed(() => state.trash.reduce((sum, entry) => sum + (entry.size || 0), 0));

function toggle(entry, event) {
  if (event.ctrlKey || event.metaKey) {
    if (selected.value.has(entry.info)) selected.value.delete(entry.info);
    else selected.value.add(entry.info);
    return;
  }
  selected.value.clear();
  selected.value.add(entry.info);
}

function selectAll() {
  if (selected.value.size === rows.value.length) selected.value.clear();
  else rows.value.forEach((entry) => selected.value.add(entry.info));
}

function selectedInfos() {
  return [...selected.value];
}

function entryMenu(event, entry) {
  event.preventDefault();
  if (!selected.value.has(entry.info)) {
    selected.value.clear();
    selected.value.add(entry.info);
  }
  const infos = selectedInfos();
  openContextMenu(event, [
    { label: 'Restore', icon: 'restore', run: () => restore(infos) },
    { label: 'Restore To…', icon: 'folder-plus', disabled: infos.length !== 1, run: () => restoreTo(entry) },
    { separator: true },
    { label: 'Copy Original Path', icon: 'link', run: () => copyPathToClipboard(entry.path) },
    { label: 'Open Original Location', icon: 'folder', run: () => openInTerminal(entry.path.replace(/\/[^/]+$/, '')) },
    { separator: true },
    { label: 'Properties', icon: 'info', run: () => showProperties([entry.trashed]) },
    { separator: true },
    { label: 'Delete Permanently', icon: 'trash-2', danger: true, run: () => remove(infos) },
  ]);
}

async function restore(infos) {
  await restoreTrashItems(infos);
  selected.value.clear();
}

async function remove(infos) {
  await deleteTrashItems(infos, true);
  selected.value.clear();
}

async function restoreTo(entry) {
  const destination = await AppService.ChooseFolder('Restore to folder', entry.path.replace(/\/[^/]+$/, ''));
  if (!destination) return;
  try {
    const result = await AppService.RestoreTrash(entry.info);
    if (result) {
      // Move the restored item into the chosen folder.
      await OperationService.Move([result], destination, null);
      toast.success('Restored');
    }
  } catch (error) {
    toast.error(String(error?.message || error));
  }
  await loadTrash();
  selected.value.clear();
}

function sort(key) {
  if (sortBy.value === key) sortDesc.value = !sortDesc.value;
  else { sortBy.value = key; sortDesc.value = false; }
}
</script>

<template>
  <div class="trash-view">
    <header class="trash-head">
      <div class="grow">
        <h2 class="trash-title">Trash</h2>
        <p class="trash-sub muted">
          {{ pluralise(state.trash.length, 'item') }} · {{ formatSize(totalSize) }}
        </p>
      </div>
      <input v-model="filter" class="input" placeholder="Filter…" style="width: 180px; height: 30px" spellcheck="false" />
      <button class="btn btn-sm btn-outline" type="button" @click="loadTrash">
        <Icon name="refresh" :size="14" /> Refresh
      </button>
      <button
        class="btn btn-sm btn-danger-ghost"
        type="button"
        :disabled="!state.trash.length"
        @click="emptyTrash"
      >
        <Icon name="trash-2" :size="14" /> Empty Trash
      </button>
    </header>

    <div v-if="selected.size" class="trash-actions">
      <span class="chip accent">{{ selected.size }} selected</span>
      <button class="btn btn-sm btn-outline" type="button" @click="restore(selectedInfos())">
        <Icon name="restore" :size="14" /> Restore
      </button>
      <button class="btn btn-sm btn-danger-ghost" type="button" @click="remove(selectedInfos())">
        <Icon name="trash-2" :size="14" /> Delete permanently
      </button>
      <button class="btn btn-sm btn-ghost" type="button" @click="selected.clear()">Clear</button>
    </div>

    <div v-if="!state.trash.length && !state.trashLoading" class="empty">
      <Icon name="trash" :size="36" class="empty-icon" />
      <span class="empty-title">Trash is empty</span>
      <span class="empty-text">Deleted files land here so you can change your mind.</span>
    </div>

    <div v-else class="trash-list">
      <div class="trash-row is-head">
        <button class="cell cell-check" type="button" @click="selectAll">
          <span class="check" :class="{ 'is-on': selected.size > 0 && selected.size === rows.length }">
            <Icon name="check" :size="11" :stroke="2.6" />
          </span>
        </button>
        <button class="cell cell-name head" type="button" @click="sort('name')">
          Name <Icon v-if="sortBy === 'name'" :name="sortDesc ? 'chevron-down' : 'chevron-up'" :size="11" />
        </button>
        <button class="cell cell-origin head" type="button" @click="sort('origin')">
          Original location <Icon v-if="sortBy === 'origin'" :name="sortDesc ? 'chevron-down' : 'chevron-up'" :size="11" />
        </button>
        <button class="cell cell-size head" type="button" @click="sort('size')">
          Size <Icon v-if="sortBy === 'size'" :name="sortDesc ? 'chevron-down' : 'chevron-up'" :size="11" />
        </button>
        <button class="cell cell-date head" type="button" @click="sort('deleted')">
          Deleted <Icon v-if="sortBy === 'deleted'" :name="sortDesc ? 'chevron-down' : 'chevron-up'" :size="11" />
        </button>
        <span class="cell cell-tools" />
      </div>

      <div
        v-for="entry in rows"
        :key="entry.info"
        class="trash-row"
        :class="{ 'is-selected': selected.has(entry.info) }"
        @click="toggle(entry, $event)"
        @dblclick="restore([entry.info])"
        @contextmenu="entryMenu($event, entry)"
      >
        <span class="cell cell-check">
          <span class="check" :class="{ 'is-on': selected.has(entry.info) }">
            <Icon name="check" :size="11" :stroke="2.6" />
          </span>
        </span>
        <span class="cell cell-name">
          <Icon :name="entry.isDir ? 'folder' : 'file'" :size="16" />
          <span class="truncate">{{ entry.name }}</span>
        </span>
        <span class="cell cell-origin truncate muted" :title="entry.path">{{ entry.path }}</span>
        <span class="cell cell-size mono tnum faint">{{ formatSize(entry.size) }}</span>
        <span class="cell cell-date muted">{{ formatDate(entry.deletedAt, 'auto') }}</span>
        <span class="cell cell-tools">
          <button class="icon-btn is-sm" type="button" title="Restore" @click.stop="restore([entry.info])">
            <Icon name="restore" :size="14" />
          </button>
          <button class="icon-btn is-sm" type="button" title="Delete permanently" @click.stop="remove([entry.info])">
            <Icon name="trash-2" :size="14" />
          </button>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.trash-view { display: flex; flex-direction: column; height: 100%; overflow: hidden; }

.trash-head {
  display: flex;
  align-items: center;
  gap: var(--s-2);
  padding: var(--s-3) var(--s-4);
  border-bottom: 1px solid var(--border);
  flex: 0 0 auto;
}
.trash-title { font-size: var(--fs-xl); }
.trash-sub { font-size: var(--fs-xs); }

.trash-actions {
  display: flex;
  align-items: center;
  gap: var(--s-2);
  padding: var(--s-2) var(--s-4);
  background: var(--accent-softer);
  border-bottom: 1px solid var(--border);
}

.trash-list { flex: 1 1 auto; overflow: auto; padding-bottom: var(--s-6); }

.trash-row {
  display: grid;
  grid-template-columns: 34px minmax(160px, 2fr) minmax(160px, 2fr) 92px 130px 76px;
  align-items: center;
  height: 34px;
  padding: 0 var(--s-3);
  border-radius: var(--r-sm);
  transition: background var(--t-fast);
  content-visibility: auto;
  contain-intrinsic-size: auto 34px;
}
.trash-row:hover { background: var(--hover); }
.trash-row.is-selected { background: var(--selection); }
.trash-row.is-head {
  height: 28px;
  color: var(--text-faint);
  font-size: var(--fs-xs);
  font-weight: 600;
  letter-spacing: 0.03em;
  text-transform: uppercase;
  position: sticky;
  top: 0;
  background: var(--bg);
  border-bottom: 1px solid var(--border);
  border-radius: 0;
  z-index: 2;
}

.cell { display: flex; align-items: center; gap: 9px; min-width: 0; padding-right: var(--s-3); }
.cell-check { justify-content: center; padding-right: 0; }
.cell-size { justify-content: flex-end; }
.cell-tools { justify-content: flex-end; gap: 2px; padding-right: 0; }
.head { font: inherit; letter-spacing: inherit; text-transform: inherit; color: inherit; height: 100%; }
.head:hover { color: var(--text); }
</style>

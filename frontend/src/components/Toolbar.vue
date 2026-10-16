<script setup>
import { ref, computed, watch } from 'vue';
import Icon from './Icon.vue';
import PathBar from './PathBar.vue';
import {
  state, activeTab, canGoBack, canGoForward, goBack, goForward, goUp, refreshTab,
  newTab, createFolder, createFile, clipboardPaste, clipboardHasItems,
  toggleHidden, setViewMode, setSort, openDialog, saveViewDefaults, updateSettings,
  startSearch, stopSearch, showTrash, addBookmark, openContextMenu,
} from '../store/app.js';

const pathBar = ref(null);
const searchInput = ref(null);
const searching = ref(false);

const tab = computed(() => activeTab.value);
const sortOptions = [
  { key: 'name', label: 'Name' },
  { key: 'size', label: 'Size' },
  { key: 'modified', label: 'Modified' },
  { key: 'created', label: 'Created' },
  { key: 'type', label: 'Type' },
  { key: 'ext', label: 'Extension' },
  { key: 'perm', label: 'Permissions' },
  { key: 'owner', label: 'Owner' },
];

async function toggleSearch() {
  searching.value = !searching.value;
  if (searching.value) {
    await new Promise((resolve) => requestAnimationFrame(resolve));
    searchInput.value?.focus();
    searchInput.value?.select();
  } else if (tab.value?.mode === 'search') {
    await stopSearch(tab.value);
  }
}

function onSearchKeydown(event) {
  if (event.key === 'Escape') {
    event.preventDefault();
    toggleSearch();
  }
}

watch(() => tab.value?.mode, (mode) => {
  if (mode !== 'search') searching.value = false;
});

function sortMenu(event) {
  openContextMenu(event, sortOptions.map((option) => ({
    label: option.label,
    checked: tab.value?.sortBy === option.key,
    run: () => setSort(tab.value, option.key),
  })).concat([
    { separator: true },
    {
      label: 'Folders first',
      checked: tab.value?.dirsFirst,
      run: () => { tab.value.dirsFirst = !tab.value.dirsFirst; refreshTab(tab.value); },
    },
    {
      label: 'Group by kind',
      checked: tab.value?.groupByKind,
      run: () => { tab.value.groupByKind = !tab.value.groupByKind; refreshTab(tab.value); },
    },
    { separator: true },
    { label: 'Save as default view', icon: 'save', run: () => saveViewDefaults(tab.value) },
  ]));
}

function togglePreview() {
  state.settings.showPreview = !state.settings.showPreview;
  updateSettings({ showPreview: state.settings.showPreview }, true);
}

function moreMenu(event) {
  openContextMenu(event, [
    {
      label: state.settings.showPreview ? 'Hide preview pane' : 'Show preview pane',
      icon: 'panel-right',
      shortcut: 'F11',
      run: togglePreview,
    },
    { label: 'Add to bookmarks', icon: 'bookmark', run: () => addBookmark(tab.value?.path) },
    { separator: true },
    { label: 'Trash', icon: 'trash', run: showTrash },
    { separator: true },
    { label: 'Settings', icon: 'settings', shortcut: 'Ctrl+,', run: () => openDialog('settings') },
    { label: 'Keyboard shortcuts', icon: 'keyboard', shortcut: 'Ctrl+/', run: () => openDialog('shortcuts') },
    { label: 'About Mura Files', icon: 'info', run: () => openDialog('about') },
  ]);
}

defineExpose({ focusPath: () => pathBar.value?.beginEdit() });
</script>

<template>
  <div class="toolbar">
    <div class="group">
      <button
        class="icon-btn"
        type="button"
        title="Back (Alt+Left)"
        :disabled="!canGoBack(tab)"
        @click="goBack(tab)"
      >
        <Icon name="chevron-left" :size="17" />
      </button>
      <button
        class="icon-btn"
        type="button"
        title="Forward (Alt+Right)"
        :disabled="!canGoForward(tab)"
        @click="goForward(tab)"
      >
        <Icon name="chevron-right" :size="17" />
      </button>
      <button class="icon-btn" type="button" title="Up (Alt+Up)" @click="goUp(tab)">
        <Icon name="arrow-up" :size="16" />
      </button>
      <button class="icon-btn" type="button" title="Refresh (F5)" @click="refreshTab(tab)">
        <Icon name="refresh" :size="15" :class="{ spinning: tab?.loading }" />
      </button>
    </div>

    <div class="vdivider" />

    <PathBar ref="pathBar" />

    <div class="search-wrap" :class="{ 'is-open': searching }">
      <Icon name="search" :size="14" class="search-icon" />
      <input
        ref="searchInput"
        v-model="tab.searchPattern"
        class="input search-input"
        type="search"
        placeholder="Search this folder…"
        spellcheck="false"
        @keydown.enter="startSearch(tab)"
        @keydown="onSearchKeydown"
      />
      <button
        v-if="tab?.mode === 'search'"
        class="icon-btn is-sm search-clear"
        type="button"
        title="Stop search"
        @click="stopSearch(tab)"
      >
        <Icon name="x" :size="13" />
      </button>
    </div>

    <button
      class="icon-btn"
      type="button"
      :class="{ 'is-active': searching }"
      title="Search (Ctrl+F)"
      @click="toggleSearch"
    >
      <Icon name="search" :size="16" />
    </button>

    <div class="vdivider" />

    <div class="group">
      <button class="icon-btn" type="button" title="New folder (Ctrl+Shift+N)" @click="createFolder(tab)">
        <Icon name="folder-plus" :size="16" />
      </button>
      <button class="icon-btn" type="button" title="New file (Ctrl+Alt+N)" @click="createFile(tab)">
        <Icon name="file-plus" :size="16" />
      </button>
      <button
        class="icon-btn"
        type="button"
        title="Paste (Ctrl+V)"
        :disabled="!clipboardHasItems"
        @click="clipboardPaste(tab)"
      >
        <Icon name="paste" :size="16" />
      </button>
    </div>

    <div class="vdivider" />

    <div class="segmented" role="group" aria-label="View mode">
      <button
        type="button"
        :class="{ 'is-active': tab?.viewMode === 'list' }"
        title="List view (Ctrl+1)"
        @click="setViewMode(tab, 'list')"
      >
        <Icon name="layout-list" :size="14" />
      </button>
      <button
        type="button"
        :class="{ 'is-active': tab?.viewMode === 'grid' }"
        title="Grid view (Ctrl+2)"
        @click="setViewMode(tab, 'grid')"
      >
        <Icon name="layout-grid" :size="14" />
      </button>
      <button
        type="button"
        :class="{ 'is-active': tab?.viewMode === 'columns' }"
        title="Compact view (Ctrl+3)"
        @click="setViewMode(tab, 'columns')"
      >
        <Icon name="layout-columns" :size="14" />
      </button>
    </div>

    <button class="icon-btn" type="button" title="Sort and grouping" @click="sortMenu">
      <Icon :name="tab?.sortDesc ? 'sort-desc' : 'sort-asc'" :size="16" />
    </button>

    <button
      class="icon-btn"
      type="button"
      :class="{ 'is-active': tab?.showHidden }"
      title="Show hidden files (Ctrl+H)"
      @click="toggleHidden(tab)"
    >
      <Icon :name="tab?.showHidden ? 'eye' : 'eye-off'" :size="16" />
    </button>

    <button
      class="icon-btn"
      type="button"
      :class="{ 'is-active': state.settings.showPreview }"
      title="Toggle preview pane (F11)"
      @click="togglePreview"
    >
      <Icon name="panel-right" :size="16" />
    </button>

    <button class="icon-btn" type="button" title="More actions" @click="moreMenu">
      <Icon name="more-horizontal" :size="16" />
    </button>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: var(--s-2);
  height: var(--toolbar-h);
  padding: 0 var(--s-3);
  background: var(--bg);
  border-bottom: 1px solid var(--border);
  flex: 0 0 auto;
}

.group { display: flex; align-items: center; gap: 1px; flex: 0 0 auto; }

.search-wrap {
  position: relative;
  display: flex;
  align-items: center;
  width: 0;
  opacity: 0;
  overflow: hidden;
  transition: width var(--t-slow), opacity var(--t-base);
}
.search-wrap.is-open { width: 230px; opacity: 1; }
.search-icon {
  position: absolute;
  left: 10px;
  color: var(--text-faint);
  pointer-events: none;
}
.search-input {
  height: 32px;
  padding-left: 30px;
  padding-right: 30px;
}
.search-input::-webkit-search-cancel-button { display: none; }
.search-clear { position: absolute; right: 4px; }

.more { display: flex; align-items: center; }

.spinning { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>

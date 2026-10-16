<script setup>
import { computed, ref } from 'vue';
import Icon from './Icon.vue';
import FileList from './FileList.vue';
import FileGrid from './FileGrid.vue';
import TrashView from './TrashView.vue';
import {
  state, activeTab, openEntry, buildBackgroundMenu, openContextMenu,
  refreshTab, createFolder, openDialog, stopSearch, startSearch, setSort,
  openArchive, extractSelection, copyPathToClipboard, navigate,
} from '../store/app.js';
import { pluralise } from '../lib/format.js';

const tab = computed(() => activeTab.value);
const contentEl = ref(null);

const columns = computed(() => {
  const list = ['name', 'size', 'kind', 'modified'];
  if (tab.value?.viewMode === 'columns') return ['name', 'size', 'modified'];
  return list;
});

function onBackgroundMenu(event) {
  if (event.target.closest('.row, .tile, .list-head')) return;
  event.preventDefault();
  openContextMenu(event, buildBackgroundMenu());
}

function retrySearch() {
  if (tab.value) startSearch(tab.value);
}

/** Breadcrumb for search results showing which folder each hit lives in. */
function hitFolder(entry) {
  return entry.dir || '';
}
</script>

<template>
  <div ref="contentEl" class="content" data-file-drop-target="content" @contextmenu="onBackgroundMenu">
    <!-- Search banner -->
    <div v-if="tab?.mode === 'search'" class="search-banner">
      <Icon name="search" :size="15" />
      <div class="grow">
        <span class="search-title">
          Results for <strong>{{ tab.searchPattern }}</strong> in
          <button class="link" type="button" @click="copyPathToClipboard(tab.path)">{{ tab.path }}</button>
        </span>
        <span class="search-meta faint">
          {{ pluralise(tab.searchFound, 'match', 'matches') }} ·
          {{ tab.searchScanned.toLocaleString() }} items scanned
          <template v-if="tab.searchRunning"> · still searching…</template>
        </span>
      </div>
      <button
        v-if="tab.searchRunning"
        class="btn btn-sm btn-outline"
        type="button"
        @click="stopSearch(tab)"
      >
        <Icon name="x" :size="13" /> Stop
      </button>
      <button v-else class="btn btn-sm btn-outline" type="button" @click="retrySearch">
        <Icon name="refresh" :size="13" /> Search again
      </button>
      <button class="btn btn-sm btn-ghost" type="button" @click="stopSearch(tab)">Close</button>
    </div>

    <!-- Archive banner -->
    <div v-if="tab?.mode === 'archive'" class="search-banner is-archive">
      <Icon name="file-archive" :size="15" />
      <div class="grow">
        <span class="search-title">
          Viewing <strong>{{ tab.archivePath.split('/').pop() }}</strong>
        </span>
        <span class="search-meta faint">
          {{ pluralise(tab.archiveInfo?.total || 0, 'entry', 'entries') }}
          <template v-if="tab.archiveInfo?.truncated"> · listing truncated</template>
        </span>
      </div>
      <button class="btn btn-sm btn-outline" type="button" @click="extractSelection(tab)">
        <Icon name="package" :size="13" /> Extract
      </button>
      <button class="btn btn-sm btn-ghost" type="button" @click="stopSearch(tab)">Close</button>
    </div>

    <!-- Error -->
    <div v-if="tab?.error && tab.mode === 'browse'" class="error-banner">
      <Icon name="alert-triangle" :size="15" />
      <span class="grow">{{ tab.error }}</span>
      <button class="btn btn-sm btn-outline" type="button" @click="refreshTab(tab)">Retry</button>
    </div>

    <!-- Trash -->
    <TrashView v-if="tab?.mode === 'trash'" />

    <!-- Empty folder -->
    <div v-else-if="tab && !tab.loading && !tab.entries.length && !tab.error" class="empty">
      <Icon
        :name="tab.mode === 'search' ? 'search' : 'folder-open'"
        :size="40"
        class="empty-icon"
      />
      <span class="empty-title">
        {{ tab.mode === 'search' ? (tab.searchRunning ? 'Searching…' : 'No matches found') : 'This folder is empty' }}
      </span>
      <span class="empty-text">
        {{ tab.mode === 'search'
          ? 'Try a different pattern, or enable hidden files to widen the search.'
          : 'Drop files here, or create something new to get started.' }}
      </span>
      <div v-if="tab.mode === 'browse'" class="empty-actions">
        <button class="btn btn-primary" type="button" @click="createFolder(tab)">
          <Icon name="folder-plus" :size="15" /> New Folder
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-else-if="tab?.loading && !tab.entries.length" class="loading">
      <Icon name="loader" :size="20" class="spin" />
      <span class="faint">Loading…</span>
    </div>

    <!-- Files -->
    <FileGrid
      v-else-if="tab?.viewMode === 'grid'"
      :entries="tab.entries"
      :size="state.settings.gridSize"
    />
    <FileList v-else :entries="tab.entries" :columns="columns" />
  </div>
</template>

<style scoped>
.content {
  position: relative;
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg);
  overflow: hidden;
}

.search-banner,
.error-banner {
  display: flex;
  align-items: center;
  gap: var(--s-3);
  padding: var(--s-2) var(--s-4);
  border-bottom: 1px solid var(--border);
  flex: 0 0 auto;
  background: var(--surface);
}
.search-banner.is-archive { background: var(--surface-2); }
.error-banner { background: var(--danger-soft); color: var(--danger); }

.search-title { display: block; font-size: var(--fs-md); }
.search-title strong { font-weight: 600; }
.search-meta { display: block; font-size: var(--fs-xs); }
.link {
  color: var(--accent);
  text-decoration: underline;
  text-underline-offset: 2px;
  max-width: 40ch;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
  display: inline-block;
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--s-3);
  height: 100%;
  font-size: var(--fs-sm);
}

.empty-actions { margin-top: var(--s-3); }
.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>

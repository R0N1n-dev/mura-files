<script setup>
import { computed, ref, watch } from 'vue';
import Icon from './Icon.vue';
import {
  state, activeTab, selectedEntries, previewEntry, openEntry, showProperties,
  startRename, openInTerminal, copyPathToClipboard, showAppPicker, openContextMenu,
  clipboardCopy, deleteSelection, buildEntryMenu, isArchiveName, extractSelection,
} from '../store/app.js';
import { formatSize, formatDate, describeKind, pluralise } from '../lib/format.js';
import { fileUrl, FileService } from '../lib/fs.js';

const textPreview = ref(null);
const textLoading = ref(false);
const textError = ref('');
const showAll = ref(false);

const tab = computed(() => activeTab.value);
const entries = computed(() => selectedEntries.value);
const entry = computed(() => previewEntry.value);
const multiple = computed(() => entries.value.length > 1);

const kind = computed(() => entry.value?.kind || 'other');
const isImage = computed(() => kind.value === 'image');
const isVideo = computed(() => kind.value === 'video');
const isAudio = computed(() => kind.value === 'audio');
const isPdf = computed(() => entry.value?.ext === '.pdf');
const isText = computed(() => ['text', 'code'].includes(kind.value) || ['.json', '.yml', '.yaml', '.toml', '.ini', '.conf', '.md', '.log', '.sh', '.desktop'].includes(entry.value?.ext));
const isArchive = computed(() => !!entry.value && !entry.value.isDir && isArchiveName(entry.value.name));

watch(entry, async (value) => {
  textPreview.value = null;
  textError.value = '';
  showAll.value = false;
  if (!value || !isText.value || value.isDir) return;
  textLoading.value = true;
  try {
    textPreview.value = await FileService.ReadText(value.path, 96 * 1024);
    if (textPreview.value?.binary) {
      textPreview.value = null;
      textError.value = 'Binary file — no text preview available.';
    }
  } catch (error) {
    textError.value = String(error?.message || error);
  } finally {
    textLoading.value = false;
  }
});

const infoRows = computed(() => {
  const value = entry.value;
  if (!value) return [];
  const rows = [
    ['Name', value.desktopName || value.name],
    ['Type', describeKind(value)],
    ['Size', value.isDir ? '--' : formatSize(value.size)],
    ['Modified', formatDate(value.modTime, 'full')],
  ];
  if (value.isDir) rows.splice(2, 0, ['Contents', 'Folder']);
  if (!value.isDir) rows.push(['Accessed', formatDate(value.accessTime, 'full')]);
  rows.push(['Permissions', `${value.perm} (${value.mode})`]);
  rows.push(['Owner', `${value.owner}:${value.group}`]);
  if (value.isSymlink) rows.push(['Target', value.target]);
  if (value.mime) rows.push(['MIME type', value.mime]);
  return rows;
});

const aggregate = computed(() => {
  const list = entries.value;
  const folders = list.filter((item) => item.isDir).length;
  const files = list.length - folders;
  const bytes = list.reduce((total, item) => total + (item.isDir ? 0 : item.size || 0), 0);
  return { folders, files, bytes };
});

function onMenu(event) {
  if (!entry.value) return;
  openContextMenu(event, buildEntryMenu(entry.value));
}
</script>

<template>
  <aside class="preview" @contextmenu="onMenu">
    <div v-if="!entries.length" class="empty">
      <Icon name="eye" :size="30" class="empty-icon" />
      <p class="empty-text">Select a file to see a preview and its details.</p>
    </div>

    <template v-else>
      <div v-if="multiple" class="preview-head">
        <Icon name="layout-grid" :size="16" />
        <div class="grow">
          <div class="preview-title">{{ pluralise(entries.length, 'item') }}</div>
          <div class="preview-sub muted">
            {{ aggregate.folders }} folders · {{ aggregate.files }} files · {{ formatSize(aggregate.bytes) }}
          </div>
        </div>
      </div>

      <template v-else-if="entry">
        <div class="preview-head">
          <Icon :name="entry.icon || 'file'" :size="16" />
          <div class="grow">
            <div class="preview-title truncate" :title="entry.name">{{ entry.desktopName || entry.name }}</div>
            <div class="preview-sub muted truncate">{{ describeKind(entry) }}</div>
          </div>
          <button class="icon-btn is-sm" type="button" title="Rename" @click="startRename(entry.path, entry.name)">
            <Icon name="pencil" :size="13" />
          </button>
          <button class="icon-btn is-sm" type="button" title="Open" @click="openEntry(tab, entry)">
            <Icon name="external-link" :size="13" />
          </button>
        </div>

        <div class="preview-media">
          <img
            v-if="isImage"
            :src="fileUrl(entry.path)"
            :alt="entry.name"
            class="media-image"
            draggable="false"
          />
          <video
            v-else-if="isVideo"
            :src="fileUrl(entry.path)"
            class="media-video"
            controls
            preload="metadata"
          />
          <div v-else-if="isAudio" class="media-audio">
            <Icon name="file-audio" :size="34" />
            <audio :src="fileUrl(entry.path)" controls preload="metadata" />
          </div>
          <iframe
            v-else-if="isPdf"
            :src="fileUrl(entry.path)"
            class="media-frame"
            title="PDF preview"
          />
          <div v-else-if="isText" class="media-text">
            <div v-if="textLoading" class="media-note faint">
              <Icon name="loader" :size="16" class="spin" /> Loading…
            </div>
            <pre v-else-if="textPreview" class="text-body" :class="{ 'is-clamped': !showAll }">{{ textPreview.content }}</pre>
            <div v-else class="media-note faint">{{ textError || 'Nothing to preview.' }}</div>
            <button
              v-if="textPreview && textPreview.truncated"
              class="btn btn-sm btn-outline text-more"
              type="button"
              @click="showAll = !showAll"
            >
              {{ showAll ? 'Show less' : `Show all (${formatSize(textPreview.size)})` }}
            </button>
          </div>
          <div v-else class="media-fallback">
            <Icon :name="entry.icon || 'file'" :size="46" :stroke="1.2" />
            <span class="faint">{{ formatSize(entry.size) }}</span>
          </div>
        </div>

        <div v-if="isArchive" class="preview-actions">
          <button class="btn btn-sm btn-outline grow" type="button" @click="extractSelection(tab)">
            <Icon name="package" :size="14" /> Extract Here
          </button>
        </div>

        <div class="preview-actions">
          <button class="btn btn-sm btn-outline grow" type="button" @click="showAppPicker(entry)">
            <Icon name="sparkles" :size="14" /> Open With
          </button>
          <button class="btn btn-sm btn-outline grow" type="button" @click="openInTerminal(entry.isDir ? entry.path : entry.dir)">
            <Icon name="terminal" :size="14" /> Terminal
          </button>
        </div>

        <dl class="preview-info">
          <template v-for="row in infoRows" :key="row[0]">
            <dt>{{ row[0] }}</dt>
            <dd class="truncate" :title="String(row[1])">{{ row[1] }}</dd>
          </template>
        </dl>

        <div class="preview-foot">
          <button class="btn btn-sm btn-ghost" type="button" @click="showProperties([entry.path])">
            <Icon name="info" :size="13" /> Properties
          </button>
          <button class="btn btn-sm btn-ghost" type="button" @click="copyPathToClipboard(entry.path)">
            <Icon name="link" :size="13" /> Copy path
          </button>
        </div>
      </template>
    </template>
  </aside>
</template>

<style scoped>
.preview {
  width: var(--preview-w, 320px);
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  background: var(--surface);
  border-left: 1px solid var(--border);
  overflow: hidden;
}

.preview-head {
  display: flex;
  align-items: center;
  gap: var(--s-3);
  padding: var(--s-3);
  border-bottom: 1px solid var(--border);
  flex: 0 0 auto;
}
.preview-title { font-size: var(--fs-md); font-weight: 600; }
.preview-sub { font-size: var(--fs-xs); }

.preview-media {
  flex: 0 1 auto;
  max-height: 46vh;
  min-height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--s-3);
  background: var(--bg-deep);
  border-bottom: 1px solid var(--border);
  overflow: hidden;
}
.media-image { max-width: 100%; max-height: 42vh; object-fit: contain; border-radius: var(--r-sm); }
.media-video { width: 100%; max-height: 42vh; border-radius: var(--r-sm); background: #000; }
.media-frame { width: 100%; height: 42vh; border: 0; border-radius: var(--r-sm); background: #fff; }
.media-audio { display: flex; flex-direction: column; align-items: center; gap: var(--s-3); color: var(--text-muted); }
.media-audio audio { width: 100%; }

.media-text {
  width: 100%;
  height: 100%;
  max-height: 42vh;
  display: flex;
  flex-direction: column;
  gap: var(--s-2);
  overflow: hidden;
}
.text-body {
  flex: 1 1 auto;
  margin: 0;
  padding: 0;
  overflow: auto;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  line-height: 1.55;
  color: var(--text-muted);
  white-space: pre-wrap;
  word-break: break-word;
  user-select: text;
}
.text-body.is-clamped { max-height: 22vh; }
.media-note { display: flex; align-items: center; gap: var(--s-2); font-size: var(--fs-sm); }
.media-fallback { display: flex; flex-direction: column; align-items: center; gap: var(--s-2); color: var(--text-muted); }
.text-more { align-self: flex-start; }

.preview-actions {
  display: flex;
  gap: var(--s-2);
  padding: var(--s-2) var(--s-3);
  border-bottom: 1px solid var(--border);
}

.preview-info {
  flex: 1 1 auto;
  display: grid;
  grid-template-columns: 82px minmax(0, 1fr);
  gap: 5px var(--s-2);
  margin: 0;
  padding: var(--s-3);
  overflow: auto;
  font-size: var(--fs-xs);
}
.preview-info dt { color: var(--text-faint); }
.preview-info dd { margin: 0; color: var(--text-muted); user-select: text; }

.preview-foot {
  display: flex;
  gap: var(--s-2);
  padding: var(--s-2) var(--s-3) var(--s-3);
  border-top: 1px solid var(--border);
}

.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>

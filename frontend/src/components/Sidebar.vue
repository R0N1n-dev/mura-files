<script setup>
import { computed, ref } from 'vue';
import Icon from './Icon.vue';
import {
  state, activeTab, navigate, newTab, showTrash, addBookmark, removeBookmark,
  renameBookmark, reorderBookmarks, ejectDrive, refreshDrives, openContextMenu,
  openInTerminal, copyPathToClipboard, toggleTreeNode, loadTreeNode, expandTreeTo,
  openPrompt, openDialog, showProperties,
} from '../store/app.js';
import { formatSize } from '../lib/format.js';

const sectionCollapsed = ref({ bookmarks: false, places: false, drives: false, tree: false });
const dragBookmark = ref('');

const places = computed(() => state.places.filter((place) => place.kind !== 'trash'));
const currentPath = computed(() => activeTab.value?.path || '');
const activeKind = computed(() => activeTab.value?.mode || 'browse');

function isCurrent(path) {
  return activeKind.value === 'browse' && currentPath.value === path;
}

function go(path) {
  const tab = activeTab.value;
  if (tab) navigate(tab, path);
}

function placeIcon(place) {
  const map = {
    home: 'home', desktop: 'desktop', document: 'documents', download: 'download',
    music: 'music', picture: 'pictures', video: 'videos', public: 'public',
    template: 'templates', trash: 'trash', root: 'root',
  };
  return map[place.kind] || 'folder';
}

function driveIcon(drive) {
  if (drive.network) return 'network';
  if (drive.removable) return 'drive-removable';
  if (drive.path === '/') return 'hard-drive';
  return 'hard-drive';
}

function usageLabel(drive) {
  if (!drive.total) return drive.fsType || '';
  return `${formatSize(drive.free)} free of ${formatSize(drive.total)}`;
}

/* ------------------------------ context menus ---------------------------- */

function placeMenu(event, place) {
  event.preventDefault();
  openContextMenu(event, [
    { label: 'Open', icon: 'folder', run: () => go(place.path) },
    { label: 'Open in new tab', icon: 'plus', run: () => newTab(place.path) },
    { separator: true },
    { label: 'Open in Terminal', icon: 'terminal', run: () => openInTerminal(place.path) },
    { label: 'Copy path', icon: 'link', run: () => copyPathToClipboard(place.path) },
    { separator: true },
    { label: 'Add to bookmarks', icon: 'bookmark', run: () => addBookmark(place.path) },
    { label: 'Properties', icon: 'info', run: () => showProperties([place.path]) },
  ]);
}

function bookmarkMenu(event, bookmark) {
  event.preventDefault();
  openContextMenu(event, [
    { label: 'Open', icon: 'folder', run: () => go(bookmark.path) },
    { label: 'Open in new tab', icon: 'plus', run: () => newTab(bookmark.path) },
    { separator: true },
    {
      label: 'Rename…',
      icon: 'pencil',
      run: () => openPrompt({
        title: 'Rename Bookmark',
        label: 'Name',
        value: bookmark.name,
        confirmLabel: 'Rename',
        onConfirm: (name) => renameBookmark(bookmark.path, name),
      }),
    },
    { label: 'Copy path', icon: 'link', run: () => copyPathToClipboard(bookmark.path) },
    { separator: true },
    { label: 'Remove bookmark', icon: 'trash', danger: true, run: () => removeBookmark(bookmark.path) },
  ]);
}

function driveMenu(event, drive) {
  event.preventDefault();
  openContextMenu(event, [
    { label: 'Open', icon: 'folder', run: () => go(drive.path) },
    { label: 'Open in new tab', icon: 'plus', run: () => newTab(drive.path) },
    { separator: true },
    { label: 'Open in Terminal', icon: 'terminal', run: () => openInTerminal(drive.path) },
    { label: 'Copy path', icon: 'link', run: () => copyPathToClipboard(drive.path) },
    { separator: true },
    {
      label: 'Eject',
      icon: 'eject',
      danger: true,
      disabled: !drive.removable,
      run: () => ejectDrive(drive),
    },
  ]);
}

function bookmarkDragStart(event, bookmark) {
  dragBookmark.value = bookmark.path;
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('text/plain', bookmark.path);
}

function bookmarkDrop(event, target) {
  event.preventDefault();
  if (!dragBookmark.value || dragBookmark.value === target.path) return;
  const paths = state.bookmarks.map((bookmark) => bookmark.path);
  const from = paths.indexOf(dragBookmark.value);
  const to = paths.indexOf(target.path);
  if (from < 0 || to < 0) return;
  paths.splice(to, 0, paths.splice(from, 1)[0]);
  reorderBookmarks(paths);
  dragBookmark.value = '';
}

/* -------------------------------- folder tree ---------------------------- */

function treeNodeMenu(event, entry) {
  event.preventDefault();
  openContextMenu(event, [
    { label: 'Open', icon: 'folder', run: () => go(entry.path) },
    { label: 'Open in new tab', icon: 'plus', run: () => newTab(entry.path) },
    { separator: true },
    { label: 'Open in Terminal', icon: 'terminal', run: () => openInTerminal(entry.path) },
    { label: 'Copy path', icon: 'link', run: () => copyPathToClipboard(entry.path) },
  ]);
}

function toggleSection(key) {
  sectionCollapsed.value[key] = !sectionCollapsed.value[key];
}

defineExpose({ expandTreeTo, loadTreeNode });
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-scroll">
      <!-- Places -->
      <section class="section">
        <button class="section-head" type="button" @click="toggleSection('places')">
          <Icon :name="sectionCollapsed.places ? 'chevron-right' : 'chevron-down'" :size="13" />
          <span>Places</span>
        </button>
        <div v-show="!sectionCollapsed.places" class="section-body">
          <button
            v-for="place in places"
            :key="place.path"
            class="item"
            :class="{ 'is-active': isCurrent(place.path), 'is-missing': !place.present }"
            type="button"
            :title="place.path"
            @click="go(place.path)"
            @contextmenu="placeMenu($event, place)"
            @auxclick.middle="newTab(place.path)"
          >
            <Icon :name="placeIcon(place)" :size="16" class="item-icon" />
            <span class="item-label truncate">{{ place.name }}</span>
          </button>
        </div>
      </section>

      <!-- Bookmarks -->
      <section class="section">
        <button class="section-head" type="button" @click="toggleSection('bookmarks')">
          <Icon :name="sectionCollapsed.bookmarks ? 'chevron-right' : 'chevron-down'" :size="13" />
          <span>Bookmarks</span>
          <span class="section-action" role="button" title="Bookmark current folder" @click.stop="addBookmark()">
            <Icon name="plus" :size="13" />
          </span>
        </button>
        <div v-show="!sectionCollapsed.bookmarks" class="section-body">
          <p v-if="!state.bookmarks.length" class="section-empty">
            Press the + button to bookmark the folder you are browsing.
          </p>
          <button
            v-for="bookmark in state.bookmarks"
            :key="bookmark.path"
            class="item"
            :class="{ 'is-active': isCurrent(bookmark.path) }"
            type="button"
            draggable="true"
            :title="bookmark.path"
            @click="go(bookmark.path)"
            @contextmenu="bookmarkMenu($event, bookmark)"
            @dragstart="bookmarkDragStart($event, bookmark)"
            @dragover.prevent
            @drop="bookmarkDrop($event, bookmark)"
            @auxclick.middle="newTab(bookmark.path)"
          >
            <Icon name="bookmark" :size="15" class="item-icon bookmark-icon" />
            <span class="item-label truncate">{{ bookmark.name }}</span>
          </button>
        </div>
      </section>

      <!-- Drives -->
      <section class="section">
        <button class="section-head" type="button" @click="toggleSection('drives')">
          <Icon :name="sectionCollapsed.drives ? 'chevron-right' : 'chevron-down'" :size="13" />
          <span>Drives</span>
          <span class="section-action" role="button" title="Rescan drives" @click.stop="refreshDrives">
            <Icon name="refresh" :size="12" />
          </span>
        </button>
        <div v-show="!sectionCollapsed.drives" class="section-body">
          <button
            v-for="drive in state.drives"
            :key="drive.path + drive.device"
            class="item drive"
            :class="{ 'is-active': isCurrent(drive.path) }"
            type="button"
            :title="`${drive.device}\n${usageLabel(drive)}`"
            @click="go(drive.path)"
            @contextmenu="driveMenu($event, drive)"
            @auxclick.middle="newTab(drive.path)"
          >
            <Icon :name="driveIcon(drive)" :size="16" class="item-icon" />
            <span class="drive-info">
              <span class="item-label truncate">{{ drive.name }}</span>
              <span v-if="drive.total" class="drive-bar" :title="usageLabel(drive)">
                <span
                  class="drive-bar-fill"
                  :class="{ 'is-high': drive.percent > 88 }"
                  :style="{ width: `${Math.min(100, drive.percent || 0)}%` }"
                />
              </span>
            </span>
          </button>
        </div>
      </section>

      <!-- Trash shortcut -->
      <section class="section">
        <div class="section-body">
          <button
            class="item"
            :class="{ 'is-active': activeKind === 'trash' }"
            type="button"
            title="Trash"
            @click="showTrash()"
          >
            <Icon name="trash" :size="16" class="item-icon" />
            <span class="item-label truncate">Trash</span>
          </button>
          <button class="item" type="button" title="Settings" @click="openDialog('settings')">
            <Icon name="settings" :size="16" class="item-icon" />
            <span class="item-label truncate">Settings</span>
          </button>
        </div>
      </section>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: var(--sidebar-w, 244px);
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  background: var(--surface);
  border-right: 1px solid var(--border);
  overflow: hidden;
}

.sidebar-scroll {
  flex: 1 1 auto;
  overflow-y: auto;
  overflow-x: hidden;
  padding: var(--s-2) var(--s-2) var(--s-4);
}

.section { margin-bottom: var(--s-2); }

.section-head {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  height: 26px;
  padding: 0 8px;
  border-radius: var(--r-sm);
  color: var(--text-faint);
  font-size: 10.5px;
  font-weight: 650;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  transition: color var(--t-fast);
}
.section-head:hover { color: var(--text-muted); }
.section-action {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 5px;
  opacity: 0;
  transition: opacity var(--t-fast), background var(--t-fast);
}
.section-head:hover .section-action { opacity: 1; }
.section-action:hover { background: var(--hover); color: var(--text); }

.section-body { display: flex; flex-direction: column; gap: 1px; }
.section-empty {
  padding: 4px 10px 8px;
  font-size: var(--fs-xs);
  color: var(--text-faint);
  line-height: 1.5;
}

.item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 30px;
  padding: 0 9px;
  border-radius: var(--r-sm);
  color: var(--text-muted);
  font-size: var(--fs-md);
  text-align: left;
  transition: background var(--t-fast), color var(--t-fast);
}
.item:hover { background: var(--hover); color: var(--text); }
.item.is-active {
  background: var(--accent-soft);
  color: var(--accent);
  font-weight: 550;
}
.item.is-missing { opacity: 0.45; }
.item-icon { flex: 0 0 auto; opacity: 0.85; }
.item.is-active .item-icon { opacity: 1; }
.item-label { flex: 1 1 auto; min-width: 0; }
.bookmark-icon { color: var(--text-faint); }

.drive { height: auto; min-height: 34px; padding-top: 4px; padding-bottom: 4px; }
.drive-info { display: flex; flex-direction: column; gap: 4px; flex: 1 1 auto; min-width: 0; }
.drive-bar {
  display: block;
  height: 3px;
  border-radius: 2px;
  background: var(--surface-3);
  overflow: hidden;
}
.drive-bar-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--accent);
  opacity: 0.75;
  transition: width var(--t-base);
}
.drive-bar-fill.is-high { background: var(--warning); opacity: 1; }
</style>

<script setup>
import { ref } from 'vue';
import Icon from './Icon.vue';
import {
  state, tabById, newTab, selectTab, closeTab, closeOtherTabs,
  closeTabsToRight, duplicateTab, reorderTabs, openContextMenu, openPathsInNewTab,
} from '../store/app.js';

const dragId = ref('');
const dropIndex = ref(-1);

function onDragStart(event, tab) {
  dragId.value = tab.id;
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('text/plain', tab.path || '');
}

function onDragOver(event, index) {
  if (!dragId.value) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = 'move';
  dropIndex.value = index;
}

function onDrop(event, index) {
  event.preventDefault();
  const external = event.dataTransfer.getData('text/plain');
  if (!dragId.value && external) {
    // A path dragged in from another window opens in a new tab.
    openPathsInNewTab([external]);
  } else if (dragId.value) {
    reorderTabs(dragId.value, index);
  }
  dragId.value = '';
  dropIndex.value = -1;
}

function onDragEnd() {
  dragId.value = '';
  dropIndex.value = -1;
}

function tabMenu(event, tab) {
  event.preventDefault();
  selectTab(tab.id);
  const index = state.tabs.findIndex((item) => item.id === tab.id);
  openContextMenu(event, [
    { label: 'New Tab', icon: 'plus', run: () => newTab(tab.path) },
    { separator: true },
    { label: 'Duplicate Tab', icon: 'copy', run: () => duplicateTab(tab.id) },
    { label: 'Close Tab', icon: 'x', shortcut: 'Ctrl+W', run: () => closeTab(tab.id) },
    {
      label: 'Close Other Tabs',
      icon: 'x-circle',
      disabled: state.tabs.length < 2,
      run: () => closeOtherTabs(tab.id),
    },
    {
      label: 'Close Tabs to the Right',
      icon: 'chevron-right',
      disabled: index >= state.tabs.length - 1,
      run: () => closeTabsToRight(tab.id),
    },
    { separator: true },
    {
      label: 'Copy Path',
      icon: 'link',
      run: () => navigator.clipboard?.writeText(tab.path).catch(() => {}),
    },
  ]);
}

function middleClick(event, tab) {
  if (event.button !== 1) return;
  event.preventDefault();
  closeTab(tab.id);
}
</script>

<template>
  <div class="tabbar">
    <div class="tabs" role="tablist">
      <button
        v-for="(tab, index) in state.tabs"
        :key="tab.id"
        class="tab"
        :class="{
          'is-active': tab.id === state.activeTabId,
          'is-drop-before': dropIndex === index,
          'is-dragging': dragId === tab.id,
        }"
        role="tab"
        :aria-selected="tab.id === state.activeTabId"
        draggable="true"
        :title="tab.path"
        @click="selectTab(tab.id)"
        @mousedown.middle="middleClick($event, tab)"
        @auxclick.middle="middleClick($event, tab)"
        @contextmenu="tabMenu($event, tab)"
        @dragstart="onDragStart($event, tab)"
        @dragover="onDragOver($event, index)"
        @drop="onDrop($event, index)"
        @dragend="onDragEnd"
      >
        <Icon
          :name="tab.mode === 'search' ? 'search' : tab.mode === 'trash' ? 'trash' : tab.mode === 'archive' ? 'file-archive' : 'folder'"
          :size="13"
          class="tab-icon"
        />
        <span class="tab-label truncate">{{ tab.title || 'Home' }}</span>
        <span
          class="tab-close"
          role="button"
          :aria-label="`Close ${tab.title}`"
          @click.stop="closeTab(tab.id)"
        >
          <Icon name="x" :size="12" />
        </span>
      </button>
    </div>

    <button class="icon-btn new-tab" type="button" title="New tab (Ctrl+T)" @click="newTab()">
      <Icon name="plus" :size="15" />
    </button>
  </div>
</template>

<style scoped>
.tabbar {
  display: flex;
  align-items: center;
  gap: var(--s-1);
  height: var(--tabbar-h);
  padding: 0 var(--s-2) 0 var(--s-3);
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  flex: 0 0 auto;
}

.tabs {
  display: flex;
  align-items: stretch;
  gap: 3px;
  flex: 1 1 auto;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
}
.tabs::-webkit-scrollbar { display: none; }

.tab {
  position: relative;
  display: flex;
  align-items: center;
  gap: 7px;
  max-width: 210px;
  min-width: 108px;
  padding: 0 8px 0 10px;
  border-radius: var(--r-sm) var(--r-sm) 0 0;
  color: var(--text-muted);
  font-size: var(--fs-sm);
  font-weight: 500;
  transition: background var(--t-fast), color var(--t-fast);
  flex: 0 0 auto;
}
.tab:hover { background: var(--hover); color: var(--text); }
.tab.is-active {
  background: var(--bg);
  color: var(--text);
}
.tab.is-active::after {
  content: "";
  position: absolute;
  left: 8px;
  right: 8px;
  bottom: -1px;
  height: 2px;
  border-radius: 2px 2px 0 0;
  background: var(--accent);
}
.tab.is-drop-before::before {
  content: "";
  position: absolute;
  left: -2px;
  top: 5px;
  bottom: 5px;
  width: 2px;
  border-radius: 2px;
  background: var(--accent);
}
.tab.is-dragging { opacity: 0.45; }

.tab-icon { opacity: 0.75; }
.tab.is-active .tab-icon { color: var(--accent); opacity: 1; }
.tab-label { flex: 1 1 auto; min-width: 0; }

.tab-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 5px;
  color: var(--text-faint);
  opacity: 0;
  transition: background var(--t-fast), opacity var(--t-fast), color var(--t-fast);
}
.tab:hover .tab-close,
.tab.is-active .tab-close { opacity: 1; }
.tab-close:hover { background: var(--active); color: var(--text); }

.new-tab { flex: 0 0 auto; }
</style>

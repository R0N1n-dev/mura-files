<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue';
import Icon from './Icon.vue';
import {
  state, activeTab, selectedEntries, selectionSize, openDialog,
  clearFinishedOperations, cancelOperation, openContextMenu, emptyTrash,
} from '../store/app.js';
import { formatSize, formatDuration, pluralise, shortenPath } from '../lib/format.js';
import { FileService } from '../lib/fs.js';

const tab = computed(() => activeTab.value);
const usage = ref(null);
let usageTimer = null;

async function refreshUsage() {
  const path = tab.value?.path;
  if (!path) return;
  try {
    usage.value = await FileService.Usage(path);
  } catch {
    usage.value = null;
  }
}

onMounted(() => {
  refreshUsage();
  usageTimer = setInterval(refreshUsage, 15000);
});

onUnmounted(() => {
  if (usageTimer) clearInterval(usageTimer);
});

const summary = computed(() => {
  const current = tab.value;
  if (!current) return '';
  if (current.mode === 'search') {
    if (current.searchRunning) {
      return `Searching… ${current.searchFound.toLocaleString()} found, ${current.searchScanned.toLocaleString()} scanned`;
    }
    return `${pluralise(current.searchFound, 'result')} in ${formatDuration(current.searchElapsed / 1000)}`;
  }
  if (current.mode === 'trash') return pluralise(state.trash.length, 'item');
  if (current.mode === 'archive') {
    const info = current.archiveInfo;
    if (!info) return '';
    return `${pluralise(info.total, 'entry', 'entries')} · ${formatSize(info.uncompressed)}`;
  }
  const total = current.entries.length;
  const hidden = current.listing?.hidden || 0;
  const folders = current.entries.filter((entry) => entry.isDir).length;
  const files = total - folders;
  let text = `${pluralise(total, 'item')} (${folders} folders, ${files} files)`;
  if (hidden && !current.showHidden) text += ` · ${hidden} hidden`;
  return text;
});

const selectionText = computed(() => {
  const entries = selectedEntries.value;
  if (!entries.length) return '';
  const folders = entries.filter((entry) => entry.isDir).length;
  const files = entries.length - folders;
  let text = `Selected ${pluralise(entries.length, 'item')}`;
  if (files) text += ` · ${formatSize(selectionSize.value)}`;
  if (folders && files) text += ` (${folders} folders)`;
  return text;
});

const runningOps = computed(() => state.ops.filter((op) => op.status === 'running' || op.status === 'queued'));
const primaryOp = computed(() => runningOps.value[0] || null);

const usagePercent = computed(() => Math.min(100, usage.value?.percent || 0));

function opsMenu(event) {
  const finished = state.ops.filter((op) => op.status !== 'running' && op.status !== 'queued');
  const items = [];
  if (finished.length) {
    items.push({ label: 'Clear finished', icon: 'x-circle', run: () => clearFinishedOperations() });
    items.push({ separator: true });
  }
  if (!state.ops.length) {
    items.push({ label: 'No recent operations', disabled: true });
  }
  state.ops.slice().reverse().forEach((op) => {
    const label = `${op.title || op.kind} — ${op.status}`;
    items.push({
      label,
      icon: op.status === 'running' ? 'loader' : op.status === 'error' ? 'alert-circle' : 'check-circle',
      run: () => {},
    });
    if (op.status === 'running') {
      items.push({ label: '  Cancel', icon: 'x', run: () => cancelOperation(op.id) });
    }
  });
  openContextMenu(event, items);
}

function usageMenu(event) {
  openContextMenu(event, [
    { label: usage.value?.fsType ? `Filesystem: ${usage.value.fsType}` : 'Filesystem', disabled: true },
    { label: `Free: ${formatSize(usage.value?.free || 0)}`, disabled: true },
    { label: `Total: ${formatSize(usage.value?.total || 0)}`, disabled: true },
    { separator: true },
    { label: 'Refresh', icon: 'refresh', run: refreshUsage },
  ]);
}

function trashMenu(event) {
  openContextMenu(event, [
    { label: 'Empty Trash', icon: 'trash-2', danger: true, run: emptyTrash },
  ]);
}
</script>

<template>
  <footer class="statusbar">
    <div class="left">
      <button class="stat" type="button" @click="openDialog('about')">
        <Icon name="info" :size="12" />
        <span>{{ summary }}</span>
      </button>
    </div>

    <div class="center">
      <span v-if="selectionText" class="stat is-accent">{{ selectionText }}</span>
      <span v-else-if="tab?.mode === 'search' && tab.searchRunning" class="stat">
        <Icon name="loader" :size="12" class="spin" />
        <span>Scanning…</span>
      </span>
    </div>

    <div class="right">
      <button
        v-if="tab?.mode === 'trash'"
        class="stat"
        type="button"
        title="Trash actions"
        @click="trashMenu"
      >
        <Icon name="more-horizontal" :size="13" />
      </button>

      <button
        v-if="primaryOp"
        class="stat op"
        type="button"
        :title="primaryOp.title"
        @click="opsMenu"
      >
        <Icon name="loader" :size="12" class="spin" />
        <span class="op-title truncate">{{ primaryOp.title }}</span>
        <span class="op-bar">
          <span class="op-bar-fill" :style="{ width: `${Math.min(100, primaryOp.percent || 0)}%` }" />
        </span>
        <span class="tnum">{{ Math.round(primaryOp.percent || 0) }}%</span>
        <span v-if="runningOps.length > 1" class="chip">+{{ runningOps.length - 1 }}</span>
      </button>

      <button class="stat" type="button" :title="shortenPath(tab?.path || '')" @click="usageMenu">
        <Icon name="hard-drive" :size="12" />
        <span class="usage-bar" :title="`${usagePercent.toFixed(0)}% used`">
          <span
            class="usage-bar-fill"
            :class="{ 'is-high': usagePercent > 88 }"
            :style="{ width: `${usagePercent}%` }"
          />
        </span>
        <span class="tnum">{{ formatSize(usage?.free || 0) }} free</span>
      </button>

      <button class="stat" type="button" title="Settings" @click="openDialog('settings')">
        <Icon name="settings" :size="13" />
      </button>
    </div>
  </footer>
</template>

<style scoped>
.statusbar {
  display: flex;
  align-items: center;
  gap: var(--s-2);
  height: var(--statusbar-h);
  padding: 0 var(--s-2) 0 var(--s-3);
  background: var(--surface);
  border-top: 1px solid var(--border);
  font-size: var(--fs-xs);
  color: var(--text-muted);
  flex: 0 0 auto;
}

.left, .right { display: flex; align-items: center; gap: 2px; flex: 1 1 0; min-width: 0; }
.center { display: flex; align-items: center; justify-content: center; flex: 0 1 auto; min-width: 0; }
.right { justify-content: flex-end; }

.stat {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 20px;
  padding: 0 7px;
  border-radius: var(--r-xs);
  color: inherit;
  font-size: inherit;
  transition: background var(--t-fast), color var(--t-fast);
  min-width: 0;
}
.stat:hover { background: var(--hover); color: var(--text); }
.stat.is-accent { color: var(--accent); }

.op { max-width: 320px; }
.op-title { max-width: 140px; }
.op-bar {
  display: block;
  width: 60px;
  height: 3px;
  border-radius: 2px;
  background: var(--surface-3);
  overflow: hidden;
}
.op-bar-fill {
  display: block;
  height: 100%;
  background: var(--accent);
  transition: width var(--t-base);
}

.usage-bar {
  display: block;
  width: 46px;
  height: 4px;
  border-radius: 2px;
  background: var(--surface-3);
  overflow: hidden;
}
.usage-bar-fill {
  display: block;
  height: 100%;
  background: var(--accent);
  opacity: 0.8;
  transition: width var(--t-base);
}
.usage-bar-fill.is-high { background: var(--warning); opacity: 1; }

.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>

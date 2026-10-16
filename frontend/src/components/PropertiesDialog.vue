<script setup>
import { ref, computed } from 'vue';
import ModalShell from './ModalShell.vue';
import Icon from './Icon.vue';
import { state, closeProperties, computeChecksum } from '../store/app.js';
import { formatSize, formatDate, describeKind, pluralise } from '../lib/format.js';

const checksumAlgo = ref('sha256');

const props = computed(() => state.dialogs.properties);
const single = computed(() => (props.value?.entries?.length === 1 ? props.value.entries[0] : null));

const totalSize = computed(() => {
  const count = props.value?.count;
  if (count) return count.bytes;
  return (props.value?.entries || []).reduce((sum, entry) => sum + (entry.size || 0), 0);
});

function rowsFor(entry) {
  const rows = [
    ['Name', entry.name],
    ['Location', entry.dir],
    ['Type', describeKind(entry)],
    ['Size', entry.isDir ? '--' : `${formatSize(entry.size)} (${entry.size.toLocaleString()} bytes)`],
    ['Modified', formatDate(entry.modTime, 'full')],
    ['Accessed', formatDate(entry.accessTime, 'full')],
    ['Permissions', `${entry.perm} (${entry.mode})`],
    ['Owner', `${entry.owner}:${entry.group} (${entry.uid}:${entry.gid})`],
    ['Links', String(entry.nlink)],
  ];
  if (entry.isSymlink) rows.push(['Target', entry.target || '--']);
  if (entry.mime) rows.push(['MIME type', entry.mime]);
  return rows;
}

function runChecksum() {
  if (single.value) computeChecksum(single.value.path, checksumAlgo.value);
}
</script>

<template>
  <ModalShell
    v-if="props"
    :title="props.entries.length > 1 ? pluralise(props.entries.length, 'item') : (single?.name || 'Properties')"
    :subtitle="props.entries.length > 1 ? 'Multiple selection' : single?.dir"
    icon="info"
    wide
    @close="closeProperties"
  >
    <div v-if="props.loading" class="prop-loading faint">
      <Icon name="loader" :size="16" class="spin" /> Gathering details…
    </div>

    <template v-else>
      <div class="prop-summary panel">
        <div class="prop-stat">
          <span class="prop-stat-value">{{ props.count?.files?.toLocaleString() ?? 0 }}</span>
          <span class="prop-stat-label">Files</span>
        </div>
        <div class="prop-stat">
          <span class="prop-stat-value">{{ props.count?.dirs?.toLocaleString() ?? 0 }}</span>
          <span class="prop-stat-label">Folders</span>
        </div>
        <div class="prop-stat">
          <span class="prop-stat-value">{{ formatSize(totalSize) }}</span>
          <span class="prop-stat-label">Total size</span>
        </div>
        <div v-if="props.count?.truncated" class="prop-stat is-warn">
          <span class="prop-stat-value"><Icon name="alert-triangle" :size="14" /></span>
          <span class="prop-stat-label">Count truncated</span>
        </div>
      </div>

      <dl v-if="single" class="prop-table">
        <template v-for="row in rowsFor(single)" :key="row[0]">
          <dt>{{ row[0] }}</dt>
          <dd :title="String(row[1])">{{ row[1] }}</dd>
        </template>
      </dl>

      <div v-if="props.usage" class="prop-usage">
        <div class="prop-usage-head">
          <span>Disk usage on {{ props.usage.fsType || 'this filesystem' }}</span>
          <span class="tnum">{{ formatSize(props.usage.free) }} free of {{ formatSize(props.usage.total) }}</span>
        </div>
        <div class="progress">
          <span :style="{ width: `${Math.min(100, props.usage.percent || 0)}%` }" />
        </div>
      </div>

      <div v-if="single && !single.isDir" class="prop-checksum">
        <div class="prop-checksum-head">
          <span>Checksum</span>
          <div class="row" style="gap: var(--s-2)">
            <select v-model="checksumAlgo" class="select" style="width: 110px; height: 28px">
              <option value="sha256">SHA-256</option>
              <option value="sha1">SHA-1</option>
              <option value="md5">MD5</option>
            </select>
            <button class="btn btn-sm btn-outline" type="button" @click="runChecksum">Compute</button>
          </div>
        </div>
        <code v-if="props.checksum" class="prop-hash">{{ props.checksum }}</code>
      </div>

      <p v-if="props.count?.truncated" class="field-hint">
        This folder is very large, so the totals above are estimates.
      </p>
    </template>

    <template #footer>
      <button class="btn btn-primary" type="button" @click="closeProperties">Close</button>
    </template>
  </ModalShell>
</template>

<style scoped>
.prop-loading { display: flex; align-items: center; gap: var(--s-2); padding: var(--s-4) 0; }

.prop-summary {
  display: flex;
  gap: var(--s-2);
  padding: var(--s-3);
  margin-bottom: var(--s-4);
}
.prop-stat {
  flex: 1 1 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.prop-stat-value { font-size: var(--fs-lg); font-weight: 650; letter-spacing: -0.01em; }
.prop-stat-label { font-size: var(--fs-xs); color: var(--text-faint); }
.prop-stat.is-warn { color: var(--warning); }

.prop-table {
  display: grid;
  grid-template-columns: 104px minmax(0, 1fr);
  gap: 7px var(--s-3);
  margin: 0 0 var(--s-4);
  font-size: var(--fs-sm);
}
.prop-table dt { color: var(--text-faint); }
.prop-table dd { margin: 0; user-select: text; word-break: break-all; }

.prop-usage { margin-bottom: var(--s-4); }
.prop-usage-head {
  display: flex;
  justify-content: space-between;
  gap: var(--s-3);
  margin-bottom: 6px;
  font-size: var(--fs-xs);
  color: var(--text-muted);
}

.prop-checksum { display: flex; flex-direction: column; gap: var(--s-2); }
.prop-checksum-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--s-3);
  font-size: var(--fs-sm);
  color: var(--text-muted);
}
.prop-hash {
  display: block;
  padding: var(--s-2) var(--s-3);
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  word-break: break-all;
  user-select: text;
}

.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>

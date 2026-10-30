<script setup>
import { computed } from 'vue';
import Icon from './Icon.vue';
import {
  state, cancelOperation, clearFinishedOperations, toggleOpsPanel,
} from '../store/app.js';
import { formatSize, formatDuration } from '../lib/format.js';

const ops = computed(() => state.ops.slice().reverse());
const active = computed(() => ops.value.filter((op) => op.status === 'running' || op.status === 'queued'));
const finished = computed(() => ops.value.filter((op) => op.status !== 'running' && op.status !== 'queued'));

const STATUS_ICONS = {
  queued: 'clock',
  running: 'loader',
  done: 'check-circle',
  cancelled: 'x-circle',
  error: 'alert-circle',
};

function statusIcon(status) {
  return STATUS_ICONS[status] || 'info';
}

function speedLabel(op) {
  if (!op.speed) return '';
  return `${formatSize(Math.round(op.speed))}/s`;
}

function etaLabel(op) {
  if (op.status !== 'running') return '';
  if (op.etaSeconds === undefined || op.etaSeconds < 0) return '';
  return `${formatDuration(op.etaSeconds)} left`;
}
</script>

<template>
  <aside v-if="state.opsPanel.open" class="ops-panel" aria-label="Operation queue">
    <header class="ops-head">
      <Icon name="activity" :size="15" />
      <h2 class="ops-title">Operations</h2>
      <span v-if="active.length" class="chip accent">{{ active.length }} running</span>
      <div class="grow" />
      <button
        class="icon-btn is-sm"
        type="button"
        title="Clear finished"
        :disabled="!finished.length"
        @click="clearFinishedOperations"
      >
        <Icon name="x-circle" :size="14" />
      </button>
      <button class="icon-btn is-sm" type="button" title="Close" @click="toggleOpsPanel(false)">
        <Icon name="x" :size="14" />
      </button>
    </header>

    <div class="ops-body">
      <p v-if="!ops.length" class="ops-empty faint">
        Nothing has run yet. Copying, moving, extracting and sizing folders appear here.
      </p>

      <article v-for="op in ops" :key="op.id" class="op-card" :class="`is-${op.status}`">
        <div class="op-row">
          <Icon
            :name="statusIcon(op.status)"
            :size="15"
            class="op-status"
            :class="{ spin: op.status === 'running' }"
          />
          <span class="op-name truncate" :title="op.title">{{ op.title || op.kind }}</span>
          <span class="op-percent tnum">{{ Math.round(op.percent || 0) }}%</span>
          <button
            v-if="op.status === 'running' || op.status === 'queued'"
            class="icon-btn is-sm"
            type="button"
            title="Cancel"
            @click="cancelOperation(op.id)"
          >
            <Icon name="x" :size="13" />
          </button>
        </div>

        <div v-if="op.status === 'running' || op.status === 'queued'" class="progress">
          <span :style="{ width: `${Math.min(100, op.percent || 0)}%` }" />
        </div>

        <div class="op-meta faint">
          <span v-if="op.status === 'running'" class="truncate" :title="op.current">{{ op.current }}</span>
          <span v-else-if="op.message" class="truncate">{{ op.message }}</span>
          <span v-if="op.totalItems" class="tnum">{{ op.currentItem }}/{{ op.totalItems }}</span>
          <span v-if="speedLabel(op)">{{ speedLabel(op) }}</span>
          <span v-if="etaLabel(op)">{{ etaLabel(op) }}</span>
        </div>

        <ul v-if="op.errors && op.errors.length" class="op-errors">
          <li v-for="(error, index) in op.errors.slice(0, 6)" :key="index">{{ error }}</li>
          <li v-if="op.errors.length > 6" class="faint">…and {{ op.errors.length - 6 }} more</li>
        </ul>
      </article>
    </div>
  </aside>
</template>

<style scoped>
.ops-panel {
  position: fixed;
  right: var(--s-3);
  bottom: calc(var(--statusbar-h) + var(--s-2));
  z-index: 320;
  width: 380px;
  max-height: min(60vh, 520px);
  display: flex;
  flex-direction: column;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-lg);
  box-shadow: var(--shadow-3);
  overflow: hidden;
  animation: ops-in var(--t-base) cubic-bezier(0.32, 0.72, 0, 1);
}

@keyframes ops-in {
  from { opacity: 0; transform: translateY(8px) scale(0.99); }
  to { opacity: 1; transform: none; }
}

.ops-head {
  display: flex;
  align-items: center;
  gap: var(--s-2);
  padding: var(--s-2) var(--s-2) var(--s-2) var(--s-3);
  border-bottom: 1px solid var(--border);
  color: var(--text-muted);
}
.ops-title { font-size: var(--fs-md); font-weight: 600; color: var(--text); }

.ops-body { overflow-y: auto; padding: var(--s-2); display: flex; flex-direction: column; gap: var(--s-2); }
.ops-empty { padding: var(--s-4) var(--s-2); font-size: var(--fs-sm); line-height: 1.5; }

.op-card {
  padding: var(--s-2) var(--s-3);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  background: var(--surface);
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.op-card.is-error { border-color: var(--danger); }

.op-row { display: flex; align-items: center; gap: var(--s-2); }
.op-status { color: var(--text-faint); flex: 0 0 auto; }
.op-card.is-running .op-status { color: var(--accent); }
.op-card.is-done .op-status { color: var(--success); }
.op-card.is-error .op-status { color: var(--danger); }
.op-card.is-cancelled .op-status { color: var(--warning); }
.op-name { flex: 1 1 auto; font-size: var(--fs-sm); }
.op-percent { font-size: var(--fs-xs); color: var(--text-muted); }

.op-meta {
  display: flex;
  align-items: center;
  gap: var(--s-2);
  font-size: var(--fs-xs);
  min-width: 0;
}
.op-meta > span:first-child { flex: 1 1 auto; min-width: 0; }

.op-errors {
  margin: 0;
  padding-left: var(--s-4);
  font-size: var(--fs-xs);
  color: var(--danger);
  max-height: 90px;
  overflow-y: auto;
}

.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>

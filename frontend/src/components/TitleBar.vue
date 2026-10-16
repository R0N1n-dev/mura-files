<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import Icon from './Icon.vue';
import { AppService } from '../lib/fs.js';
import { state, activeTab, openDialog } from '../store/app.js';

const maximised = ref(false);
let poll = null;

async function syncWindowState() {
  try {
    const info = await AppService.WindowState();
    maximised.value = !!info?.maximised || !!info?.fullscreen;
  } catch {
    /* the window may not exist yet during the first paint */
  }
}

async function toggleMaximise() {
  await AppService.ToggleMaximise();
  setTimeout(syncWindowState, 90);
}

function minimise() { AppService.Minimise().catch(() => {}); }
function close() { AppService.CloseWindow().catch(() => {}); }
function maximise() { toggleMaximise(); }

/** Double clicking the title bar toggles maximise, like every other window. */
function onDoubleClick(event) {
  if (event.target.closest('button, .no-drag')) return;
  toggleMaximise();
}

onMounted(() => {
  syncWindowState();
  poll = setInterval(syncWindowState, 1500);
});

onUnmounted(() => {
  if (poll) clearInterval(poll);
});
</script>

<template>
  <header class="titlebar drag-region" @dblclick="onDoubleClick">
    <div class="brand no-drag">
      <span class="brand-mark" aria-hidden="true">
        <svg viewBox="0 0 24 24" width="17" height="17">
          <path d="M4 20a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h4l2 2.5h8a2 2 0 0 1 2 2V18a2 2 0 0 1-2 2z"
                fill="currentColor" opacity="0.92" />
          <path d="M4 20a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h4l2 2.5h8a2 2 0 0 1 2 2V18a2 2 0 0 1-2 2z"
                fill="none" stroke="currentColor" stroke-width="1.4" opacity="0.5" />
        </svg>
      </span>
      <span class="brand-name">Mura</span>
    </div>

    <div class="title-text truncate">
      <span class="truncate">{{ activeTab?.title || 'Files' }}</span>
    </div>

    <div class="spacer drag-region" />

    <nav class="window-controls no-drag">
      <button class="wc" type="button" title="Minimise" @click="minimise">
        <Icon name="minimise" :size="15" />
      </button>
      <button
        class="wc"
        type="button"
        :title="maximised ? 'Restore' : 'Maximise'"
        @click="maximise"
      >
        <Icon :name="maximised ? 'restore-window' : 'maximise'" :size="13" />
      </button>
      <button class="wc wc-close" type="button" title="Close" @click="close">
        <Icon name="x" :size="15" />
      </button>
    </nav>
  </header>
</template>

<style scoped>
.titlebar {
  display: flex;
  align-items: center;
  gap: var(--s-3);
  height: var(--titlebar-h);
  padding: 0 var(--s-2) 0 var(--s-4);
  background: var(--surface);
  background-image: var(--titlebar-tint);
  border-bottom: 1px solid var(--border);
  flex: 0 0 auto;
  position: relative;
  z-index: 20;
}

.brand {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--accent);
  flex: 0 0 auto;
}
.brand-mark { display: flex; }
.brand-name {
  font-size: var(--fs-md);
  font-weight: 650;
  letter-spacing: -0.01em;
  color: var(--text);
}

.title-text {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  justify-content: center;
  font-size: var(--fs-sm);
  font-weight: 500;
  color: var(--text-muted);
  pointer-events: none;
}

.spacer { flex: 0 0 96px; }

.window-controls { display: flex; align-items: center; gap: 2px; }

.wc {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 26px;
  border-radius: var(--r-sm);
  color: var(--text-muted);
  transition: background var(--t-fast), color var(--t-fast);
}
.wc:hover { background: var(--hover); color: var(--text); }
.wc:active { background: var(--active); }
.wc-close:hover { background: var(--danger); color: #fff; }
</style>

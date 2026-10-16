<script setup>
import { ref, computed } from 'vue';
import ModalShell from './ModalShell.vue';
import Icon from './Icon.vue';
import { state, closeAppPicker, launchApp, openInTerminal } from '../store/app.js';
import { iconUrl } from '../lib/fs.js';

const filter = ref('');

const apps = computed(() => {
  const list = state.dialogs.appPicker?.apps || [];
  const needle = filter.value.trim().toLowerCase();
  if (!needle) return list;
  return list.filter((app) => app.name.toLowerCase().includes(needle));
});

const entry = computed(() => state.dialogs.appPicker?.entry || null);
</script>

<template>
  <ModalShell
    v-if="state.dialogs.appPicker"
    title="Open With"
    :subtitle="entry?.name"
    icon="sparkles"
    @close="closeAppPicker"
  >
    <div class="field">
      <input
        v-model="filter"
        class="input"
        placeholder="Filter applications…"
        spellcheck="false"
        data-autofocus
      />
    </div>

    <div class="app-list">
      <p v-if="state.dialogs.appPicker.loading" class="faint app-note">
        <Icon name="loader" :size="15" class="spin" /> Loading applications…
      </p>
      <p v-else-if="!apps.length" class="faint app-note">No matching applications.</p>
      <button
        v-for="app in apps"
        :key="app.path"
        class="app-item"
        type="button"
        @click="launchApp(app, entry)"
      >
        <img v-if="app.icon" :src="iconUrl(app.icon)" :alt="app.name" class="app-icon" draggable="false" />
        <Icon v-else name="package" :size="18" class="app-icon" />
        <span class="truncate">{{ app.name }}</span>
      </button>
    </div>

    <template #footer>
      <button class="btn btn-ghost grow" type="button" style="justify-content: flex-start" @click="openInTerminal(entry?.isDir ? entry.path : entry?.dir); closeAppPicker()">
        <Icon name="terminal" :size="14" /> Open in Terminal
      </button>
      <button class="btn btn-ghost" type="button" @click="closeAppPicker">Cancel</button>
    </template>
  </ModalShell>
</template>

<style scoped>
.app-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: var(--s-3);
  max-height: 46vh;
  overflow-y: auto;
}
.app-note { display: flex; align-items: center; gap: var(--s-2); padding: var(--s-3) 0; font-size: var(--fs-sm); }
.app-item {
  display: flex;
  align-items: center;
  gap: var(--s-3);
  height: 38px;
  padding: 0 var(--s-2);
  border-radius: var(--r-sm);
  color: var(--text);
  font-size: var(--fs-md);
  text-align: left;
  transition: background var(--t-fast);
}
.app-item:hover { background: var(--hover); }
.app-icon { width: 20px; height: 20px; object-fit: contain; color: var(--text-muted); flex: 0 0 auto; }
.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>

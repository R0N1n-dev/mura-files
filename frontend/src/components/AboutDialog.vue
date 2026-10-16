<script setup>
import ModalShell from './ModalShell.vue';
import Icon from './Icon.vue';
import { state, closeDialog, openDialog } from '../store/app.js';
</script>

<template>
  <ModalShell v-if="state.dialogs.about" icon="" @close="closeDialog('about')">
    <div class="about">
      <div class="about-mark">
        <svg viewBox="0 0 24 24" width="40" height="40">
          <path d="M4 20a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h4l2 2.5h8a2 2 0 0 1 2 2V18a2 2 0 0 1-2 2z"
                fill="currentColor" opacity="0.95" />
        </svg>
      </div>
      <h2 class="about-title">Mura Files</h2>
      <p class="about-version faint">Version {{ state.info?.version || '0.1.0' }}</p>
      <p class="about-text muted">
        A fast, elegant file manager for Linux, built with Go and Vue.
        Everything runs locally: no telemetry, no cloud, no accounts.
      </p>

      <dl class="about-table">
        <dt>System</dt><dd>{{ state.info?.os?.name }}</dd>
        <dt>Kernel</dt><dd>{{ state.info?.os?.kernel }} · {{ state.info?.os?.arch }}</dd>
        <dt>Desktop</dt><dd>{{ state.info?.os?.desktop || '—' }}</dd>
        <dt>Shell</dt><dd>{{ state.info?.os?.shell || '—' }}</dd>
        <dt>Terminal</dt><dd>{{ state.info?.terminal || 'not detected' }}</dd>
      </dl>

      <div class="about-actions">
        <button class="btn btn-outline" type="button" @click="closeDialog('about'); openDialog('shortcuts')">
          <Icon name="keyboard" :size="14" /> Shortcuts
        </button>
        <button class="btn btn-outline" type="button" @click="closeDialog('about'); openDialog('settings')">
          <Icon name="settings" :size="14" /> Settings
        </button>
      </div>
    </div>

    <template #footer>
      <button class="btn btn-primary" type="button" @click="closeDialog('about')">Close</button>
    </template>
  </ModalShell>
</template>

<style scoped>
.about { display: flex; flex-direction: column; align-items: center; text-align: center; gap: var(--s-2); padding-top: var(--s-2); }
.about-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72px;
  height: 72px;
  border-radius: var(--r-xl);
  background: var(--accent-soft);
  color: var(--accent);
  margin-bottom: var(--s-2);
}
.about-title { font-size: var(--fs-2xl); letter-spacing: -0.02em; }
.about-version { font-size: var(--fs-sm); }
.about-text { font-size: var(--fs-md); line-height: 1.55; max-width: 380px; }
.about-table {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 5px var(--s-4);
  margin: var(--s-4) 0 0;
  font-size: var(--fs-sm);
  text-align: left;
  width: 100%;
  max-width: 400px;
}
.about-table dt { color: var(--text-faint); }
.about-table dd { margin: 0; user-select: text; }
.about-actions { display: flex; gap: var(--s-2); margin-top: var(--s-5); }
</style>

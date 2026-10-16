<script setup>
import { onMounted } from 'vue';
import Icon from './components/Icon.vue';
import AppShell from './components/AppShell.vue';
import { state, bootstrap } from './store/app.js';

onMounted(() => {
  bootstrap();
});
</script>

<template>
  <AppShell v-if="state.ready && !state.bootError" />

  <div v-else-if="state.bootError" class="boot-error">
    <Icon name="alert-triangle" :size="34" />
    <h1>Mura Files could not start</h1>
    <p class="muted">{{ state.bootError }}</p>
    <p class="faint">Try restarting the application. If the problem persists, check the terminal output for details.</p>
  </div>

  <div v-else class="boot-splash">
    <div class="splash-mark">
      <svg viewBox="0 0 24 24" width="34" height="34">
        <path d="M4 20a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h4l2 2.5h8a2 2 0 0 1 2 2V18a2 2 0 0 1-2 2z" fill="currentColor" />
      </svg>
    </div>
    <span class="splash-name">Mura Files</span>
    <span class="splash-spinner" />
  </div>
</template>

<style scoped>
.boot-splash,
.boot-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--s-3);
  height: 100vh;
  text-align: center;
  padding: var(--s-6);
}

.splash-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 66px;
  height: 66px;
  border-radius: var(--r-xl);
  background: var(--accent-soft);
  color: var(--accent);
  animation: breathe 2.4s ease-in-out infinite;
}
@keyframes breathe {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.04); opacity: 0.86; }
}

.splash-name {
  font-size: var(--fs-xl);
  font-weight: 650;
  letter-spacing: -0.02em;
}

.splash-spinner {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid var(--border-strong);
  border-top-color: var(--accent);
  animation: spin 800ms linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.boot-error { color: var(--danger); gap: var(--s-2); }
.boot-error h1 { font-size: var(--fs-xl); color: var(--text); }
.boot-error p { max-width: 460px; font-size: var(--fs-md); }
</style>

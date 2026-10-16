<script setup>
import { ref, watch, nextTick } from 'vue';
import ModalShell from './ModalShell.vue';
import { state, closePrompt } from '../store/app.js';

const value = ref('');
const input = ref(null);

watch(() => state.dialogs.prompt, async (config) => {
  if (!config) return;
  value.value = config.value || '';
  await nextTick();
  input.value?.focus();
  if (config.selectStem) {
    const dot = value.value.lastIndexOf('.');
    if (dot > 0) input.value?.setSelectionRange(0, dot);
    else input.value?.select();
  } else {
    input.value?.select();
  }
}, { immediate: true });

function confirm() {
  const config = state.dialogs.prompt;
  const trimmed = value.value.trim();
  if (!config || !trimmed) return;
  closePrompt();
  config.onConfirm?.(trimmed);
}
</script>

<template>
  <ModalShell
    v-if="state.dialogs.prompt"
    :title="state.dialogs.prompt.title"
    :subtitle="state.dialogs.prompt.subtitle"
    icon="pencil"
    @close="closePrompt"
  >
    <div class="field">
      <label for="prompt-input">{{ state.dialogs.prompt.label }}</label>
      <input
        id="prompt-input"
        ref="input"
        v-model="value"
        class="input"
        spellcheck="false"
        autocomplete="off"
        @keydown.enter.prevent="confirm"
      />
      <p v-if="state.dialogs.prompt.hint" class="field-hint">{{ state.dialogs.prompt.hint }}</p>
    </div>

    <template #footer>
      <button class="btn btn-ghost" type="button" @click="closePrompt">Cancel</button>
      <button class="btn btn-primary" type="button" :disabled="!value.trim()" @click="confirm">
        {{ state.dialogs.prompt.confirmLabel }}
      </button>
    </template>
  </ModalShell>
</template>

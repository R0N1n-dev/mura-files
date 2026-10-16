<script setup>
import ModalShell from './ModalShell.vue';
import Icon from './Icon.vue';
import { state, closeConfirm } from '../store/app.js';

function confirm() {
  const config = state.dialogs.confirm;
  closeConfirm();
  config?.onConfirm?.();
}
</script>

<template>
  <ModalShell
    v-if="state.dialogs.confirm"
    :title="state.dialogs.confirm.title"
    :icon="state.dialogs.confirm.danger ? 'alert-triangle' : 'help'"
    @close="closeConfirm"
  >
    <p class="confirm-message">{{ state.dialogs.confirm.message }}</p>
    <p v-if="state.dialogs.confirm.detail" class="confirm-detail muted">
      <Icon name="info" :size="13" />
      <span>{{ state.dialogs.confirm.detail }}</span>
    </p>

    <template #footer>
      <button class="btn btn-ghost" type="button" data-autofocus @click="closeConfirm">
        {{ state.dialogs.confirm.cancelLabel }}
      </button>
      <button
        class="btn"
        :class="state.dialogs.confirm.danger ? 'btn-danger' : 'btn-primary'"
        type="button"
        @click="confirm"
      >
        {{ state.dialogs.confirm.confirmLabel }}
      </button>
    </template>
  </ModalShell>
</template>

<style scoped>
.confirm-message { font-size: var(--fs-lg); line-height: 1.5; }
.confirm-detail {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin-top: var(--s-3);
  font-size: var(--fs-sm);
}
</style>

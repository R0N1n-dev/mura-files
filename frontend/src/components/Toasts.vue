<script setup>
import Icon from './Icon.vue';
import { state, dismissToast } from '../store/app.js';

const ICONS = {
  info: 'info',
  success: 'check-circle',
  warning: 'alert-triangle',
  error: 'alert-circle',
};
</script>

<template>
  <Teleport to="body">
    <div class="toast-stack" aria-live="polite">
      <TransitionGroup name="pop">
        <div
          v-for="item in state.toasts"
          :key="item.id"
          class="toast"
          :class="`is-${item.kind}`"
        >
          <Icon :name="ICONS[item.kind] || 'info'" :size="17" class="toast-icon" />
          <div class="grow">
            <div v-if="item.title" class="toast-title">{{ item.title }}</div>
            <div class="toast-text">{{ item.message }}</div>
          </div>
          <button class="icon-btn is-sm" type="button" title="Dismiss" @click="dismissToast(item.id)">
            <Icon name="x" :size="13" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

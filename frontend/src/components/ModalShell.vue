<script setup>
import { onMounted, onUnmounted, ref } from 'vue';
import Icon from './Icon.vue';

const props = defineProps({
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
  icon: { type: String, default: '' },
  wide: { type: Boolean, default: false },
  tall: { type: Boolean, default: false },
  closeOnBackdrop: { type: Boolean, default: true },
});

const emit = defineEmits(['close']);
const panel = ref(null);

function onBackdrop(event) {
  if (!props.closeOnBackdrop) return;
  if (event.target === event.currentTarget) emit('close');
}

function onKeydown(event) {
  if (event.key === 'Escape') {
    event.stopPropagation();
    emit('close');
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown, true);
  // Focus the first sensible control so the dialog is usable immediately.
  requestAnimationFrame(() => {
    const target = panel.value?.querySelector('[data-autofocus]') ||
      panel.value?.querySelector('input, textarea, select, button');
    target?.focus?.();
  });
});

onUnmounted(() => window.removeEventListener('keydown', onKeydown, true));
</script>

<template>
  <Teleport to="body">
    <div class="modal-backdrop" @mousedown="onBackdrop">
      <div
        ref="panel"
        class="modal"
        :class="{ 'is-wide': wide, 'is-tall': tall }"
        role="dialog"
        aria-modal="true"
      >
        <header v-if="title" class="modal-head">
          <span v-if="icon" class="modal-icon"><Icon :name="icon" :size="18" /></span>
          <div class="grow">
            <h2 class="modal-title">{{ title }}</h2>
            <p v-if="subtitle" class="modal-sub">{{ subtitle }}</p>
          </div>
          <button class="icon-btn" type="button" title="Close" @click="emit('close')">
            <Icon name="x" :size="16" />
          </button>
        </header>

        <div class="modal-body"><slot /></div>

        <footer v-if="$slots.footer" class="modal-foot"><slot name="footer" /></footer>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.modal-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: var(--r-md);
  background: var(--accent-soft);
  color: var(--accent);
  flex: 0 0 auto;
}
</style>

<script setup>
import { computed, onMounted, onUnmounted } from 'vue';
import Icon from './Icon.vue';
import { state, closeContextMenu } from '../store/app.js';

const menu = computed(() => state.contextMenu);

function run(item) {
  if (item.disabled || item.separator) return;
  closeContextMenu();
  item.run?.();
}

function onGlobalMouseDown(event) {
  if (!menu.value.open) return;
  if (event.target.closest('.menu')) return;
  closeContextMenu();
}

function onKeydown(event) {
  if (event.key === 'Escape') closeContextMenu();
}

onMounted(() => {
  window.addEventListener('mousedown', onGlobalMouseDown, true);
  window.addEventListener('keydown', onKeydown);
  window.addEventListener('blur', closeContextMenu);
  window.addEventListener('resize', closeContextMenu);
});

onUnmounted(() => {
  window.removeEventListener('mousedown', onGlobalMouseDown, true);
  window.removeEventListener('keydown', onKeydown);
  window.removeEventListener('blur', closeContextMenu);
  window.removeEventListener('resize', closeContextMenu);
});
</script>

<template>
  <Teleport to="body">
    <div
      v-if="menu.open"
      class="menu context-menu"
      role="menu"
      :style="{ left: `${menu.x}px`, top: `${menu.y}px` }"
      @contextmenu.prevent
    >
      <template v-for="(item, index) in menu.items" :key="index">
        <div v-if="item.separator" class="menu-sep" />
        <button
          v-else
          class="menu-item"
          :class="{ 'is-danger': item.danger }"
          type="button"
          role="menuitem"
          :disabled="item.disabled"
          @click="run(item)"
        >
          <Icon v-if="item.icon" :name="item.icon" :size="15" class="menu-icon" />
          <span v-else class="menu-icon-spacer" />
          <span class="truncate">{{ item.label }}</span>
          <span v-if="item.checked" class="menu-check"><Icon name="check" :size="13" :stroke="2.4" /></span>
          <span v-else-if="item.shortcut" class="menu-shortcut">{{ item.shortcut }}</span>
        </button>
      </template>
    </div>
  </Teleport>
</template>

<style scoped>
.context-menu {
  position: fixed;
  z-index: 600;
  max-height: min(70vh, 520px);
  overflow-y: auto;
  animation: menu-in var(--t-fast) ease-out;
}

@keyframes menu-in {
  from { opacity: 0; transform: translateY(-3px) scale(0.99); }
  to { opacity: 1; transform: none; }
}

.menu-icon-spacer { width: 15px; flex: 0 0 auto; }
.menu-check { margin-left: auto; color: var(--accent); }
</style>

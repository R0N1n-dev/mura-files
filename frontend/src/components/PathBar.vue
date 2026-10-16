<script setup>
import { ref, computed, watch, nextTick } from 'vue';
import Icon from './Icon.vue';
import { FileService, attempt } from '../lib/fs.js';
import { activeTab, navigate } from '../store/app.js';

const editing = ref(false);
const draft = ref('');
const completions = ref([]);
const completionIndex = ref(-1);
const input = ref(null);

/** Split the current path into clickable breadcrumb segments. */
const crumbs = computed(() => {
  const path = activeTab.value?.path || '';
  if (!path) return [];
  const parts = path.split('/').filter(Boolean);
  const items = [{ name: '/', path: '/' }];
  let current = '';
  parts.forEach((part) => {
    current += `/${part}`;
    items.push({ name: part, path: current });
  });
  return items;
});

/** Show the tail of long paths so the deepest folders stay visible. */
const visibleCrumbs = computed(() => {
  const items = crumbs.value;
  if (items.length <= 5) return items;
  return [items[0], { ellipsis: true }, ...items.slice(-3)];
});

async function beginEdit() {
  draft.value = activeTab.value?.path || '';
  editing.value = true;
  completions.value = [];
  completionIndex.value = -1;
  await nextTick();
  input.value?.focus();
  input.value?.select();
}

function cancelEdit() {
  editing.value = false;
  completions.value = [];
  completionIndex.value = -1;
}

async function commit() {
  const target = draft.value.trim();
  cancelEdit();
  if (!target) return;
  const tab = activeTab.value;
  if (!tab) return;
  const resolved = await FileService.Resolve(target);
  await navigate(tab, resolved || target);
}

let completionToken = 0;
async function refreshCompletions() {
  const token = ++completionToken;
  const value = draft.value;
  if (!value) {
    completions.value = [];
    return;
  }
  const { value: list } = await attempt(FileService.Complete(value, 12));
  if (token !== completionToken) return;
  completions.value = list || [];
  completionIndex.value = completions.value.length ? 0 : -1;
}

function applyCompletion(item) {
  if (!item) return;
  draft.value = item.path;
  completions.value = [];
  completionIndex.value = -1;
  if (!item.isDir) commit();
  else nextTick(() => input.value?.focus());
}

function onKeydown(event) {
  if (event.key === 'Enter') {
    event.preventDefault();
    if (completionIndex.value >= 0 && completions.value[completionIndex.value]) {
      const item = completions.value[completionIndex.value];
      if (item.isDir && draft.value !== item.path) {
        applyCompletion(item);
        return;
      }
    }
    commit();
    return;
  }
  if (event.key === 'Escape') {
    event.preventDefault();
    cancelEdit();
    return;
  }
  if (event.key === 'Tab' && completions.value.length) {
    event.preventDefault();
    applyCompletion(completions.value[Math.max(0, completionIndex.value)]);
    return;
  }
  if (event.key === 'ArrowDown' && completions.value.length) {
    event.preventDefault();
    completionIndex.value = Math.min(completionIndex.value + 1, completions.value.length - 1);
    return;
  }
  if (event.key === 'ArrowUp' && completions.value.length) {
    event.preventDefault();
    completionIndex.value = Math.max(completionIndex.value - 1, 0);
  }
}

function goTo(path) {
  const tab = activeTab.value;
  if (tab) navigate(tab, path);
}

// Leave edit mode whenever the folder changes underneath us.
watch(() => activeTab.value?.path, () => cancelEdit());

defineExpose({ beginEdit });
</script>

<template>
  <div class="pathbar">
    <div v-if="!editing" class="crumbs" @dblclick="beginEdit" @click="beginEdit">
      <template v-for="(crumb, index) in visibleCrumbs" :key="index">
        <span v-if="crumb.ellipsis" class="crumb-ellipsis" title="Show full path">
          <Icon name="more-horizontal" :size="14" />
        </span>
        <button
          v-else
          class="crumb"
          type="button"
          :class="{ 'is-last': index === visibleCrumbs.length - 1 }"
          @click.stop="goTo(crumb.path)"
        >
          <Icon v-if="index === 0" name="hard-drive" :size="13" class="crumb-icon" />
          <span class="truncate">{{ index === 0 ? 'root' : crumb.name }}</span>
        </button>
        <Icon
          v-if="index < visibleCrumbs.length - 1 && !visibleCrumbs[index + 1]?.ellipsis"
          name="chevron-right"
          :size="12"
          class="crumb-sep"
        />
      </template>
    </div>

    <div v-else class="edit-wrap">
      <Icon name="terminal" :size="13" class="edit-icon" />
      <input
        ref="input"
        v-model="draft"
        class="input mono path-input"
        spellcheck="false"
        autocomplete="off"
        placeholder="Enter a path…"
        @input="refreshCompletions"
        @keydown="onKeydown"
        @blur="setTimeout(cancelEdit, 130)"
      />
      <div v-if="completions.length" class="completions menu">
        <button
          v-for="(item, index) in completions"
          :key="item.path"
          class="menu-item"
          :class="{ 'is-active': index === completionIndex }"
          type="button"
          @mousedown.prevent="applyCompletion(item)"
          @mouseenter="completionIndex = index"
        >
          <Icon :name="item.isDir ? 'folder' : 'file'" :size="14" class="menu-icon" />
          <span class="truncate">{{ item.name }}</span>
          <span v-if="item.isLink" class="chip">link</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.pathbar { flex: 1 1 auto; min-width: 0; position: relative; }

.crumbs {
  display: flex;
  align-items: center;
  gap: 1px;
  height: 32px;
  padding: 0 4px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  overflow: hidden;
  cursor: text;
}

.crumb {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  max-width: 180px;
  height: 24px;
  padding: 0 8px;
  border-radius: 5px;
  color: var(--text-muted);
  font-size: var(--fs-md);
  transition: background var(--t-fast), color var(--t-fast);
}
.crumb:hover { background: var(--hover); color: var(--text); }
.crumb.is-last { color: var(--text); font-weight: 550; }
.crumb-icon { color: var(--text-faint); }
.crumb-sep { color: var(--text-faint); opacity: 0.6; flex: 0 0 auto; }
.crumb-ellipsis {
  display: inline-flex;
  align-items: center;
  padding: 0 4px;
  color: var(--text-faint);
}

.edit-wrap { position: relative; display: flex; align-items: center; }
.edit-icon {
  position: absolute;
  left: 10px;
  color: var(--text-faint);
  pointer-events: none;
  z-index: 1;
}
.path-input { height: 32px; padding-left: 32px; }

.completions {
  position: absolute;
  top: calc(100% + 5px);
  left: 0;
  right: 0;
  z-index: 60;
  max-height: 300px;
  overflow: auto;
}
.completions .menu-item.is-active { background: var(--hover); }
</style>

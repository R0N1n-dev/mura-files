<script setup>
import ModalShell from './ModalShell.vue';
import { state, closeDialog } from '../store/app.js';

const groups = [
  {
    title: 'Navigation',
    items: [
      ['Alt + Left', 'Go back'],
      ['Alt + Right', 'Go forward'],
      ['Alt + Up', 'Go to parent folder'],
      ['Home', 'Jump to the first item'],
      ['End', 'Jump to the last item'],
      ['Ctrl + L', 'Edit the location bar'],
      ['Ctrl + R / F5', 'Refresh the current folder'],
      ['Ctrl + T', 'New tab'],
      ['Ctrl + W', 'Close tab'],
      ['Ctrl + Tab', 'Next tab'],
      ['Ctrl + 1..3', 'List / Grid / Compact view'],
    ],
  },
  {
    title: 'Selection',
    items: [
      ['Click', 'Select an item'],
      ['Ctrl + Click', 'Add or remove from the selection'],
      ['Shift + Click', 'Select a range'],
      ['Ctrl + A', 'Select everything'],
      ['Ctrl + Shift + A', 'Invert the selection'],
      ['Drag', 'Rubber-band select'],
    ],
  },
  {
    title: 'Files',
    items: [
      ['Enter', 'Open the selection'],
      ['F2', 'Rename'],
      ['Ctrl + C / Ctrl + X', 'Copy / cut'],
      ['Ctrl + V', 'Paste'],
      ['Ctrl + D', 'Duplicate'],
      ['Delete', 'Move to Trash'],
      ['Shift + Delete', 'Delete permanently'],
      ['Ctrl + Shift + N', 'New folder'],
      ['Ctrl + Alt + N', 'New file'],
      ['Alt + Enter', 'Properties'],
    ],
  },
  {
    title: 'View and search',
    items: [
      ['Ctrl + F', 'Search the current folder'],
      ['Ctrl + P', 'Command palette'],
      ['Ctrl + H', 'Toggle hidden files'],
      ['F11 / Space', 'Toggle the preview pane'],
      ['Ctrl + Shift + F', 'Toggle fullscreen'],
      ['Ctrl + ,', 'Settings'],
      ['Ctrl + /', 'This list'],
      ['F12', 'Developer tools'],
      ['Esc', 'Close menus and dialogs'],
    ],
  },
];
</script>

<template>
  <ModalShell
    v-if="state.dialogs.shortcuts"
    title="Keyboard shortcuts"
    subtitle="Everything you need without reaching for the mouse."
    icon="keyboard"
    wide
    tall
    @close="closeDialog('shortcuts')"
  >
    <div class="shortcut-grid">
      <section v-for="group in groups" :key="group.title" class="shortcut-group">
        <h3 class="group-title">{{ group.title }}</h3>
        <div v-for="[keys, label] in group.items" :key="keys" class="shortcut-row">
          <span class="shortcut-label">{{ label }}</span>
          <span class="shortcut-keys">
            <template v-for="(key, index) in keys.split(' + ')" :key="key">
              <span v-if="index > 0" class="shortcut-plus">+</span>
              <kbd class="kbd">{{ key }}</kbd>
            </template>
          </span>
        </div>
      </section>
    </div>

    <template #footer>
      <button class="btn btn-primary" type="button" @click="closeDialog('shortcuts')">Close</button>
    </template>
  </ModalShell>
</template>

<style scoped>
.shortcut-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--s-5) var(--s-6);
}
.shortcut-group { display: flex; flex-direction: column; gap: 5px; }
.group-title {
  margin-bottom: 4px;
  font-size: 10.5px;
  font-weight: 650;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: var(--text-faint);
}
.shortcut-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--s-3);
  min-height: 26px;
}
.shortcut-label { font-size: var(--fs-sm); color: var(--text-muted); }
.shortcut-keys { display: inline-flex; align-items: center; gap: 3px; flex: 0 0 auto; }
.shortcut-plus { color: var(--text-faint); font-size: 10px; }
</style>

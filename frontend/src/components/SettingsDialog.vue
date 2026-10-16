<script setup>
import { ref, computed } from 'vue';
import ModalShell from './ModalShell.vue';
import Icon from './Icon.vue';
import {
  state, closeDialog, updateSettings, notify, refreshTheme,
} from '../store/app.js';
import { AppService } from '../lib/fs.js';
import { formatSize } from '../lib/format.js';

const section = ref('appearance');
const cache = ref(null);

const sections = [
  { key: 'appearance', label: 'Appearance', icon: 'palette' },
  { key: 'behaviour', label: 'Behaviour', icon: 'sliders' },
  { key: 'browsing', label: 'Browsing', icon: 'folder' },
  { key: 'integrations', label: 'Integrations', icon: 'terminal' },
  { key: 'about', label: 'About & Data', icon: 'info' },
];

const themes = [
  { key: 'system', label: 'System', icon: 'monitor' },
  { key: 'dark', label: 'Dark', icon: 'moon' },
  { key: 'light', label: 'Light', icon: 'sun' },
];

const accents = ['', 'indigo', 'violet', 'teal', 'rose', 'amber', 'emerald', 'sky', 'slate'];

const terminals = computed(() => state.info?.terminal || '');

function set(key, value) {
  updateSettings({ [key]: value }, true);
}

function toggle(key) {
  set(key, !state.settings[key]);
}

async function loadCacheStats() {
  try {
    cache.value = await AppService.CacheStats();
  } catch {
    cache.value = null;
  }
}

async function clearCache() {
  try {
    await AppService.ClearThumbnailCache();
    await loadCacheStats();
    notify('Thumbnail cache cleared', 'success');
  } catch (error) {
    notify(String(error?.message || error), 'error');
  }
}

function accentStyle(accent) {
  const map = {
    '': '#7c8cff', indigo: '#7c8cff', violet: '#a78bfa', teal: '#2dd4bf',
    rose: '#fb7185', amber: '#fbbf24', emerald: '#34d399', sky: '#38bdf8',
    slate: '#94a3b8',
  };
  return { background: map[accent] || '#7c8cff' };
}

loadCacheStats();
</script>

<template>
  <ModalShell
    v-if="state.dialogs.settings"
    title="Settings"
    subtitle="Tune Mura Files to match the way you work."
    icon="settings"
    wide
    tall
    @close="closeDialog('settings')"
  >
    <div class="settings">
      <nav class="settings-nav">
        <button
          v-for="item in sections"
          :key="item.key"
          class="nav-item"
          :class="{ 'is-active': section === item.key }"
          type="button"
          @click="section = item.key"
        >
          <Icon :name="item.icon" :size="15" />
          <span>{{ item.label }}</span>
        </button>
      </nav>

      <div class="settings-body">
        <!-- Appearance -->
        <template v-if="section === 'appearance'">
          <div class="group">
            <h3 class="group-title">Theme</h3>
            <div class="theme-row">
              <button
                v-for="theme in themes"
                :key="theme.key"
                class="theme-card"
                :class="{ 'is-active': state.settings.theme === theme.key }"
                type="button"
                @click="set('theme', theme.key)"
              >
                <Icon :name="theme.icon" :size="18" />
                <span>{{ theme.label }}</span>
              </button>
            </div>
          </div>

          <div class="group">
            <h3 class="group-title">Accent colour</h3>
            <div class="accent-row">
              <button
                v-for="accent in accents"
                :key="accent || 'default'"
                class="accent-dot"
                :class="{ 'is-active': state.settings.accentColour === accent }"
                type="button"
                :title="accent || 'Theme default'"
                :style="accentStyle(accent)"
                @click="set('accentColour', accent)"
              />
            </div>
          </div>

          <div class="group">
            <h3 class="group-title">Layout</h3>
            <label class="row-item">
              <span class="grow">Show sidebar</span>
              <button class="switch" :class="{ 'is-on': state.settings.showSidebar }" type="button" @click="toggle('showSidebar')" />
            </label>
            <label class="row-item">
              <span class="grow">Show status bar</span>
              <button class="switch" :class="{ 'is-on': state.settings.showStatusBar }" type="button" @click="toggle('showStatusBar')" />
            </label>
            <label class="row-item">
              <span class="grow">Show preview pane by default</span>
              <button class="switch" :class="{ 'is-on': state.settings.showPreview }" type="button" @click="toggle('showPreview')" />
            </label>
            <label class="row-item">
              <span class="grow">Thumbnails in grid view</span>
              <button class="switch" :class="{ 'is-on': state.settings.showThumbnails }" type="button" @click="toggle('showThumbnails')" />
            </label>
            <div class="row-item">
              <span class="grow">Grid tile size</span>
              <input
                class="range"
                type="range"
                min="80"
                max="200"
                step="8"
                :value="state.settings.gridSize"
                @input="set('gridSize', Number($event.target.value))"
              />
              <span class="tnum faint">{{ state.settings.gridSize }}px</span>
            </div>
          </div>
        </template>

        <!-- Behaviour -->
        <template v-else-if="section === 'behaviour'">
          <div class="group">
            <h3 class="group-title">Deleting</h3>
            <label class="row-item">
              <span class="grow">Move files to the Trash by default<small>Turn off to delete permanently, with no undo.</small></span>
              <button class="switch" :class="{ 'is-on': state.settings.useTrash }" type="button" @click="toggle('useTrash')" />
            </label>
            <label class="row-item">
              <span class="grow">Ask before deleting</span>
              <button class="switch" :class="{ 'is-on': state.settings.confirmDelete }" type="button" @click="toggle('confirmDelete')" />
            </label>
            <label class="row-item">
              <span class="grow">Ask before moving to the Trash</span>
              <button class="switch" :class="{ 'is-on': state.settings.confirmTrash }" type="button" @click="toggle('confirmTrash')" />
            </label>
          </div>

          <div class="group">
            <h3 class="group-title">Opening</h3>
            <label class="row-item">
              <span class="grow">Open items with a single click</span>
              <button class="switch" :class="{ 'is-on': state.settings.singleClickOpen }" type="button" @click="toggle('singleClickOpen')" />
            </label>
            <label class="row-item">
              <span class="grow">Confirm before closing multiple tabs</span>
              <button class="switch" :class="{ 'is-on': state.settings.confirmCloseTabs }" type="button" @click="toggle('confirmCloseTabs')" />
            </label>
          </div>

          <div class="group">
            <h3 class="group-title">Session</h3>
            <label class="row-item">
              <span class="grow">Reopen the tabs from last time<small>Restores the folders you had open when you quit.</small></span>
              <button class="switch" :class="{ 'is-on': state.settings.restoreTabs }" type="button" @click="toggle('restoreTabs')" />
            </label>
          </div>
        </template>

        <!-- Browsing -->
        <template v-else-if="section === 'browsing'">
          <div class="group">
            <h3 class="group-title">Listing</h3>
            <label class="row-item">
              <span class="grow">Show hidden files<small>Files and folders whose name starts with a dot.</small></span>
              <button class="switch" :class="{ 'is-on': state.settings.showHidden }" type="button" @click="toggle('showHidden')" />
            </label>
            <label class="row-item">
              <span class="grow">Show file extensions</span>
              <button class="switch" :class="{ 'is-on': state.settings.showExtensions }" type="button" @click="toggle('showExtensions')" />
            </label>
            <label class="row-item">
              <span class="grow">Keep folders above files</span>
              <button class="switch" :class="{ 'is-on': state.settings.dirsFirst }" type="button" @click="toggle('dirsFirst')" />
            </label>
            <label class="row-item">
              <span class="grow">Watch folders for live changes<small>Uses inotify to refresh the view when files change on disk.</small></span>
              <button class="switch" :class="{ 'is-on': state.settings.watchEnabled }" type="button" @click="toggle('watchEnabled')" />
            </label>
          </div>

          <div class="group">
            <h3 class="group-title">Formatting</h3>
            <div class="row-item">
              <span class="grow">Date format</span>
              <select class="select" style="width: 150px" :value="state.settings.dateFormat" @change="set('dateFormat', $event.target.value)">
                <option value="auto">Smart</option>
                <option value="relative">Relative</option>
                <option value="iso">ISO</option>
                <option value="long">Long</option>
                <option value="full">Full</option>
              </select>
            </div>
            <div class="row-item">
              <span class="grow">Size units</span>
              <select class="select" style="width: 150px" :value="state.settings.sizeFormat" @change="set('sizeFormat', $event.target.value)">
                <option value="binary">Binary (KiB, MiB)</option>
                <option value="decimal">Decimal (kB, MB)</option>
              </select>
            </div>
          </div>
        </template>

        <!-- Integrations -->
        <template v-else-if="section === 'integrations'">
          <div class="group">
            <h3 class="group-title">Terminal</h3>
            <p class="group-hint">
              Detected: <strong>{{ terminals || 'none found' }}</strong>
            </p>
            <div class="field">
              <label for="terminal-cmd">Custom command</label>
              <input
                id="terminal-cmd"
                class="input mono"
                :value="state.settings.terminalCommand"
                placeholder="e.g. alacritty --working-directory %d"
                spellcheck="false"
                @change="set('terminalCommand', $event.target.value)"
              />
              <p class="field-hint">%d is replaced with the folder being browsed.</p>
            </div>
            <div class="field">
              <label for="editor-cmd">Editor command</label>
              <input
                id="editor-cmd"
                class="input mono"
                :value="state.settings.editorCommand"
                placeholder="e.g. code %f"
                spellcheck="false"
                @change="set('editorCommand', $event.target.value)"
              />
              <p class="field-hint">%f is replaced with the selected file.</p>
            </div>
          </div>

          <div class="group">
            <h3 class="group-title">Available helpers</h3>
            <div class="tools">
              <span
                v-for="tool in state.info?.tools || []"
                :key="tool.name"
                class="chip"
                :class="tool.available ? 'success' : ''"
                :title="tool.path || 'Not installed'"
              >
                {{ tool.name }}
              </span>
            </div>
          </div>
        </template>

        <!-- About & data -->
        <template v-else>
          <div class="group">
            <h3 class="group-title">System</h3>
            <dl class="data-table">
              <dt>Distribution</dt><dd>{{ state.info?.os?.name }}</dd>
              <dt>Kernel</dt><dd>{{ state.info?.os?.kernel }}</dd>
              <dt>Architecture</dt><dd>{{ state.info?.os?.arch }}</dd>
              <dt>Desktop</dt><dd>{{ state.info?.os?.desktop || '—' }}</dd>
              <dt>Session</dt><dd>{{ state.info?.os?.session || '—' }}</dd>
              <dt>Host</dt><dd>{{ state.info?.os?.hostname }}</dd>
              <dt>User</dt><dd>{{ state.info?.os?.user }}</dd>
            </dl>
          </div>

          <div class="group">
            <h3 class="group-title">Thumbnail cache</h3>
            <div class="row-item">
              <span class="grow">
                {{ cache ? `${cache.count} thumbnails · ${formatSize(cache.bytes)}` : 'Reading cache…' }}
              </span>
              <button class="btn btn-sm btn-outline" type="button" @click="clearCache">Clear</button>
              <button class="btn btn-sm btn-ghost" type="button" @click="loadCacheStats">Refresh</button>
            </div>
          </div>

          <div class="group">
            <h3 class="group-title">Theme source</h3>
            <p class="group-hint">
              The desktop reports a
              <strong>{{ state.theme.dark ? 'dark' : 'light' }}</strong> preference
              (source: {{ state.theme.source }}).
              <button class="btn btn-sm btn-ghost" type="button" @click="refreshTheme">Re-check</button>
            </p>
          </div>
        </template>
      </div>
    </div>

    <template #footer>
      <span class="grow faint" style="font-size: var(--fs-xs)">Settings are saved automatically.</span>
      <button class="btn btn-primary" type="button" @click="closeDialog('settings')">Done</button>
    </template>
  </ModalShell>
</template>

<style scoped>
.settings { display: flex; gap: var(--s-5); min-height: 380px; }

.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 172px;
  flex: 0 0 auto;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 9px;
  height: 32px;
  padding: 0 var(--s-2);
  border-radius: var(--r-sm);
  color: var(--text-muted);
  font-size: var(--fs-md);
  text-align: left;
  transition: background var(--t-fast), color var(--t-fast);
}
.nav-item:hover { background: var(--hover); color: var(--text); }
.nav-item.is-active { background: var(--accent-soft); color: var(--accent); font-weight: 550; }

.settings-body { flex: 1 1 auto; min-width: 0; display: flex; flex-direction: column; gap: var(--s-5); }
.group { display: flex; flex-direction: column; gap: var(--s-2); }
.group-title {
  font-size: 10.5px;
  font-weight: 650;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: var(--text-faint);
}
.group-hint { font-size: var(--fs-sm); color: var(--text-muted); }

.row-item {
  display: flex;
  align-items: center;
  gap: var(--s-3);
  min-height: 32px;
  font-size: var(--fs-md);
}
.row-item small { display: block; font-size: var(--fs-xs); color: var(--text-faint); margin-top: 2px; }

.theme-row { display: flex; gap: var(--s-2); }
.theme-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  width: 82px;
  padding: var(--s-3) 0;
  border: 1px solid var(--border-strong);
  border-radius: var(--r-md);
  color: var(--text-muted);
  font-size: var(--fs-sm);
  transition: border-color var(--t-fast), color var(--t-fast), background var(--t-fast);
}
.theme-card:hover { background: var(--hover); color: var(--text); }
.theme-card.is-active { border-color: var(--accent); color: var(--accent); background: var(--accent-softer); }

.accent-row { display: flex; gap: var(--s-2); }
.accent-dot {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: 2px solid transparent;
  box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.18);
  transition: transform var(--t-fast), border-color var(--t-fast);
}
.accent-dot:hover { transform: scale(1.08); }
.accent-dot.is-active { border-color: var(--text); }

.range { flex: 1 1 auto; max-width: 200px; accent-color: var(--accent); }

.tools { display: flex; flex-wrap: wrap; gap: 5px; }

.data-table {
  display: grid;
  grid-template-columns: 116px minmax(0, 1fr);
  gap: 6px var(--s-3);
  margin: 0;
  font-size: var(--fs-sm);
}
.data-table dt { color: var(--text-faint); }
.data-table dd { margin: 0; user-select: text; }
</style>

/**
 * Application entry point.
 *
 * The Wails runtime is imported for its side effects (it installs the window
 * drag handling, context menu hooks and file drop plumbing) before the Vue
 * application mounts.
 */

import { createApp } from 'vue';
import '@wailsio/runtime';

import './styles/tokens.css';
import './styles/base.css';
import './styles/components.css';

import App from './App.vue';

createApp(App).mount('#app');

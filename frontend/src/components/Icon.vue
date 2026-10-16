<script setup>
import { computed } from 'vue';
import { ICONS, FILE_ICONS, PLACE_ICONS, FILLED } from '../lib/icons.js';

const props = defineProps({
  /** Icon key from the icon set. */
  name: { type: String, required: true },
  /** Rendered size in pixels. */
  size: { type: [Number, String], default: 16 },
  /** Stroke width, kept thin for a refined look. */
  stroke: { type: [Number, String], default: 1.7 },
  /** Optional explicit colour. */
  color: { type: String, default: '' },
});

const markup = computed(() =>
  ICONS[props.name] || FILE_ICONS[props.name] || PLACE_ICONS[props.name] || FILE_ICONS.file);

const filled = computed(() => FILLED.has(props.name));

const style = computed(() => {
  const size = typeof props.size === 'number' ? `${props.size}px` : props.size;
  return {
    width: size,
    height: size,
    ...(props.color ? { color: props.color } : {}),
  };
});
</script>

<template>
  <svg
    class="icon"
    :style="style"
    viewBox="0 0 24 24"
    :fill="filled ? 'currentColor' : 'none'"
    :stroke="filled ? 'none' : 'currentColor'"
    :stroke-width="stroke"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    v-html="markup"
  />
</template>

<style scoped>
.icon {
  display: block;
  flex: 0 0 auto;
  /* Keep icons crisp at fractional device pixel ratios. */
  shape-rendering: geometricPrecision;
}
</style>

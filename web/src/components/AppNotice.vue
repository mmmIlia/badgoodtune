<script setup lang="ts">
withDefaults(defineProps<{ kind?: 'error' | 'info'; message: string }>(), { kind: 'info' })
</script>

<template>
  <div class="notice" :class="`notice-${kind}`" role="status" aria-live="polite">{{ message }}</div>
</template>

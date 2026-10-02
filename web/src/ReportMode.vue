<script setup lang="ts">
const boot = JSON.parse(document.getElementById('boot-config')?.textContent || '{}')
</script>
<template>
  <p v-if="boot.mode === 'offline'" class="report-mode">Archived report · For local serving, use <code>teleskope serve snapshot &lt;report-directory&gt;</code></p>
</template>
<style>
.report-mode { margin: 12px 24px; color: var(--muted); font-size: 12px; }
</style>

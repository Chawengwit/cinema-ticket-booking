<script setup lang="ts">
import { ref, onMounted } from "vue";

const health = ref<any>(null);
const error = ref<string | null>(null);
const loading = ref(false);

onMounted(async () => {
  loading.value = true;
  try {
    const res = await fetch("/api/health");
    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`);
    }

    health.value = await res.json();
  } catch (e: any) {
    error.value = e?.message ?? "Failed to fetch /health.";
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div class="card space-y-3">
    <div class="flex items-center justify-between">
      <p class="section-title">Health Check</p>
      <span class="pill" :class="error ? 'text-[var(--color-error)] border border-[var(--color-error)]/40' : 'text-[var(--color-success)] border border-[var(--color-success)]/40'">
        {{ error ? "Degraded" : "Healthy" }}
      </span>
    </div>
    <p class="section-subtitle">SPA → API → Mongo + Redis connectivity</p>

    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-else-if="loading" class="alert alert-info">Checking...</div>
    <pre
      v-else
      class="rounded-2xl bg-slate-900/70 p-4 text-xs text-slate-100 ring-1 ring-white/5 overflow-auto max-h-72"
    >
{{ JSON.stringify(health, null, 2) }}</pre>
  </div>
</template>

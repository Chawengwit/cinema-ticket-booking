<template>
  <header class="sticky top-0 z-50 border-b border-white/5 bg-slate-900/70 backdrop-blur-xl">
    <nav class="container-wide flex items-center justify-between py-3">
      <div class="flex items-center gap-3">
        <span class="flex h-10 w-10 items-center justify-center rounded-2xl bg-[var(--color-primary)]/20 text-lg font-bold text-[var(--color-primary)] ring-1 ring-[var(--color-primary)]/30">
          CT
        </span>
        <div>
          <p class="text-sm font-semibold text-white">Cinema Ticket</p>
          <p class="text-xs text-slate-400">Real-time seat locking</p>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <div v-if="isAuthed" class="hidden sm:block text-right">
          <p class="text-sm font-semibold text-white flex items-center gap-2 justify-end">
            {{ displayName }}
            <span class="pill" :class="rolePillClass" title="Role">{{ roleText }}</span>
          </p>
          <p class="text-xs text-slate-400">{{ displayEmail }}</p>
        </div>

        <span v-if="loading" class="hidden sm:inline text-xs text-slate-400">Loading...</span>

        <button v-if="!isAuthed" @click="doLogin" class="btn btn-primary">
          Sign in with Google
        </button>
        <button v-else @click="doLogout" class="btn btn-secondary">
          Logout
        </button>
      </div>
    </nav>
  </header>
</template>

<script setup lang="ts">
import { computed } from "vue";

type Role = "ADMIN" | "USER" | string;

const props = defineProps<{
  user: any | null;
  role: Role;
  isAuthed: boolean;
  loading?: boolean;
}>();

const emit = defineEmits<{
  (e: "login"): void;
  (e: "logout"): void;
}>();

const displayName = computed(() => props.user?.name ?? "User");
const displayEmail = computed(() => props.user?.email ?? "");

const roleText = computed(() => props.role ?? "USER");
const rolePillClass = computed(() =>
  roleText.value === "ADMIN"
    ? "text-amber-300 border border-amber-300/40"
    : "text-emerald-300 border border-emerald-300/40"
);

function doLogin() {
  emit("login");
}

function doLogout() {
  emit("logout");
}
</script>

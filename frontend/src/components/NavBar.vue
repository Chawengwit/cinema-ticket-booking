<template>
  <header class="sticky top-0 z-50 border-b bg-white/80 backdrop-blur">
    <nav class="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
      <div class="font-bold">Cinema Ticket</div>

      <div class="flex items-center gap-3">
        <div v-if="isAuthed" class="hidden sm:block text-right">
          <p class="text-sm font-medium text-gray-900">
            {{ displayName }}
            <span
              class="ml-2 rounded-full border px-2 py-0.5 text-xs font-semibold"
              :class="rolePillClass"
              title="Role"
            >
              {{ roleText }}
            </span>
          </p>
          <p class="text-xs text-gray-500">{{ displayEmail }}</p>
        </div>

        <span v-if="loading" class="hidden sm:inline text-xs text-gray-500">Loading...</span>

        <!-- Login -->
        <button
          v-if="!isAuthed"
          @click="doLogin"
          class="inline-flex items-center rounded-xl bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-800 active:scale-[0.99]"
        >
          Sign in with Google
        </button>

        <!-- Logout -->
        <button
          v-else
          @click="doLogout"
          class="inline-flex items-center rounded-xl border border-gray-300 bg-white px-4 py-2 text-sm font-semibold text-gray-900 hover:bg-gray-50 active:scale-[0.99]"
        >
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
    ? "border-amber-300 bg-amber-50 text-amber-700"
    : "border-emerald-300 bg-emerald-50 text-emerald-700"
);

function doLogin() {
  emit("login");
}

function doLogout() {
  emit("logout");
}
</script>

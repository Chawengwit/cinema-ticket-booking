<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import NavBar from "./components/NavBar.vue";
import HealthCard from "./components/HealthCard.vue";
import AdminDashboard from "./components/AdminDashboard.vue";
import BookingFlow from "./components/BookingFlow.vue";

const AUTH_KEY = "access_token";
const API_ORIGIN = import.meta.env.VITE_API_ORIGIN || "http://localhost:8080";

type MeResponse = {
  ok: boolean;
  role?: string;
  user?: {
    id?: string;
    email?: string;
    name?: string;
    picture?: string;
    role?: string;
  };
  error?: string;
};

const user = ref<MeResponse["user"] | null>(null);
const role = ref<string>("USER");
const loadingMe = ref(false);
const error = ref<string | null>(null);

const view = ref<"home" | "admin">("home");

const isAuthed = computed(() => !!localStorage.getItem(AUTH_KEY));
const isAdmin = computed(() => (role.value || "").toUpperCase() === "ADMIN");

function startLogin() {
  window.location.href = `${API_ORIGIN}/api/auth/google/login`;
}

function logout() {
  localStorage.removeItem(AUTH_KEY);
  user.value = null;
  role.value = "USER";
  error.value = null;
  view.value = "home";
}

async function fetchMe() {
  error.value = null;

  const token = localStorage.getItem(AUTH_KEY);
  if (!token) return;

  loadingMe.value = true;
  try {
    const res = await fetch(`${API_ORIGIN}/api/me`, {
      headers: { Authorization: `Bearer ${token}` },
    });

    if (res.status === 401) {
      logout();
      return;
    }

    const data: MeResponse = await res.json();
    if (!res.ok || data.ok !== true) {
      throw new Error(data?.error || "Failed to fetch /api/me");
    }

    user.value = data.user ?? null;
    role.value = data.role ?? data.user?.role ?? "USER";

    if (!isAdmin.value && view.value === "admin") {
      view.value = "home";
    }
  } catch (e: any) {
    error.value = e?.message ?? "Failed to fetch /api/me";
  } finally {
    loadingMe.value = false;
  }
}

onMounted(() => {
  fetchMe();
});
</script>

<template>
  <div class="page-shell">
    <NavBar
      :user="user"
      :role="role"
      :isAuthed="isAuthed"
      :loading="loadingMe"
      @login="startLogin"
      @logout="logout"
    />

    <main class="container-wide py-8 space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-white">Cinema Ticket Booking</h1>
          <p class="text-sm text-slate-400">Movie → Seats → Payment → Completed</p>
        </div>
        <div class="flex gap-2">
          <button class="btn" :class="view === 'home' ? 'btn-primary' : 'btn-ghost'" @click="view = 'home'">
            Home
          </button>
          <button v-if="isAdmin" class="btn" :class="view === 'admin' ? 'btn-primary' : 'btn-ghost'" @click="view = 'admin'">
            Admin
          </button>
        </div>
      </div>

      <template v-if="view === 'home'">
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div class="card col-span-2 space-y-4">
            <div class="flex items-center justify-between gap-3">
              <div>
                <p class="section-title">Booking Flow</p>
                <p class="section-subtitle">Pick a movie, choose seats, mock pay, and complete booking.</p>
              </div>
              <div class="flex items-center gap-2">
                <button v-if="!isAuthed" class="btn btn-primary" @click="startLogin">Sign in with Google</button>
                <button v-else class="btn btn-secondary" @click="logout">Logout</button>
              </div>
            </div>

            <div v-if="error" class="alert alert-error">{{ error }}</div>

            <BookingFlow :apiOrigin="API_ORIGIN" :authKey="AUTH_KEY" :isAuthed="isAuthed" />
          </div>

          <div class="card space-y-3">
            <p class="section-title">Environment</p>
            <div class="grid grid-cols-2 gap-3 text-sm">
              <div class="card-muted space-y-1">
                <p class="text-xs uppercase text-slate-400">API Origin</p>
                <p class="font-semibold text-white break-words">{{ API_ORIGIN }}</p>
              </div>
              <div class="card-muted space-y-1">
                <p class="text-xs uppercase text-slate-400">Auth</p>
                <p class="font-semibold text-white">{{ isAuthed ? "Authenticated" : "Guest" }}</p>
              </div>
            </div>
            <div class="alert alert-info text-xs">
              Tip: Admin dashboard is visible only for accounts listed in <span class="font-mono">ADMIN_EMAILS</span>.
            </div>
            <HealthCard />
          </div>
        </div>
      </template>

      <template v-else>
        <AdminDashboard />
      </template>
    </main>
  </div>
</template>

<!-- frontend/src/App.vue -->
<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import NavBar from "./components/NavBar.vue";
import SeatEventsCard from "./components/SeatEventsCard.vue";
import HealthCard from "./components/HealthCard.vue";
import AdminDashboard from "./components/AdminDashboard.vue";

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

    // ถ้าไม่ใช่ admin แต่ดันอยู่หน้า admin -> กลับ home
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
  <div class="min-h-screen bg-gray-50">
    <NavBar
      :user="user"
      :role="role"
      :isAuthed="isAuthed"
      :loading="loadingMe"
      @login="startLogin"
      @logout="logout"
    />

    <main class="mx-auto max-w-6xl p-6 space-y-6">
      <!-- Tabs -->
      <div class="flex gap-2">
        <button
          class="rounded-xl px-4 py-2 text-sm font-semibold"
          :class="view === 'home' ? 'bg-black text-white' : 'bg-white border'"
          @click="view = 'home'"
        >
          Home
        </button>

        <button
          v-if="isAdmin"
          class="rounded-xl px-4 py-2 text-sm font-semibold"
          :class="view === 'admin' ? 'bg-black text-white' : 'bg-white border'"
          @click="view = 'admin'"
        >
          Admin
        </button>
      </div>

      <!-- HOME -->
      <template v-if="view === 'home'">
        <div class="rounded-2xl bg-white p-6 shadow">
          <h1 class="text-2xl font-bold">Cinema Ticket Booking</h1>

          <p v-if="!isAuthed" class="mt-3 text-gray-600">
            Not signed in. Click <b>Sign in with Google</b>.
          </p>

          <div v-else class="mt-4">
            <div v-if="user" class="rounded-xl bg-green-50 p-4">
              <span class="font-semibold text-green-700">Logged in user: </span>

              <span class="text-sm font-semibold text-gray-900 ml-6">
                {{ user.name || "User" }}
                <span class="ml-2 text-xs text-gray-600">({{ role }})</span>
              </span>
            </div>

            <p v-else class="text-gray-600">Signed in. Loading profile...</p>
          </div>

          <p v-if="error" class="mt-3 text-sm text-red-600">{{ error }}</p>
        </div>

        <SeatEventsCard />
        <HealthCard />
      </template>

      <!-- ADMIN -->
      <template v-else>
        <AdminDashboard />
      </template>
    </main>
  </div>
</template>

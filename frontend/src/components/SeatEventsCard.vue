<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from "vue";

const AUTH_KEY = "access_token";
const API_ORIGIN = import.meta.env.VITE_API_ORIGIN || "http://localhost:8080";

const showtimeId = ref("demo-001");
const connected = ref(false);
const logs = ref<string[]>([]);
let ws: WebSocket | null = null;

function wsBase(url: string) {
  // http:// -> ws:// , https:// -> wss://
  return url.replace(/^http/, "ws");
}

function connect() {
  const token = localStorage.getItem(AUTH_KEY);
  if (!token) {
    logs.value.unshift("[error] no token, please login first");
    return;
  }
  if (ws) ws.close();

  const url = `${wsBase(API_ORIGIN)}/ws/showtimes/${encodeURIComponent(showtimeId.value)}/seats?token=${encodeURIComponent(token)}`;
  ws = new WebSocket(url);

  ws.onopen = () => {
    connected.value = true;
    logs.value.unshift(`[ws] connected: ${url}`);
  };
  ws.onclose = () => {
    connected.value = false;
    logs.value.unshift("[ws] disconnected");
  };
  ws.onerror = () => {
    logs.value.unshift("[ws] error");
  };
  ws.onmessage = (ev) => {
    logs.value.unshift(ev.data);
  };
}

function disconnect() {
  ws?.close();
  ws = null;
}

async function lockDemo() {
  const token = localStorage.getItem(AUTH_KEY);
  if (!token) return;

  const res = await fetch(`${API_ORIGIN}/api/showtimes/${showtimeId.value}/seats/lock`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ seat_ids: ["A1", "A2"] }),
  });
  logs.value.unshift(`[lock] status=${res.status} body=${await res.text()}`);
}

async function releaseDemo() {
  const token = localStorage.getItem(AUTH_KEY);
  if (!token) return;

  const res = await fetch(`${API_ORIGIN}/api/showtimes/${showtimeId.value}/seats/lock`, {
    method: "DELETE",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ seat_ids: ["A1", "A2"] }),
  });
  logs.value.unshift(`[release] status=${res.status} body=${await res.text()}`);
}

onMounted(() => {});
onBeforeUnmount(() => disconnect());
</script>

<template>
  <div class="rounded-2xl bg-white p-6 shadow space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-bold">Seat Events (WebSocket)</h2>
      <span class="text-sm" :class="connected ? 'text-green-600' : 'text-gray-500'">
        {{ connected ? "CONNECTED" : "DISCONNECTED" }}
      </span>
    </div>

    <div class="flex flex-col gap-3 md:flex-row md:items-center">
      <label class="text-sm font-medium">Showtime ID</label>
      <input v-model="showtimeId" class="w-full md:w-64 rounded-lg border px-3 py-2 text-sm" />
      <button @click="connect" class="rounded-lg bg-black px-4 py-2 text-sm text-white">Connect</button>
      <button @click="disconnect" class="rounded-lg border px-4 py-2 text-sm">Disconnect</button>
      <button @click="lockDemo" class="rounded-lg bg-emerald-600 px-4 py-2 text-sm text-white">Lock A1,A2</button>
      <button @click="releaseDemo" class="rounded-lg bg-rose-600 px-4 py-2 text-sm text-white">Release A1,A2</button>
    </div>

    <div class="rounded-xl bg-gray-50 p-3 h-64 overflow-auto text-xs font-mono">
      <div v-for="(l, idx) in logs" :key="idx" class="border-b py-1 border-gray-200">
        {{ l }}
      </div>
    </div>
  </div>
</template>

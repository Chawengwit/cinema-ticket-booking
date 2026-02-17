<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

const AUTH_KEY = "access_token";
const API_ORIGIN = import.meta.env.VITE_API_ORIGIN || "http://localhost:8080";

type BookingItem = any;
type AuditItem = any;

const tab = ref<"bookings" | "audit">("bookings");

// ---------------- helpers ----------------
function authHeaders() {
  const token = localStorage.getItem(AUTH_KEY);
  return token ? { Authorization: `Bearer ${token}` } : {};
}

function toISOStart(dateStr: string) {
  if (!dateStr) return "";
  const d = new Date(dateStr + "T00:00:00");
  return d.toISOString();
}

function toISOEnd(dateStr: string) {
  if (!dateStr) return "";
  const d = new Date(dateStr + "T23:59:59.999");
  return d.toISOString();
}

// ---------------- bookings ----------------
const b_showtime = ref("");
const b_status = ref("");
const b_user = ref("");
const b_from = ref("");
const b_to = ref("");
const b_limit = ref(20);
const b_skip = ref(0);

const bookings = ref<BookingItem[]>([]);
const bookingsTotal = ref(0);
const bookingsLoading = ref(false);
const bookingsError = ref<string | null>(null);

const bookingsPage = computed(() => Math.floor(b_skip.value / b_limit.value) + 1);
const bookingsHasPrev = computed(() => b_skip.value > 0);
const bookingsHasNext = computed(() => b_skip.value + b_limit.value < bookingsTotal.value);

async function loadBookings() {
  bookingsLoading.value = true;
  bookingsError.value = null;

  try {
    const qs = new URLSearchParams();
    if (b_showtime.value) qs.set("showtime_id", b_showtime.value);
    if (b_status.value) qs.set("status", b_status.value);
    if (b_user.value) qs.set("user_id", b_user.value);
    if (b_from.value) qs.set("from", toISOStart(b_from.value));
    if (b_to.value) qs.set("to", toISOEnd(b_to.value));
    qs.set("limit", String(b_limit.value));
    qs.set("skip", String(b_skip.value));

    const res = await fetch(`${API_ORIGIN}/api/admin/bookings?${qs.toString()}`, {
      headers: { ...authHeaders() },
    });

    const data = await res.json().catch(() => ({}));
    if (!res.ok || !data?.ok) {
      throw new Error(data?.error || `HTTP_${res.status}`);
    }

    bookings.value = data.items ?? [];
    bookingsTotal.value = data.total ?? 0;
  } catch (e: any) {
    bookingsError.value = e?.message ?? "Failed to load bookings";
  } finally {
    bookingsLoading.value = false;
  }
}

function bookingsApplyFilters() {
  b_skip.value = 0;
  loadBookings();
}

// ---------------- audit ----------------
const a_type = ref("");
const a_showtime = ref("");
const a_user = ref("");
const a_booking = ref("");
const a_from = ref("");
const a_to = ref("");
const a_limit = ref(20);
const a_skip = ref(0);

const audit = ref<AuditItem[]>([]);
const auditTotal = ref(0);
const auditLoading = ref(false);
const auditError = ref<string | null>(null);

const auditPage = computed(() => Math.floor(a_skip.value / a_limit.value) + 1);
const auditHasPrev = computed(() => a_skip.value > 0);
const auditHasNext = computed(() => a_skip.value + a_limit.value < auditTotal.value);

async function loadAudit() {
  auditLoading.value = true;
  auditError.value = null;

  try {
    const qs = new URLSearchParams();
    if (a_type.value) qs.set("type", a_type.value);
    if (a_showtime.value) qs.set("showtime_id", a_showtime.value);
    if (a_user.value) qs.set("user_id", a_user.value);
    if (a_booking.value) qs.set("booking_id", a_booking.value);
    if (a_from.value) qs.set("from", toISOStart(a_from.value));
    if (a_to.value) qs.set("to", toISOEnd(a_to.value));
    qs.set("limit", String(a_limit.value));
    qs.set("skip", String(a_skip.value));

    const res = await fetch(`${API_ORIGIN}/api/admin/audit?${qs.toString()}`, {
      headers: { ...authHeaders() },
    });

    const data = await res.json().catch(() => ({}));
    if (!res.ok || !data?.ok) {
      throw new Error(data?.error || `HTTP_${res.status}`);
    }

    audit.value = data.items ?? [];
    auditTotal.value = data.total ?? 0;
  } catch (e: any) {
    auditError.value = e?.message ?? "Failed to load audit logs";
  } finally {
    auditLoading.value = false;
  }
}

function auditApplyFilters() {
  a_skip.value = 0;
  loadAudit();
}

function fmtSeats(it: any) {
  const arr = it?.seat_ids || it?.seats || it?.seatIds || [];
  return Array.isArray(arr) ? arr.join(", ") : "-";
}

function fmtTime(it: any) {
  return it?.created_at || it?.createdAt || it?.ts || it?.time || "-";
}

onMounted(() => {
  loadBookings();
});
</script>

<template>
  <div class="rounded-2xl bg-white p-6 shadow space-y-4">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h2 class="text-xl font-bold">Admin Dashboard</h2>
        <p class="text-sm text-gray-600">Bookings + Audit Logs (filters + pagination)</p>
      </div>

      <div class="flex gap-2">
        <button
          class="rounded-xl px-4 py-2 text-sm font-semibold"
          :class="tab === 'bookings' ? 'bg-black text-white' : 'bg-gray-100 text-gray-900'"
          @click="tab = 'bookings'; loadBookings()"
        >
          Bookings
        </button>
        <button
          class="rounded-xl px-4 py-2 text-sm font-semibold"
          :class="tab === 'audit' ? 'bg-black text-white' : 'bg-gray-100 text-gray-900'"
          @click="tab = 'audit'; loadAudit()"
        >
          Audit
        </button>
      </div>
    </div>

    <!-- BOOKINGS TAB -->
    <div v-if="tab === 'bookings'" class="space-y-4">
      <div class="grid grid-cols-1 gap-3 md:grid-cols-6">
        <input v-model="b_showtime" class="rounded-xl border p-2 text-sm" placeholder="showtime_id" />
        <input v-model="b_status" class="rounded-xl border p-2 text-sm" placeholder="status (BOOKED...)" />
        <input v-model="b_user" class="rounded-xl border p-2 text-sm" placeholder="user_id" />

        <input v-model="b_from" type="date" class="rounded-xl border p-2 text-sm" />
        <input v-model="b_to" type="date" class="rounded-xl border p-2 text-sm" />

        <button
          class="rounded-xl bg-emerald-600 px-4 py-2 text-sm font-semibold text-white"
          @click="bookingsApplyFilters"
        >
          Apply
        </button>
      </div>

      <div class="flex items-center justify-between text-sm text-gray-600">
        <div>Total: <span class="font-semibold">{{ bookingsTotal }}</span></div>
        <div class="flex items-center gap-2">
          <button
            class="rounded-lg border px-3 py-1 disabled:opacity-40"
            :disabled="!bookingsHasPrev"
            @click="b_skip -= b_limit; loadBookings()"
          >
            Prev
          </button>
          <span>Page {{ bookingsPage }}</span>
          <button
            class="rounded-lg border px-3 py-1 disabled:opacity-40"
            :disabled="!bookingsHasNext"
            @click="b_skip += b_limit; loadBookings()"
          >
            Next
          </button>
        </div>
      </div>

      <div v-if="bookingsError" class="rounded-xl bg-red-50 p-3 text-sm text-red-700">
        {{ bookingsError }}
      </div>
      <div v-if="bookingsLoading" class="text-sm text-gray-600">Loading...</div>

      <div v-else class="overflow-auto rounded-xl border">
        <table class="w-full text-left text-sm">
          <thead class="bg-gray-50">
            <tr>
              <th class="p-3">ID</th>
              <th class="p-3">Showtime</th>
              <th class="p-3">User</th>
              <th class="p-3">Status</th>
              <th class="p-3">Seats</th>
              <th class="p-3">Created</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in bookings" :key="it._id || it.id" class="border-t">
              <td class="p-3 font-mono text-xs">{{ it._id || it.id }}</td>
              <td class="p-3">{{ it.showtime_id || it.showtimeId }}</td>
              <td class="p-3 font-mono text-xs">{{ it.user_id || it.userId }}</td>
              <td class="p-3 font-semibold">{{ it.status }}</td>
              <td class="p-3">{{ fmtSeats(it) }}</td>
              <td class="p-3 text-xs text-gray-600">{{ fmtTime(it) }}</td>
            </tr>

            <tr v-if="bookings.length === 0">
              <td class="p-4 text-gray-500" colspan="6">No results</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- AUDIT TAB -->
    <div v-else class="space-y-4">
      <div class="grid grid-cols-1 gap-3 md:grid-cols-6">
        <input v-model="a_type" class="rounded-xl border p-2 text-sm" placeholder="type (seat.lock...)" />
        <input v-model="a_showtime" class="rounded-xl border p-2 text-sm" placeholder="showtime_id" />
        <input v-model="a_user" class="rounded-xl border p-2 text-sm" placeholder="user_id" />
        <input v-model="a_booking" class="rounded-xl border p-2 text-sm" placeholder="booking_id" />
        <input v-model="a_from" type="date" class="rounded-xl border p-2 text-sm" />
        <input v-model="a_to" type="date" class="rounded-xl border p-2 text-sm" />

        <button
          class="rounded-xl bg-emerald-600 px-4 py-2 text-sm font-semibold text-white md:col-span-6"
          @click="auditApplyFilters"
        >
          Apply
        </button>
      </div>

      <div class="flex items-center justify-between text-sm text-gray-600">
        <div>Total: <span class="font-semibold">{{ auditTotal }}</span></div>
        <div class="flex items-center gap-2">
          <button
            class="rounded-lg border px-3 py-1 disabled:opacity-40"
            :disabled="!auditHasPrev"
            @click="a_skip -= a_limit; loadAudit()"
          >
            Prev
          </button>
          <span>Page {{ auditPage }}</span>
          <button
            class="rounded-lg border px-3 py-1 disabled:opacity-40"
            :disabled="!auditHasNext"
            @click="a_skip += a_limit; loadAudit()"
          >
            Next
          </button>
        </div>
      </div>

      <div v-if="auditError" class="rounded-xl bg-red-50 p-3 text-sm text-red-700">
        {{ auditError }}
      </div>
      <div v-if="auditLoading" class="text-sm text-gray-600">Loading...</div>

      <div v-else class="overflow-auto rounded-xl border">
        <table class="w-full text-left text-sm">
          <thead class="bg-gray-50">
            <tr>
              <th class="p-3">Time</th>
              <th class="p-3">Type</th>
              <th class="p-3">Showtime</th>
              <th class="p-3">User</th>
              <th class="p-3">Booking</th>
              <th class="p-3">Payload</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in audit" :key="it._id || it.id" class="border-t align-top">
              <td class="p-3 text-xs text-gray-600">{{ fmtTime(it) }}</td>
              <td class="p-3 font-semibold">{{ it.type }}</td>
              <td class="p-3">{{ it.showtime_id || it.showtimeId || "-" }}</td>
              <td class="p-3 font-mono text-xs">{{ it.user_id || it.userId || "-" }}</td>
              <td class="p-3 font-mono text-xs">{{ it.booking_id || it.bookingId || "-" }}</td>
              <td class="p-3">
                <pre class="max-w-[560px] whitespace-pre-wrap break-words rounded-lg bg-gray-50 p-2 text-xs">{{
                  typeof it.payload === "string" ? it.payload : JSON.stringify(it.payload ?? it, null, 2)
                }}</pre>
              </td>
            </tr>

            <tr v-if="audit.length === 0">
              <td class="p-4 text-gray-500" colspan="6">No results</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="text-xs text-gray-500">
      Tip: ถ้าเจอ 403 แปลว่า user ไม่ใช่ ADMIN หรือยังไม่ได้ตั้งค่า ADMIN_EMAILS ใน backend
    </div>
  </div>
</template>

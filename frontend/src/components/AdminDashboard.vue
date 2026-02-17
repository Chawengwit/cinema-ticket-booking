<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

const AUTH_KEY = "access_token";
const API_ORIGIN = import.meta.env.VITE_API_ORIGIN || "http://localhost:8080";

type BookingItem = any;
type AuditItem = any;

const tab = ref<"bookings" | "audit">("bookings");

// ---------------- helpers ----------------
function authHeaders(): HeadersInit {
  const token = localStorage.getItem(AUTH_KEY);
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// ---------------- common filter (movie/showtime only) ----------------
const movieFilter = ref(""); // use showtime_id as "movie/showtime filter"

// ---------------- bookings ----------------
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
    if (movieFilter.value) qs.set("showtime_id", movieFilter.value); // <-- filter movie/showtime
    qs.set("limit", String(b_limit.value));
    qs.set("skip", String(b_skip.value));

    const res = await fetch(`${API_ORIGIN}/api/admin/bookings?${qs.toString()}`, {
      headers: authHeaders(),
    });

    const data = await res.json().catch(() => ({} as any));
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
    if (movieFilter.value) qs.set("showtime_id", movieFilter.value); // <-- filter movie/showtime
    qs.set("limit", String(a_limit.value));
    qs.set("skip", String(a_skip.value));

    const res = await fetch(`${API_ORIGIN}/api/admin/audit?${qs.toString()}`, {
      headers: authHeaders(),
    });

    const data = await res.json().catch(() => ({} as any));
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
  <div class="card space-y-5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="section-title">Admin Dashboard</p>
        <p class="section-subtitle">Bookings &amp; Audit logs (filter by movie/showtime only)</p>
      </div>
      <div class="flex gap-2">
        <button class="btn" :class="tab === 'bookings' ? 'btn-primary' : 'btn-ghost'" @click="tab = 'bookings'; loadBookings()">
          Bookings
        </button>
        <button class="btn" :class="tab === 'audit' ? 'btn-primary' : 'btn-ghost'" @click="tab = 'audit'; loadAudit()">
          Audit Logs
        </button>
      </div>
    </div>

    <!-- Filter -->
    <div class="card-muted">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div class="w-full sm:max-w-md">
          <label class="text-xs uppercase text-slate-400">Movie / Showtime</label>
          <input v-model="movieFilter" class="input mt-2" placeholder="Showtime ID (e.g., SHOW1)" />
        </div>
        <div class="flex gap-2">
          <button class="btn btn-secondary" @click="movieFilter = '' ; (tab==='bookings' ? bookingsApplyFilters() : auditApplyFilters())">
            Clear
          </button>
          <button class="btn btn-primary" @click="tab==='bookings' ? bookingsApplyFilters() : auditApplyFilters()">
            Apply
          </button>
        </div>
      </div>
    </div>

    <!-- BOOKINGS TAB -->
    <div v-if="tab === 'bookings'" class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3 text-sm text-slate-300">
        <div class="pill">Total {{ bookingsTotal }}</div>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" :disabled="!bookingsHasPrev" @click="b_skip -= b_limit; loadBookings()">
            Prev
          </button>
          <span class="text-xs text-slate-400">Page {{ bookingsPage }}</span>
          <button class="btn btn-secondary" :disabled="!bookingsHasNext" @click="b_skip += b_limit; loadBookings()">
            Next
          </button>
        </div>
      </div>

      <div v-if="bookingsError" class="alert alert-error">{{ bookingsError }}</div>
      <div v-else-if="bookingsLoading" class="alert alert-info">Loading bookings...</div>
      <div v-else class="table-shell">
        <table class="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Showtime</th>
              <th>User</th>
              <th>Status</th>
              <th>Seats</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in bookings" :key="it._id || it.id">
              <td class="font-mono text-xs">{{ it._id || it.id }}</td>
              <td class="font-mono text-xs">{{ it.showtime_id || it.showtimeId }}</td>
              <td class="font-mono text-xs">{{ it.user_id || it.userId }}</td>
              <td>
                <span
                  class="badge"
                  :class="
                    (it.status || '').toUpperCase() === 'BOOKED'
                      ? 'badge-success'
                      : (it.status || '').toUpperCase() === 'PENDING'
                      ? 'badge-warning'
                      : 'badge-primary'
                  "
                >
                  {{ it.status || "-" }}
                </span>
              </td>
              <td>{{ fmtSeats(it) }}</td>
              <td class="text-xs text-slate-400">{{ fmtTime(it) }}</td>
            </tr>
            <tr v-if="bookings.length === 0">
              <td colspan="6" class="px-4 py-6 text-center text-slate-400">No bookings found</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- AUDIT TAB -->
    <div v-else class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3 text-sm text-slate-300">
        <div class="pill">Total {{ auditTotal }}</div>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" :disabled="!auditHasPrev" @click="a_skip -= a_limit; loadAudit()">
            Prev
          </button>
          <span class="text-xs text-slate-400">Page {{ auditPage }}</span>
          <button class="btn btn-secondary" :disabled="!auditHasNext" @click="a_skip += a_limit; loadAudit()">
            Next
          </button>
        </div>
      </div>

      <div v-if="auditError" class="alert alert-error">{{ auditError }}</div>
      <div v-else-if="auditLoading" class="alert alert-info">Loading audit logs...</div>
      <div v-else class="table-shell">
        <table class="table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Type</th>
              <th>Showtime</th>
              <th>User</th>
              <th>Booking</th>
              <th>Payload</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in audit" :key="it._id || it.id">
              <td class="text-xs text-slate-400">{{ fmtTime(it) }}</td>
              <td class="font-semibold">{{ it.type }}</td>
              <td class="font-mono text-xs">{{ it.showtime_id || it.showtimeId || "-" }}</td>
              <td class="font-mono text-xs">{{ it.user_id || it.userId || "-" }}</td>
              <td class="font-mono text-xs">{{ it.booking_id || it.bookingId || "-" }}</td>
              <td>
                <pre class="max-w-[560px] whitespace-pre-wrap break-words rounded-xl bg-slate-900/70 p-3 text-xs ring-1 ring-white/5"
                >{{ typeof it.payload === "string" ? it.payload : JSON.stringify(it.payload ?? it, null, 2) }}</pre>
              </td>
            </tr>
            <tr v-if="audit.length === 0">
              <td colspan="6" class="px-4 py-6 text-center text-slate-400">No audit logs found</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="text-xs text-slate-400">
      Tip: 403 responses mean the account is not in <span class="font-mono">ADMIN_EMAILS</span> or the JWT is missing/expired.
    </div>
  </div>
</template>

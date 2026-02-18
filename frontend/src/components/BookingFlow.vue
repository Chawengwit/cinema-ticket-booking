<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";

const props = defineProps<{
  apiOrigin: string;
  authKey: string;
  isAuthed: boolean;
}>();

type Movie = {
  id: string;
  title: string;
  duration?: string;
  rating?: string;
  tags?: string[];
  showtimes: { id: string; label: string }[];
};

const movies = ref<Movie[]>([
  {
    id: "MOV1",
    title: "Demo Movie A",
    duration: "2h 10m",
    rating: "PG-13",
    tags: ["Demo"],
    showtimes: [
      { id: "SHOW1", label: "SHOW1 • Demo Showtime" },
      { id: "SHOW2", label: "SHOW2 • Demo Showtime" },
    ],
  },
  {
    id: "MOV2",
    title: "Demo Movie B",
    duration: "1h 55m",
    rating: "PG",
    tags: ["Demo"],
    showtimes: [{ id: "demo-001", label: "demo-001 • Demo Showtime" }],
  },
]);

type Step = "pick_movie" | "pick_seats" | "pay" | "done";
const step = ref<Step>("pick_movie");

const selectedMovieId = ref("");
const selectedShowtimeId = ref("");

const selectedMovie = computed(() => movies.value.find((m) => m.id === selectedMovieId.value) || null);
const showtimes = computed(() => selectedMovie.value?.showtimes || []);

const token = computed(() => localStorage.getItem(props.authKey) || "");
function authHeaders(): HeadersInit {
  return token.value ? { Authorization: `Bearer ${token.value}` } : {};
}
function wsBase(url: string) {
  return url.replace(/^http/, "ws");
}

type SeatStatus = "FREE" | "LOCKED" | "BOOKED";
type Seat = { id: string; row: string; num: number; status: SeatStatus; owner?: string };

const seatRows = ["A", "B", "C", "D", "E"];
const seatCols = 10;

function buildSeats(): Seat[] {
  const out: Seat[] = [];
  for (const r of seatRows)
    for (let n = 1; n <= seatCols; n++) out.push({ id: `${r}${n}`, row: r, num: n, status: "FREE" });
  return out;
}

const seats = ref<Seat[]>(buildSeats());

const picked = ref<string[]>([]);
const lockedSeats = ref<string[]>([]);

const busy = ref(false);
const error = ref<string | null>(null);

const wsConnected = ref(false);
let ws: WebSocket | null = null;

const lockRequestId = ref<string>("");
const paymentRef = ref("");
const bookingId = ref("");
const doneMessage = ref("");

function genPaymentRef() {
  return "PAY-" + Math.random().toString(16).slice(2).toUpperCase();
}

/**
 * ✅ Auto sync seat state:
 * - locks: seatlock:showtime:*
 * - booked: seatbooked:showtime:*
 */
async function syncSeatState() {
  if (!props.isAuthed || !selectedShowtimeId.value) return;

  try {
    const res = await fetch(
      `${props.apiOrigin}/api/showtimes/${encodeURIComponent(selectedShowtimeId.value)}/seats/state`,
      { headers: authHeaders() }
    );
    const data = await res.json().catch(() => ({} as any));
    if (!res.ok || !data?.ok) return;

    // reset to FREE
    for (const s of seats.value) {
      s.status = "FREE";
      s.owner = undefined;
    }

    // apply booked first
    const booked: string[] = Array.isArray(data.booked) ? data.booked : [];
    for (const id of booked) {
      const s = seats.value.find((x) => x.id === id);
      if (s) {
        s.status = "BOOKED";
        s.owner = undefined;
      }
    }

    // apply locks
    const locks = Array.isArray(data.locks) ? data.locks : [];
    for (const l of locks) {
      const sid = l.seat_id;
      const s = seats.value.find((x) => x.id === sid);
      if (s && s.status !== "BOOKED") {
        s.status = "LOCKED";
        s.owner = l.owner;
      }
    }

    // clean picked if became not FREE
    picked.value = picked.value.filter((id) => seats.value.find((s) => s.id === id)?.status === "FREE");
  } catch {}
}

function seatClass(s: Seat) {
  const isPicked = picked.value.includes(s.id);
  const isLockedForPay = lockedSeats.value.includes(s.id);

  const base =
    "h-10 w-10 rounded-xl text-xs font-semibold flex items-center justify-center select-none ring-1 transition";

  if (s.status === "BOOKED") return `${base} bg-rose-500/15 text-rose-200 ring-rose-400/20 cursor-not-allowed`;
  if (isLockedForPay) return `${base} bg-emerald-500/18 text-emerald-200 ring-emerald-400/25 cursor-not-allowed`;
  if (s.status === "LOCKED") return `${base} bg-amber-500/15 text-amber-200 ring-amber-400/20 cursor-not-allowed`;
  if (isPicked) return `${base} bg-emerald-500/20 text-emerald-200 ring-emerald-400/30 hover:bg-emerald-500/25 cursor-pointer`;
  return `${base} bg-white/5 text-white ring-white/10 hover:bg-white/10 cursor-pointer`;
}

function toggleSeat(id: string) {
  const s = seats.value.find((x) => x.id === id);
  if (!s) return;
  if (step.value !== "pick_seats") return;

  // ถ้ามี lockedSeats อยู่แล้ว ให้ user ไปจ่ายหรือยกเลิกก่อน (กัน state ซ้อน)
  if (lockedSeats.value.length > 0) return;

  if (s.status !== "FREE") return;

  const idx = picked.value.indexOf(id);
  if (idx >= 0) picked.value.splice(idx, 1);
  else picked.value.push(id);
}

// ===== WebSocket =====
function applyEvent(type: string, seatIds: string[], owner?: string) {
  for (const id of seatIds) {
    const s = seats.value.find((x) => x.id === id);
    if (!s) continue;

    if (type === "locked") {
      if (s.status !== "BOOKED") {
        s.status = "LOCKED";
        s.owner = owner;
      }
    } else if (type === "released" || type === "timeout") {
      if (s.status !== "BOOKED") {
        s.status = "FREE";
        s.owner = undefined;
      }
    } else if (type === "booked") {
      s.status = "BOOKED";
      s.owner = undefined;
    }
  }
}

function connectWS() {
  if (!token.value || !selectedShowtimeId.value) return;
  if (ws) ws.close();

  const url = `${wsBase(props.apiOrigin)}/ws/showtimes/${encodeURIComponent(selectedShowtimeId.value)}/seats?token=${encodeURIComponent(
    token.value
  )}`;
  ws = new WebSocket(url);

  ws.onopen = async () => {
    wsConnected.value = true;
    // ✅ สำคัญ: connect แล้ว sync state ทันที
    await syncSeatState();
  };

  ws.onclose = () => {
    wsConnected.value = false;
  };

  ws.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data);
      const type = String(msg?.type || "");
      const seatIds = (msg?.seat_ids || msg?.seatIds || []) as string[];
      const owner = msg?.owner;

      if (type && Array.isArray(seatIds) && seatIds.length > 0) {
        applyEvent(type, seatIds, owner);

        // กัน user เลือกทับ (เฉพาะตอน pick_seats)
        if (step.value === "pick_seats" && (type === "locked" || type === "booked")) {
          picked.value = picked.value.filter((id) => !seatIds.includes(id));
        }
      }
    } catch {}
  };
}

function disconnectWS() {
  ws?.close();
  ws = null;
  wsConnected.value = false;
}

onBeforeUnmount(() => disconnectWS());

// ===== Step transitions =====
function goPickSeats() {
  error.value = null;
  if (!props.isAuthed) {
    error.value = "Please login first.";
    return;
  }
  if (!selectedMovieId.value || !selectedShowtimeId.value) {
    error.value = "Please select a movie and showtime.";
    return;
  }
  step.value = "pick_seats";
}

// เปลี่ยน showtime => reset state & connect ws & sync
watch(selectedShowtimeId, async () => {
  seats.value = buildSeats();
  picked.value = [];
  lockedSeats.value = [];
  lockRequestId.value = "";
  paymentRef.value = "";
  bookingId.value = "";
  doneMessage.value = "";
  error.value = null;

  disconnectWS();
  if (props.isAuthed && selectedShowtimeId.value) {
    connectWS();
    await syncSeatState();
  }
});

// ✅ ทุกครั้งที่เข้าหน้า pick_seats ให้ sync สี (แก้บัคข้อ 1)
watch(step, async (s) => {
  if (s === "pick_seats" && selectedShowtimeId.value) {
    await syncSeatState();
  }
});

// ===== API actions =====
async function lockSeats() {
  error.value = null;
  if (!props.isAuthed || !selectedShowtimeId.value) return;

  const seatsToLock = [...picked.value].sort();
  if (seatsToLock.length === 0) return;

  busy.value = true;
  try {
    const res = await fetch(
      `${props.apiOrigin}/api/showtimes/${encodeURIComponent(selectedShowtimeId.value)}/seats/lock`,
      {
        method: "POST",
        headers: { ...authHeaders(), "Content-Type": "application/json" } as any,
        body: JSON.stringify({ seat_ids: seatsToLock }),
      }
    );

    const data = await res.json().catch(() => ({} as any));
    if (res.status === 409) throw new Error(data?.error || "seats_unavailable");
    if (!res.ok || !data?.ok) throw new Error(data?.error || `HTTP_${res.status}`);

    lockRequestId.value = data.request_id;
    lockedSeats.value = seatsToLock;
    paymentRef.value = genPaymentRef();

    applyEvent("locked", seatsToLock, "me");
    picked.value = [];
    step.value = "pay";
  } catch (e: any) {
    error.value = e?.message ?? "Lock seats failed";
    // ถ้า lock fail -> sync ใหม่ให้เห็นสีจริง
    await syncSeatState();
  } finally {
    busy.value = false;
  }
}

async function releaseSeats() {
  error.value = null;
  if (!props.isAuthed || !selectedShowtimeId.value) return;
  if (lockedSeats.value.length === 0) return;

  busy.value = true;
  try {
    const res = await fetch(
      `${props.apiOrigin}/api/showtimes/${encodeURIComponent(selectedShowtimeId.value)}/seats/lock`,
      {
        method: "DELETE",
        headers: { ...authHeaders(), "Content-Type": "application/json" } as any,
        body: JSON.stringify({ seat_ids: lockedSeats.value }),
      }
    );

    const data = await res.json().catch(() => ({} as any));
    if (!res.ok || !data?.ok) throw new Error(data?.error || `HTTP_${res.status}`);

    applyEvent("released", lockedSeats.value);
    lockedSeats.value = [];
    lockRequestId.value = "";
    paymentRef.value = "";
    step.value = "pick_seats";

    await syncSeatState();
  } catch (e: any) {
    error.value = e?.message ?? "Release failed";
  } finally {
    busy.value = false;
  }
}

async function confirmBooking() {
  error.value = null;
  if (!props.isAuthed || !selectedShowtimeId.value) return;
  if (lockedSeats.value.length === 0) return;

  if (!paymentRef.value) paymentRef.value = genPaymentRef();
  if (!lockRequestId.value) {
    error.value = "Missing request_id. Please lock again.";
    return;
  }

  busy.value = true;
  try {
    const res = await fetch(
      `${props.apiOrigin}/api/showtimes/${encodeURIComponent(selectedShowtimeId.value)}/bookings/confirm`,
      {
        method: "POST",
        headers: { ...authHeaders(), "Content-Type": "application/json" } as any,
        body: JSON.stringify({
          seat_ids: lockedSeats.value,
          payment_ref: paymentRef.value,
          request_id: lockRequestId.value,
        }),
      }
    );

    const data = await res.json().catch(() => ({} as any));
    if (!res.ok || !data?.ok) throw new Error(data?.error || `HTTP_${res.status}`);

    bookingId.value = data.booking_id || data.id || "";
    doneMessage.value = "Booking completed successfully.";

    applyEvent("booked", lockedSeats.value);

    step.value = "done";
  } catch (e: any) {
    error.value = e?.message ?? "Confirm booking failed";
    await syncSeatState();
  } finally {
    busy.value = false;
  }
}

// ✅ Back rules:
// - จาก pay -> กลับ pick_seats (ไม่กลับ movie) เพื่อไม่ให้ state หลุด (แก้บัคข้อ 2)
function back() {
  if (step.value === "pay") {
    step.value = "pick_seats";
    return;
  }
  step.value = "pick_movie";
}

function startNewBooking() {
  step.value = "pick_movie";
  selectedMovieId.value = "";
  selectedShowtimeId.value = "";
  seats.value = buildSeats();
  picked.value = [];
  lockedSeats.value = [];
  lockRequestId.value = "";
  paymentRef.value = "";
  bookingId.value = "";
  doneMessage.value = "";
  error.value = null;
  disconnectWS();
}
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2 text-xs">
        <span class="pill" :class="step==='pick_movie' ? 'text-white border border-white/20' : 'text-slate-300 border border-white/10'">1) Movie</span>
        <span class="text-slate-500">→</span>
        <span class="pill" :class="step==='pick_seats' ? 'text-white border border-white/20' : 'text-slate-300 border border-white/10'">2) Seats</span>
        <span class="text-slate-500">→</span>
        <span class="pill" :class="step==='pay' ? 'text-white border border-white/20' : 'text-slate-300 border border-white/10'">3) Pay</span>
        <span class="text-slate-500">→</span>
        <span class="pill" :class="step==='done' ? 'text-white border border-white/20' : 'text-slate-300 border border-white/10'">4) Done</span>
      </div>

      <span class="pill" :class="wsConnected ? 'text-emerald-200 border border-emerald-400/20' : 'text-slate-300 border border-white/10'">
        {{ wsConnected ? "Live events: ON" : "Live events: OFF" }}
      </span>
    </div>

    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <!-- Step 1 -->
    <div v-if="step === 'pick_movie'" class="space-y-4">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <button
          v-for="m in movies"
          :key="m.id"
          class="card-muted text-left hover:bg-white/10 transition"
          :class="selectedMovieId === m.id ? 'ring-2 ring-white/20' : 'ring-1 ring-white/5'"
          @click="selectedMovieId = m.id"
        >
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="text-white font-semibold">{{ m.title }}</p>
              <p class="text-xs text-slate-400 mt-1">{{ m.duration }} • {{ m.rating }}</p>
              <div class="mt-2 flex flex-wrap gap-2">
                <span v-for="t in (m.tags || [])" :key="t" class="pill text-slate-200 border border-white/10">{{ t }}</span>
              </div>
            </div>
            <span class="pill text-slate-200 border border-white/10">{{ m.showtimes.length }} showtimes</span>
          </div>
        </button>
      </div>

      <div class="card-muted space-y-3">
        <p class="text-sm font-semibold text-white">Showtime</p>
        <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <button
            v-for="s in showtimes"
            :key="s.id"
            class="btn"
            :class="selectedShowtimeId === s.id ? 'btn-primary' : 'btn-ghost'"
            @click="selectedShowtimeId = s.id"
          >
            {{ s.label }} <span class="ml-2 text-xs opacity-70">({{ s.id }})</span>
          </button>
          <div v-if="!selectedMovie" class="text-sm text-slate-400">Select a movie first.</div>
        </div>

        <div class="flex items-center justify-between gap-3">
          <div class="text-xs text-slate-400">Login required: <span class="font-mono">{{ props.isAuthed ? "YES" : "NO" }}</span></div>
          <button class="btn btn-primary" @click="goPickSeats" :disabled="!props.isAuthed || !selectedShowtimeId">
            Continue to seat selection
          </button>
        </div>
      </div>
    </div>

    <!-- Seats / Pay / Done -->
    <div v-else class="space-y-4">
      <div class="card-muted">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-sm font-semibold text-white">
              {{ step === "pick_seats" ? "Choose seats" : step === "pay" ? "Review seats & pay" : "Booking summary" }}
            </p>
            <p class="text-xs text-slate-400">
              Showtime: <span class="font-mono text-slate-200">{{ selectedShowtimeId }}</span>
            </p>
          </div>

          <div class="flex items-center gap-2">
            <span class="pill text-slate-200 border border-white/10">Picked: {{ picked.length }}</span>
            <span class="pill text-slate-200 border border-white/10">Locked: {{ lockedSeats.length }}</span>
          </div>
        </div>

        <div class="mt-4">
          <div class="mb-3 flex items-center justify-between text-xs text-slate-400">
            <span>Screen</span>
            <div class="flex items-center gap-2">
              <span class="pill border border-white/10 text-slate-300">FREE</span>
              <span class="pill border border-emerald-400/20 text-emerald-200 bg-emerald-500/10">PICKED</span>
              <span class="pill border border-amber-400/20 text-amber-200 bg-amber-500/10">LOCKED</span>
              <span class="pill border border-emerald-400/25 text-emerald-200 bg-emerald-500/10">LOCKED (ME)</span>
              <span class="pill border border-rose-400/20 text-rose-200 bg-rose-500/10">BOOKED</span>
            </div>
          </div>

          <div class="h-2 rounded-full bg-white/10 mb-4" />

          <div class="space-y-2">
            <div v-for="r in seatRows" :key="r" class="flex items-center gap-2">
              <div class="w-6 text-xs text-slate-400">{{ r }}</div>
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="n in seatCols"
                  :key="`${r}${n}`"
                  type="button"
                  :class="seatClass(seats.find(x => x.id === `${r}${n}`)!)"
                  @click="toggleSeat(`${r}${n}`)"
                  :disabled="busy || step !== 'pick_seats'"
                >
                  {{ r }}{{ n }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
          <div class="text-xs text-slate-400">
            Note: 409 seats_unavailable means someone locked/booked it first — pick another seat.
          </div>

          <div class="flex gap-2">
            <button class="btn btn-ghost" @click="back" :disabled="busy">Back</button>

            <button
              v-if="step==='pick_seats' && lockedSeats.length===0"
              class="btn btn-primary"
              @click="lockSeats"
              :disabled="busy || picked.length===0"
            >
              Lock & Continue
            </button>

            <!-- ✅ ถ้ามี lockedSeats อยู่แล้วในหน้า pick_seats (เกิดจาก back จาก pay) ให้มีปุ่มไปจ่าย/ยกเลิก -->
            <template v-if="step==='pick_seats' && lockedSeats.length>0">
              <button class="btn btn-danger" @click="releaseSeats" :disabled="busy">Cancel & Release</button>
              <button class="btn btn-primary" @click="step='pay'" :disabled="busy || !lockRequestId">Continue to Pay</button>
            </template>

            <template v-else-if="step==='pay'">
              <button class="btn btn-danger" @click="releaseSeats" :disabled="busy || lockedSeats.length===0">
                Cancel & Release
              </button>
              <button class="btn btn-primary" @click="confirmBooking" :disabled="busy || lockedSeats.length===0 || !lockRequestId">
                Pay & Confirm
              </button>
            </template>

            <button v-else-if="step==='done'" class="btn btn-primary" @click="startNewBooking" :disabled="busy">
              Book another
            </button>
          </div>
        </div>

        <div v-if="step==='pay'" class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div class="card-muted">
            <p class="text-xs uppercase text-slate-400">Seats</p>
            <p class="text-white font-semibold mt-1">{{ lockedSeats.join(", ") }}</p>
          </div>
          <div class="card-muted">
            <p class="text-xs uppercase text-slate-400">Amount (mock)</p>
            <p class="text-white font-semibold mt-1">{{ lockedSeats.length * 200 }} THB</p>
          </div>
          <div class="card-muted">
            <p class="text-xs uppercase text-slate-400">Payment ref (auto)</p>
            <p class="text-white font-semibold mt-1 font-mono break-words">{{ paymentRef || "-" }}</p>
          </div>
        </div>

        <div v-if="step==='done'" class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div class="card-muted">
            <p class="text-xs uppercase text-slate-400">Status</p>
            <p class="text-white font-semibold mt-1">COMPLETED</p>
            <p class="text-xs text-slate-400 mt-1">{{ doneMessage }}</p>
          </div>
          <div class="card-muted">
            <p class="text-xs uppercase text-slate-400">Booking ID</p>
            <p class="text-white font-semibold mt-1 font-mono wrap-break-words">{{ bookingId || "-" }}</p>
          </div>
          <div class="card-muted">
            <p class="text-xs uppercase text-slate-400">Payment ref</p>
            <p class="text-white font-semibold mt-1 font-mono wrap-break-words">{{ paymentRef || "-" }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

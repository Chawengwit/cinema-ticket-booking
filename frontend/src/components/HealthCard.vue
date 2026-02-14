<script setup lang="ts">

    import {ref, onMounted} from 'vue';

    const health = ref<any>(null);
    const error = ref<string | null>(null);

    onMounted(async () => {
        try {
            const res = await fetch("/api/health");
            if(!res.ok){
            throw new Error(`HTTP ${res.status}`);
            }

            health.value = await res.json();

        } catch (e: any){
            error.value = e?.message ?? "Failed to fetching health checker.";
        }
    });

</script>

<template>
    <div class="mx-auto max-w-3xl rounded-2xl bg-white p-6 shadow">
        <div class="flex flex-col items-left justify-between">
            <h1 class="text-2xl font-bold tracking-tight">Cinema Ticket Booking</h1>
            <span class="round-full bg-emerald-50 px-3 py-1 text-sm font-medium text-emerald-700">what movie you love?</span>
            
            <p class="mt-2 text-slate-600">Frontend can call /health Checker..</p>
            
            <div
                v-if="error"
                class="mt-4 rounded-xl border border-rose-200 bg-rose-50 p-4 text-rose-700"
            >{{ error }}</div>
            
            <pre
                v-else
                class="mt-4 overflow-auto rounded-xl bg-slate-100 p-4 text-sm text-slate-800"
            >{{ health }}</pre>
            
        </div>
    </div>
</template>
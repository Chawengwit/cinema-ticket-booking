<script setup lang="ts">
    import { ref, onMounted } from 'vue';
    import HealthCard from './components/HealthCard.vue';

    const user = ref<any>(null);
    const error = ref<string | null>(null)

    onMounted(async () => {
        const token = localStorage.getItem("access_token");
        if(!token){
            console.log("No token found");
            return;
        }

        try {
            const res = await fetch("http://localhost:8080/api/me", {
                headers: {
                    Authorization: `Bearer ${token}`
                }
            });

            if(!res.ok){
                throw new Error("Unauthorized");
            }

            user.value = await res.json();

        } catch (err: any) {
            error.value = err.message;
        }
    });

</script>

<template>
  <div class="min-h-screen p-6">
    
    <!-- แสดง user -->
    <div v-if="user" class="mb-6 rounded-xl bg-green-50 p-4">
      <p class="font-semibold text-green-700">Logged in user:</p>
      <pre class="text-sm">{{ user }}</pre>
    </div>

    <div v-else-if="error" class="mb-6 text-red-600">
      {{ error }}
    </div>

    <HealthCard />
  </div>
</template>
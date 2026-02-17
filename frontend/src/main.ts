import { createApp } from 'vue'
import './style.css'
import App from './App.vue'



// Google OAuth Callback Handler
const params = new URLSearchParams(window.location.search);
const token = params.get("token");

if(token){
    localStorage.setItem("access_token", token);

    // remove token from url
    window.history.replaceState({}, document.title, "/");

    console.log("Login success");
}

createApp(App).mount('#app')

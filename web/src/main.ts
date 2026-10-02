import "./styles.css";
import { createApp } from "vue";
import App from "./App.vue";
import HubApp from "./components/HubApp.vue";
const host = document.getElementById("ui-root");
if (host) createApp(host.dataset.view === "hub" ? HubApp : App).mount(host);

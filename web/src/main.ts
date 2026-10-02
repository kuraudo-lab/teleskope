import { createApp } from 'vue'
import ReportMode from './ReportMode.vue'
const host = document.createElement('div')
host.id = 'report-mode'
document.body.append(host)
createApp(ReportMode).mount(host)

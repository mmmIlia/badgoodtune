import { createApp } from 'vue'

import App from './App.vue'
import { client } from './api/generated/client.gen'
import router from './router'
import './styles/foundation.css'
import './styles/catalog.css'
import './styles/forms.css'
import './styles/pages.css'
import './styles/responsive.css'

client.setConfig({ credentials: 'same-origin' })

createApp(App).use(router).mount('#app')

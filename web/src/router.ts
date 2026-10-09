import { createRouter, createWebHistory } from 'vue-router'

import { authState, ensureCurrentUser } from './state/auth'
import AlbumCreateView from './views/AlbumCreateView.vue'
import AlbumDetailView from './views/AlbumDetailView.vue'
import AlbumEditView from './views/AlbumEditView.vue'
import CatalogView from './views/CatalogView.vue'
import LoginView from './views/LoginView.vue'
import NotFoundView from './views/NotFoundView.vue'
import RegisterView from './views/RegisterView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'catalog', component: CatalogView },
    { path: '/login', name: 'login', component: LoginView, meta: { guestOnly: true } },
    { path: '/register', name: 'register', component: RegisterView, meta: { guestOnly: true } },
    { path: '/albums/new', name: 'album-create', component: AlbumCreateView, meta: { requiresAuth: true } },
    { path: '/albums/:albumId', name: 'album-detail', component: AlbumDetailView },
    { path: '/albums/:albumId/edit', name: 'album-edit', component: AlbumEditView, meta: { requiresAuth: true } },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach(async (to) => {
  await ensureCurrentUser()
  if (to.meta.requiresAuth && !authState.user) {
    return { name: 'login', query: { returnTo: to.fullPath } }
  }
  if (to.meta.guestOnly && authState.user) {
    return { name: 'catalog' }
  }
})

export default router

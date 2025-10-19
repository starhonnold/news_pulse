const routes = [
  {
    path: '/',
    component: () => import('layouts/MainLayout.vue'),
    children: [
      { path: '', component: () => import('pages/NewsPage.vue') },
      { path: 'news', redirect: '/' }, // Редирект с /news на главную
      { path: 'pulses', component: () => import('pages/IndexPage.vue') },
      { path: 'pulse/:id', component: () => import('pages/PulsePage.vue') }
    ],
  },

  // Always leave this as last one,
  // but you can also remove it
  {
    path: '/:catchAll(.*)*',
    component: () => import('pages/ErrorNotFound.vue'),
  },
]

export default routes

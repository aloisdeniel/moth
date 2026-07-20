// The example's service worker — the minimal sw.js from the @moth/react
// README. Display and click handling are app code by design: moth manages
// the subscription and the registry row, the payload shape is whatever your
// backend sends. This one expects { title, body, url }.
self.addEventListener('push', (event) => {
  const data = event.data?.json() ?? {}
  event.waitUntil(
    self.registration.showNotification(data.title ?? 'Notification', {
      body: data.body,
      data,
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  event.waitUntil(self.clients.openWindow(event.notification.data?.url ?? '/'))
})

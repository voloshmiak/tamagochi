self.addEventListener("install", () => self.skipWaiting());

self.addEventListener("activate", (event) => event.waitUntil(self.clients.claim()));

self.addEventListener("push", (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    data = { title: "Тамагочі", body: event.data ? event.data.text() : "" };
  }
  const title = data.title || "Тамагочі";
  event.waitUntil(
    self.registration.showNotification(title, {
      body: data.body || "",
      tag: data.tag || undefined,
      data: { url: data.url || "/" },
    }),
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const target = new URL(event.notification.data?.url || "/", self.location.origin).href;
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((list) => {
      for (const client of list) {
        if (new URL(client.url).origin === self.location.origin && "focus" in client) {
          client.navigate(target);
          return client.focus();
        }
      }
      return self.clients.openWindow(target);
    }),
  );
});

self.addEventListener("pushsubscriptionchange", (event) => {
  const old = event.oldSubscription;
  event.waitUntil(
    fetch("/api/v1/notifications/push-key", { credentials: "include" })
      .then((res) => (res.ok ? res.json() : Promise.reject(res.status)))
      .then(({ publicKey }) =>
        self.registration.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: publicKey }),
      )
      .then((sub) =>
        fetch("/api/v1/notifications/subscriptions", {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(sub.toJSON()),
        }),
      )
      .then(() =>
        old
          ? fetch("/api/v1/notifications/subscriptions", {
              method: "DELETE",
              credentials: "include",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ endpoint: old.endpoint }),
            })
          : undefined,
      )
      .catch(() => undefined),
  );
});

// to fix navigator.serviceWorker.controller == null
// https://stackoverflow.com/questions/38168276/navigator-serviceworker-controller-is-null-until-page-refresh/38690771#38690771
self.addEventListener('install', function(event) {
    event.waitUntil(self.skipWaiting()); // Activate worker immediately
});

self.addEventListener('activate', function(event) {
    event.waitUntil(self.clients.claim()); // Become available to all pages
});


// Show notification when received
self.addEventListener('message', (event) => {

    let notification = event.data;
    console.warn("Got a message", notification);
    self.registration.showNotification(
        notification.title,
        notification.options
    ).catch((error) => {
        console.log(error);
    });
});


const audio = new Audio(`/call.mp3`);

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

    audio.play()
});

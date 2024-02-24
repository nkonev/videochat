

self.addEventListener('activate', async () => {
  // This will be called only once when the service worker is activated.
  console.log('service worker activated');

  setInterval(async ()=>{
    const options = {
      body: "Body",
      // here you can add more properties like icon, image, vibrate, etc.
    };
    // self.registration.showNotification("title", options);
    console.log(">", self);
    const re = await fetch("https://ipinfo.io", {
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      },
    });
    console.log(">>>", await re.json());
  }, 1000)

})

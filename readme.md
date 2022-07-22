[![Build Status](https://github.com/nkonev/videochat/workflows/CI%20jobs/badge.svg)](https://github.com/nkonev/videochat/actions)

# Videochat
Your open source self-hosted videoconference platform.

[![Chat image](./.screenshots/2_chat_participants_management.png)](./screenshots.md)

# Key features:
* Well-integrated video calls into entire platform UI, no separated video rooms, text chats, etc...
* No installation on client PC - only modern browser with video camera or microphone required.
* Screen [sharing](./screenshots.md#screen-sharing).
* Multiple cameras support.
* One user can use several devices simultaneously (e. g. smartphone and PC).
* [Muting, kicking](./screenshots.md#videoconference-and-participant-management) video participants.
* Calling to user to [invite](./screenshots.md#inviting-user-to-videoconference) his or her to video conference.
* User is [speaking indication](./screenshots.md#user-is-speaking-indication-green-nickname-and-microphone).
* File [sharing](./screenshots.md#chat-files).
* [Tet-a-tet](./screenshots.md#open-tet-a-tet-chat) private chats for two.
* Horizontal scaling particular video rooms (chats) by servers.
* Horizontal scaling other microservices.
* Supports [login](./screenshots.md#login) through OpedID Connect providers: Facebook, VK.com, Google, Keycloak. Not required can be disabled.
* Internationalization: English and Russian UI.
* Simple setup with docker swarm or docker-compose.
* Self-contained frontend bundle without any CDN downloads.
* No need to edit `/etc/hosts` for development.

# Try
[Demo](https://chat.nkonev.name/)

See [screenshots](./screenshots.md)

# Installation
* Use docker-swarm [files](./deploy)
* Replace `api.site.local` with your actual hostname, remove 8080 if need
* Configure "ingress" in deploy/traefik_conf/traefik.yml and docker-compose-infra.template.yml
* Open ports to traefik and livekit, described in deploy/docker-compose-infra.template.yml

# Troubleshooting
## Reasons of not showing video
### There was a problem when jaeger all-in-one ate too much memory - one of participants didn't see other
### Mobile Chrome 101.0.4951.41 - swap it up (e. g. close application and open again) helps when video isn't connected from Mobile Chrome.
### Desktop Firefox - try to reload tab or restart entire browser - it helps when Desktop Firefox isn't able to show video from Mobile Chrome

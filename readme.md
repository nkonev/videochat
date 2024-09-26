[![Build Status](https://github.com/nkonev/videochat/workflows/CI%20jobs/badge.svg)](https://github.com/nkonev/videochat/actions)

[GitHub](https://github.com/nkonev/videochat) | [GitFlic](https://gitflic.ru/project/nkonev/videochat)

# Videochat
Your open source self-hosted videoconference platform.

# Why
Today Web is ubiquitous. Most of computer's users have web browsers. 
Usually they are modern versions of either Chrome- or Firefox-based browsers.
It seems enough to make video calls. But...

Many of popular communication platforms ignore this fact. 
Typically, they force you to install their Electron-based application on your computer.
Hence, along with web browser you open their heavy application, the resource consumption grows.

Many of popular video platforms store your data on their servers, it means they actually control your data.
This increases risks of data leak, also it makes it possible for them to sell your data, 
to track your actions and watch you. 

Moreover, they can remove all your data in some moment, 
so you can lose your messages, files, discussions, contacts, customers, clients, etc...

They show you annoying or inappropriate ads, you have no option to disable it.

This project offers you a self-hosted solution, that you can incorporate into your infrastructure, 
so you will possess your data and can you apply your own security policies, 
whether to expose this service to the Internet or not, to hide it behind your corporate VPN and so on.

# Screenshots
Click on image to open a screenshot gallery.
[![Chat image](./.screenshots/14_most_of_features.png)](./screenshots.md)

# Features:
* Free HTTPS by Let's Encrypt.
* One domain name.
* Calls from PC to Mobile and vise versa.
* Well-integrated video calls into entire platform UI, no separated video rooms, text chats, etc...
* No installation on client's machine - only modern browser with video camera or microphone required.
* Tested in Firefox and Chrome.
* Multiple cameras support - an user can transmit video simultaneously from several web cameras connected to their computer.
* Multiple devices support - an user can use several devices simultaneously (e. g. smartphone / PC / Laptop / ...).
* Screen [sharing](./screenshots.md#screen-sharing).
* Video recording, recordings are saved to Files.
* [Files](./screenshots.md#chat-files).
* Public files.
* Public messages.
* [Muting, kicking](./screenshots.md#videoconference-and-participant-management) video participants.
* Calling to user to [invite](./screenshots.md#inviting-user-to-videoconference) his or her to video conference.
* User is [speaking indication](./screenshots.md#user-is-speaking-indication-green-nickname-and-microphone).
* Persistent text chats with simple formatting. Messages are persisted in the chat.
* [Tet-a-tet](./screenshots.md#open-tet-a-tet-chat) private chats for two.
* Notifications about `@mention`, `@all`, `@here` and missed video calls.
* Pinned messages.
* Pinned chats.
* Reactions.
* Supports [login](./screenshots.md#login) through OpedID Connect providers: Facebook, VK.com, Google, Keycloak. Not required can be disabled.
* Synchronizing users with LDAP and Keycloak with conflict resolving strategies.
* Internationalization: English and Russian UI.
* Firewall friendly: only two ports for WebRTC are needed (TURN, WebRTC).
* Loadbalancer friendly: No sticky sessions required.
* Horizontal scaling, including video server itself thanks to Livekit.
* Simple setup with Ansible and Docker Swarm.
* No vendor lock on cloud provider.
* Known and popular technologies: PostgreSQL, RabbitMQ, ~Redis~ Valkey, Jaeger, Minio, Traefik, Nginx, Node.js with their communities, no rare nor exotic technologies.
* Self-contained frontend bundle without any CDN downloads - it can work in a closed network without internet access.
* No need to edit `/etc/hosts` for local demo installation or development.
* Send the message when finishing media (image, video) or file has been uploaded.
* Simple SEO-friendly blog, based on chats.

# Try
Demo server [installation](https://chat.nkonev.name/)

# Installation
Currently, Ansible installation is available. See instructions in `./install/ansible/readme.md`.

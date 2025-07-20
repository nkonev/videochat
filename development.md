# Start the development

I develop on Fedora Linux 39, 40.

## Install Docker
Install docker >= 27
[using](https://docs.docker.com/engine/install/fedora/#install-using-the-repository) rpm repository

```bash
sudo dnf -y install dnf-plugins-core
sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
sudo dnf install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

sudo systemctl start docker
sudo systemctl enable docker
```

[Add](https://docs.docker.com/engine/install/linux-postinstall/) your user to the `docker` group:
```bash
sudo usermod -aG docker $USER
```

## Install the developement software
Also you can see versions [here](./.github/workflows/ci.yml).

* Java (JDK) 21 [here](https://bell-sw.com/pages/downloads/#jdk-21-lts) or [here](https://axiomjdk.ru/pages/downloads/#/java-21-lts),
set JAVA_HOME: into `~/.bashrc` just add
```bash
export JAVA_HOME=/usr/lib/jvm/java-21
```
check:
```bash
$JAVA_HOME/bin/javac -version
```
* Node.js 22.12.0 [here](https://nodejs.org/en/download/package-manager) via `nvm` (Node Version Manager)
* Golang 1.22 [here](https://go.dev/dl/)

An example content of `~/.bashrc`:
```bash
export GOPATH=/home/nkonev/go
export GOROOT=/home/nkonev/apps/go122/go
export JAVA_HOME=/usr/lib/jvm/java-21
export PATH="$PATH:$GOROOT/bin:$GOPATH/bin:$JAVA_HOME/bin"

export EDITOR=vim
```

## Install ffmpeg
```bash
sudo dnf install ffmpeg-free
```

## Environment (PostgreSQL, RabbitMQ, ...)
To start the environment do
```
docker compose up -d
```

## How to run the code?
**(Micro)Services**
### Without IDE: 
In general you can look at makefiles in the each (micro)services, they have `run` target.
- https://github.com/nkonev/videochat/blob/master/aaa/Makefile
- https://github.com/nkonev/videochat/blob/master/chat/Makefile
- https://github.com/nkonev/videochat/blob/master/event/Makefile
- https://github.com/nkonev/videochat/blob/master/frontend/Makefile
- https://github.com/nkonev/videochat/blob/master/notification/Makefile
- https://github.com/nkonev/videochat/blob/master/public/Makefile
- https://github.com/nkonev/videochat/blob/master/storage/Makefile
- https://github.com/nkonev/videochat/blob/master/video/Makefile

E. g. for each (micro)service
```
cd aaa
make run
```

```
cd chat
make run
```
...

### With IDE:
I develop in IntelliJ IDEA with Golang and Vue plugins. I open the entire project folder, e.g. `videochat` in the one window.
All the configs doesn't require a change for the development.

#### Java (micro)service:
Just run `AaaApplication.java` with IDE.
#### Go (micro)services:
Just run `main.go` with IDE.
#### Node.js (micro)services:
Run `npm run dev` in the console.


### Congrats!
You've just finished all the preparations. Happy hacking.
Below are just notes, some of them aren't actual. The most useful are ones about configuring [Keycloak](./development.md#keycloak) and [LDAP](./development.md#ldap-example-with-opendj) (an example with OpenDJ).



# Deploying

For the actual deployment options see [here](https://github.com/nkonev/videochat/tree/master/install). Below are some (outdated / useless) notes.

## Saving and loading docker images
```
docker save nkonev/chat-frontend:latest -o /tmp/frontend.tar
scp /tmp/frontend.tar root@nkonev.name:/tmp
ssh ...
docker load -i /tmp/frontend.tar
```

## Deploy to server from a local machine
```
cd aaa
export CONNECT_LINE=user@api.site.local
make clean package push-docker-image-to-server deploy-docker-image
```

## Ansible
* https://medium.com/@knoldus/how-to-install-docker-on-rhel-using-ansible-role-62728c098351
* https://www.digitalocean.com/community/tutorials/how-to-create-and-use-templates-in-ansible-playbooks
* https://www.digitalocean.com/community/tutorial-series/how-to-write-ansible-playbooks

### Get [facts](https://www.digitalocean.com/community/tutorial-series/how-to-write-ansible-playbooks)
```bash
ansible all -i hosts.ini -m setup -a "filter=*ipv4*" -u root
```

### Dry-run
```bash
ansible-playbook -i hosts.ini playbook.yaml --check
```

### Manual apply docker swarm docker compose
```bash
docker stack deploy --compose-file /opt/videochat/docker-compose-infra.yml VIDEOCHATSTACK
journalctl -n 200 -f CONTAINER_TAG=chat-minio
```

### Variables

https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_variables.html

## Generate password
```bash
# https://passlib.readthedocs.io/en/stable/lib/passlib.hash.bcrypt.html
python
import passlib
from passlib.hash import bcrypt
bcrypt.using(rounds=10, salt="salt012345678901234567").hash("admin")
```

## check for community.docker
```
ansible-galaxy collection list
```

## Rsyslog
https://unix.stackexchange.com/questions/599812/is-rsyslog-a-mandatory-requirement-in-linux-with-journald


## Restore data
Be careful! During a short period while videochat uses an empty database, the scheduler in storage microservice can remove files because chats aren't exists.
To prevent this just stop storage for the all period of maintenance.
```
docker service scale VIDEOCHATSTACK_aaa=0
docker service scale VIDEOCHATSTACK_chat=0
docker service scale VIDEOCHATSTACK_storage=0

scp ~/videochat/backup-2024-12-10_00-02-36/chat-aaa.sql THEUSER@nkonev.name:/tmp/chat-aaa.sql
scp ~/videochat/backup-2024-12-10_00-02-36/chat.sql THEUSER@nkonev.name:/tmp/chat.sql

echo 'drop database aaa;' | docker exec -i $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_postgresql --filter desired-state=running -q)) psql -U postgres
echo 'drop database chat;' | docker exec -i $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_postgresql --filter desired-state=running -q)) psql -U postgres

cat /tmp/chat-aaa.sql | docker exec -i $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_postgresql --filter desired-state=running -q)) psql -U postgres
cat /tmp/chat.sql | docker exec -i $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_postgresql --filter desired-state=running -q)) psql -U postgres

rm -rf /tmp/chat-aaa.sql
rm -rf /tmp/chat.sql

docker service scale VIDEOCHATSTACK_aaa=1
docker service scale VIDEOCHATSTACK_chat=1

# copy from old S3 to the new
# on the new minio (target) temporarily publish minio
vim /opt/videochat/docker-compose-infra.yml
# into minio add
    ports:
    - target: 9000
      published: 9000
      protocol: tcp
      mode: host
docker stack deploy --compose-file /opt/videochat/docker-compose-infra.yml VIDEOCHATSTACK

# on the old minio (source)
docker exec -u root -it $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_minio --filter desired-state=running -q)) bash
apt update && apt install vim
vim ~/.mc/config.json
# add (by copying from local)

		"new": {
			"url": "http://new_ip:9000",
			"accessKey": "the_same_key_",
			"secretKey": "the_same_secret_",
			"api": "s3v4",
			"path": "auto"
		},

# test
mc ls new/

# copy
mc cp --recursive local/chat-avatar/ new/chat-avatar
mc cp --recursive local/user-avatar/ new/user-avatar
mc cp --recursive local/files-preview/ new/files-preview
mc cp --recursive local/files/ new/files

# on the new minio (target) remove temporarily published minio
vim /opt/videochat/docker-compose-infra.yml
# from minio remove
    ports:
    - target: 9000
      published: 9000
      protocol: tcp
      mode: host
docker stack deploy --compose-file /opt/videochat/docker-compose-infra.yml VIDEOCHATSTACK

docker service scale VIDEOCHATSTACK_storage=1
```




# AAA

[Error:java: invalid source release: 8](https://stackoverflow.com/a/26009627)


[Reactive, Security, Session MongoDb](https://medium.com/@hantsy/build-a-reactive-application-with-spring-boot-2-0-and-angular-de0ee5837fed)


## Login via curl
```
curl -i 'http://localhost:8060/api/aaa/login' -X POST -H 'Accept: application/json, text/plain, */*' -H 'Content-Type: application/x-www-form-urlencoded;charset=utf-8' -H 'X-XSRF-TOKEN: 724f5acd-3d1e-421b-a386-eb17dcacece8' -H 'Cookie: VIDEOCHAT_XSRF_TOKEN=724f5acd-3d1e-421b-a386-eb17dcacece8' --data-raw 'login=admin&password=admin'
```

## aaa-with-oauth2
use Makefile's goals
* run-oauth2-emu
* run-with-oauth2

## aaa-with-oauth2 for IDE
```
--spring.config.location=file:src/main/resources/config/application.yml,file:src/test/resources/config/oauth2-basic.yml,file:src/test/resources/config/oauth2-keycloak.yml,file:src/test/resources/config/demo-migration.yml,file:src/test/resources/config/log-email.yml
```

## Dump PostgreSQL
```
docker exec -t videochat_postgres_1 pg_dump -U aaa -b --create --column-inserts --serializable-deferrable
```

```
http://localhost:8081/api/user/list?userId=1&userId=-1
```

## Check email sending
```
curl -i -X PUT -H "Content-Type: application/json" -d '{"recipient": "nikita@example.com", "subject": "Test email", "body": "Test body"}' --url 'http://localhost:8060/internal/email'
```

## OAuth2 servers emulator
```
curl -X PUT -i 'http://localhost:9080/recreate-oauth2-mocks'
```

## Starting server for e2e tests
```
cd aaa
export JAVA_HOME=/usr/lib/jvm/java-21
```

use Makefile's goals
* run-oauth2-emu
* run-with-oauth2


```
curl -i -X POST 'http://localhost:9080/recreate-oauth2-mocks'
```

To interact with emulator, you need to use `http://localhost:8081`, not `http://127.0.0.1:8081` because od host checks in OAuth


# Go

## Gorce Go to pull the packages
```
go list -m -json all
```

## Run one test
```bash
go test ./... -count=1 -test.v -test.timeout=20s -p 1 -run TestExtractAuth
```

## Update Go modules
https://github.com/golang/go/wiki/Modules
```bash
go get -u -t ./...
```



# Node.js
## Install nodejs same version as in ci.yml on macOS:
```
brew install nvm
nvm install v15.11.0
```

## Update node
```
nvm install v16.13.0
nvm use v16.13.0
nvm alias default 16
nvm use default 16
```
Then restart (sic!) PC.

## Check for package's updates
ncu [package](https://www.npmjs.com/package/npm-check-updates)
```
npm install -g npm-check-updates
ncu
```

## Fixing fibers issue
```
# npm install --global node-gyp
$ /usr/bin/node /home/nkonev/go_1_11/videochat/frontend/node_modules/fibers/build
# yum groupinstall 'Development Tools'
```



# WebRTC
[In plain English](https://forasoft.com/blog/article/what-is-webrtc-156)
[In plain Russian](http://forasoft.github.io/webrtc-in-plain-russian/)


## Access to video camera (WebRTC) from local network without https from Mobile:

Let's suppose you decided to test videochat in your LAN. You don't have a certificate, you use plain http.

The following steps are going to help you to enable WebRTC for non-https setup (disable browser's protection mechanisms).

## Firefox
1. Install Firefox Beta (`about:config`it is working only in Beta releases and disabled in regular)
2. Open `about:config`
3. Set to true `media.devices.insecure.enabled` and `media.getusermedia.insecure.enabled`
![](./.markdown/mobile-ff-1.jpg)
![](./.markdown/mobile-ff-2.jpg)

## Chrome
1. Configure insecure origins treated as secure
![](./.markdown/mobile-ch-1.jpg)

## Validate turn server installation

Then install on client machine (your PC)
```bash
dnf install coturn-utils
```

Test (Actual value for InternalUserNamE and SeCrEt see in video.yml under turn.auth.credentials key)
```bash
turnutils_uclient -v -u InternalUserNamE -w SeCrEt your.public.ip.address
```

Correct output
```
0: Total connect time is 0
0: 2 connections are completed
1: start_mclient: msz=2, tot_send_msgs=0, tot_recv_msgs=0, tot_send_bytes ~ 0, tot_recv_bytes ~ 0
2: start_mclient: msz=2, tot_send_msgs=3, tot_recv_msgs=3, tot_send_bytes ~ 300, tot_recv_bytes ~ 300
2: start_mclient: tot_send_msgs=10, tot_recv_msgs=10
2: start_mclient: tot_send_bytes ~ 1000, tot_recv_bytes ~ 1000
2: Total transmit time is 2
2: Total lost packets 0 (0.000000%), total send dropped 0 (0.000000%)
2: Average round trip delay 11.500000 ms; min = 11 ms, max = 13 ms
2: Average jitter 0.800000 ms; min = 0 ms, max = 2 ms
```

## Firefox enable video on non-localhost
https://lists.mozilla.org/pipermail/dev-platform/2019-February/023590.html
about:config
media.devices.insecure.enabled


# Alternatives (SFU servers)
* https://github.com/versatica/mediasoup
* https://github.com/medooze/media-server
* https://github.com/meetecho/janus-gateway
* https://github.com/OpenVidu/openvidu

# Alternatives (standalone solutions)
* Zoom
* Skype
* Jitsi
* RocketChat
* Discord
* OpenMeetings
* BigBlueButton


# SDP specification
https://datatracker.ietf.org/doc/html/rfc4566#section-5

https://webrtchacks.com/limit-webrtc-bandwidth-sdp/
https://habr.com/en/company/Voximplant/blog/316840/


# Livekit
## Generate livekit token
```
docker run --rm -e LIVEKIT_KEYS="APIznJxWShGW3Kt: KEUUtCDVRqXk9me0Ok94g8G9xwtnjMeUxfNMy8dow6iA" \
    livekit/livekit-server create-join-token \
    --room "chat100" \
    --identity nkonev
```

## Interesting commits
* [Post v1.1.0 - disable ice lite by default](https://github.com/livekit/livekit/commit/0b630e15b646be8dce6b5cd6770f83f40a02e82d)
* [Improve docker connectivity by using srflx candidates](https://github.com/livekit/livekit/pull/624)
* [Support for custom TURN servers](https://github.com/livekit/livekit/pull/409)
* [Automatic token refresh](https://github.com/livekit/livekit/pull/365)
* [Region aware](https://docs.livekit.io/deploy/distributed/#multi-region-support)
* [Region aware routing](https://github.com/livekit/livekit/pull/135)
* [Region Aware node selection fixes and enhancements](https://github.com/livekit/livekit/pull/141)
* [Dev debug info](https://github.com/livekit/livekit/commit/5fdb6361cdbc48439cfb363a67bb8be5c88330bb)
* [Add pprof endpoint when running in dev mode](https://github.com/livekit/livekit/commit/a933f1513f44f36e025811eaf0a64e15948f92d1)
* [Fixed issues with reconnecting to the same Room object](https://github.com/livekit/client-sdk-js/commit/b0a5f6a271b45e2eda5bf22990c97e8adb07224a)
* [Enable simulcast by default, add jest tests](https://github.com/livekit/client-sdk-js/commit/dcfcd26d117e3d7ca44a83348dfc800e21f8b327)
* [simulcast codecs support](https://github.com/livekit/client-sdk-js/commit/3384ad79984110de4855d8c1bbec50e7b9cb2f47)
* [enable screen share simulcast by default](https://github.com/livekit/client-sdk-js/commit/faa0ad555a2f065af798983d385a1f5d0c65f878)
* [Disable simulcasted screen share for FF](https://github.com/livekit/client-sdk-js/commit/082bd301898b3ec925bc93c752bd7c6892f5c589)
* [Reconnect policy](https://github.com/livekit/client-sdk-js/pull/266)
* [Screensharing quality](https://github.com/livekit/livekit/issues/761)
* [v1.2.3 Make local track publication timeout instead of waiting indefinitely for server response](https://github.com/livekit/client-sdk-js/pull/328)
* [v1.2.5 Decrease publication timeout to 10s and clean local state on failed unpublish attempts](https://github.com/livekit/client-sdk-js/pull/363)
* [Reconnect policy](https://github.com/livekit/client-sdk-js/pull/266)
* [Clear pending track resolver on track unpublish](https://github.com/livekit/client-sdk-js/pull/363)
* [Interrupts and reconnects with Firefox participants](https://github.com/livekit/livekit/issues/1143)
* [Debugging Intermittent timeout errors](https://github.com/livekit/livekit/issues/1070)
* [to avoid using the default Google STUN|TURN servers, you'll need to provide one of the following](https://github.com/livekit/livekit/issues/1299#issuecomment-1413208559)


## Monitoring
```
docker exec -it $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_livekit --filter desired-state=running -q)) sh
watch -n 1 ./livekit-server --config /etc/livekit.yaml list-nodes
```

## Explanation of two peer connections (livekit has their own fork of ion-sfu)
https://github.com/pion/ion-sfu/issues/652#issuecomment-1078364761

## Checking limits
```
cat /proc/$(pgrep livekit)/limits
```

## Known issues of Livekit (Outdated, non-actual)

### Troubleshooting (already taken into account in the video settings)
* Poor quality of screen sharing - a) Disable [simulcast](https://github.com/livekit/livekit/issues/761), b) Increase its resolution
* Connection to livekit is interrupting if at least one participant uses Firefox. Solution is to disable ICE lite in livekit config.
* Duplication of your own video source(camera). Or interrupts on Android Chrome(sic!). The solution is to switch from mobile network to (more stable) Wi-Fi.
#### Reasons of not showing video
* jaeger all-in-one ate too much memory - one of participants didn't see other - restart jaeger.
* Mobile Chrome 101.0.4951.41 - swap it up (e. g. close application and open again) helps when video isn't connected from Mobile Chrome.
* Desktop Firefox - try to reload tab or restart entire browser - it helps when Desktop Firefox isn't able to show video. It can be in long-idled Firefox window.


## Egress
Article https://blog.livekit.io/livekit-universal-egress-launch/
```
curl -i -X PUT 'http://localhost:1237/api/video/1/record/start'
curl -i -X PUT 'http://localhost:1237/api/video/1/record/stop?egressId=EG_6Kf4zXaq7ood'
```



# ion-SFU FAQ (Obsoleted)
https://github.com/pion/ion-sfu/pull/496/files

## Explanation of two peer connections
https://github.com/pion/ion-sfu/issues/652#issuecomment-1078364761
> ion-sfu does not support perfect negotiation, becuase there were some issues on browser implementation, thats why it uses 2 pc, one to publish and other one to subscribe, that way negotiations would be always one sided.

### Tracing
You can match media stream id, media track id in console (F12) and in `about:webrtc`

Peer connection does not have id. [1](https://github.com/w3c/webrtc-pc/issues/1775), [2](https://github.com/pion/webrtc/issues/1250)


## Known issues (Outdated, non-actual)

### Codecs
When Call started in th next sequence
* Desktop Firefox hd vp8 (1)
* Mobile Chrome hd vp8 (2)
* Desktop Firefox hd vp8 (3)


then Firefox (1) won't see video from Firefox (3). If we replace Chrome (2) with Firefox client then problem will be gone.


When Call started in th next sequence
* Desktop Firefox hd vp8 (1)
* Mobile Chrome hd h264 (2)
* Desktop Firefox hd vp8 (3)


then all works ok.


Also it works good when all the devices use the same h264.


### IceLite
When one of participants has the public IP (it's possible) there are no video. I turned on IceLite in config in order to fix it.


### I don't see my image from camera when I connect from mobile
Some mobile operators impede WebRTC traffic.

Solution: try to use Wi-Fi.


### Simulcast
* https://github.com/pion/webrtc/tree/master/examples/simulcast
* https://github.com/pion/ion-sfu/pull/189
* https://github.com/pion/ion-sfu/pull/227
* https://github.com/pion/ion-sdk-flutter/commit/d480792ce89fd1d87dc010f85aafaad8139f8671#diff-29436ed00f4c4d104d7a3a703144724e4dff5b5d01c2b7da70ea54b2ef39b780R65

In `receiver.go` we have isSimulcast: len(track.RID()) > 0, given this and fact that Firefox doesn't sent rid we acn't enable simulcast in Firefox.

But according to https://webrtchacks.com/sfu-simulcast/ H.264/SVC, where scalability capabilities are fully built into the codec itself.

Firefox [doesn't support simulcast for H264](https://bugzilla.mozilla.org/show_bug.cgi?id=1210175)

Firefox [bug about layer order](https://bugzilla.mozilla.org/show_bug.cgi?id=1663368)


## Interesting forks ion-sfu
* https://github.com/edudip/ion-sfu/commits/master
* https://github.com/cryptagon/ion-sfu/commits/master-tandem (With fixing simulcast)

## Generate ports
```python
for x in range(35200, 35401):
    print("""
      - target: %d
        published: %d
        protocol: udp
        mode: host""" % (x, x))
```


# For Github CI
```bash
# get parents
git rev-parse 965b2800^@

# show changed dirs
git diff --dirstat=files,0 HEAD~1 HEAD | sed 's/^[ 0-9.]\+% //g' | cut -d'/' -f1 | uniq

# check is commit contain "[force] string"
git show -s --format=%s dda6c910 | grep -F [force]


# merge commit
./scripts/should-run.sh 965b2800 frontend https://chat.nkonev.name

# with force commit
./scripts/should-run.sh dda6c910 frontend https://chat.nkonev.name
./scripts/should-run.sh dda6c910 aaa https://chat.nkonev.name

# or condition for e2e-test
./scripts/should-run.sh 965b2800 'frontend|aaa' https://chat.nkonev.name && echo "ok" || echo "ko"
```


# Firewalld (not actual, not needed)

## Firewalld help
[Solve no route to host whe invoke host from container by add firewalld rich rule](https://forums.docker.com/t/no-route-to-host-network-request-from-container-to-host-ip-port-published-from-other-container/39063/6)
[Firewalld examples](https://www.rootusers.com/how-to-use-firewalld-rich-rules-and-zones-for-filtering-and-nat/)
```bash
firewall-cmd --permanent --zone=public --list-rich-rules
firewall-cmd --get-default-zone
```

## Add firewall exception on dev
```bash
firewall-cmd --zone=public --add-port=8081/tcp
```

## Open firewall on macOS
```
https://www.macworld.co.uk/how-to/how-open-specific-ports-in-os-x-1010-firewall-3616405/
```

## Add firewall exception on prod (not working, not need)
[link](https://www.digitalocean.com/community/tutorials/how-to-configure-the-linux-firewall-for-docker-swarm-on-centos-7)
```
firewall-cmd --zone=public --add-port=3478/tcp  --permanent
firewall-cmd --zone=public --add-port=3478/udp  --permanent
firewall-cmd --zone=public --add-port=40000-40020/udp  --permanent
firewall-cmd --zone=public --add-port=40000-40020/tcp  --permanent
firewall-cmd --zone=public --add-port=57001-57021/tcp  --permanent
firewall-cmd --zone=public --add-port=57001-57021/udp  --permanent

firewall-cmd --reload

systemctl restart docker

firewall-cmd --list-all-zones
```

## Temporarily allow firewalld ports for usage in local network (not necessary in Fedora)
```
firewall-cmd --zone=public --add-port=8081/tcp
firewall-cmd --zone=public --add-port=3478/tcp
firewall-cmd --zone=public --add-port=3478/udp
firewall-cmd --zone=public --add-port=5000-5100/udp
```


# Keycloak

There is already configured Keycloak in `docker-compose.keycloak.yml`. Run
```bash
docker compose -f docker-compose.keycloak.yml up -d
```

## Configuring Keycloak as an OIDC provider for "Login via Keycloak" in aaa:
1. Login as admin `keycloak_admin`:`admin_password`
2. Create realm `my_realm2`
3. Create client `my_client2` with all defaults (Save) and then, after saving set
Valid redirect URIs
```
http://localhost:8081/*
http://localhost:8060/*
http://localhost:9080/*
```
4. Realm `my_realm2` should be chosen
5. Client scopes -> Create client scope with name `openid` and set `Include in token scope` in the bottom the page
6. Assign this scope onto client
Clients -> `my_client2` -> client scopes -> Add client scope, choose `openid`, press Add, then default
7. Users -> Add user `user2`
8. User's -> Credentials -> Set password `user_password2`, disable temporal
9. Realm roles -> Create role `USER`
10. User's -> Role Mappings -> `Assign roles to user2` -> Choose `Fileter by realm roles` at top left -> add `USER` role
11. Clients -> `my_client2` -> `Capability config` -> enable `Client authentication`, then grab Client Secret from credentials tab

### Login as user1 (get 3 tokens)
```bash
curl -Ss -H 'Content-Type: application/x-www-form-urlencoded' 'http://localhost:8484/realms/my_realm2/protocol/openid-connect/token' -d 'client_id=my_client2&grant_type=password&scope=openid&username=user2&password=user_password2' | jq '.'
```

## Configuring Keycloak for getting users via REST API:
* https://medium.com/@imsanthiyag/introduction-to-keycloak-admin-api-44beb9011f7d
* https://www.keycloak.org/docs-api/22.0.1/rest-api/index.html#_users

### 1. Using admin user (not recommended)
1.1 Get access_token for admin
```bash
curl -Ss -X POST 'http://localhost:8484/realms/master/protocol/openid-connect/token' --header 'Content-Type: application/x-www-form-urlencoded' --data-urlencode 'username=keycloak_admin' --data-urlencode 'password=admin_password' --data-urlencode 'grant_type=password' --data-urlencode 'client_id=admin-cli' | jq 
```


1.2 Request users (do it fast, token expires after 60 sec)
```bash
curl -Ss -H 'Authorization: Bearer ey_PASTE_TOKEN' http://localhost:8484/admin/realms/my_realm2/users | jq
```

### 2. Using User Client Credentials Grant (we go this way)

2.1 How I changed my_client2 to make REST API working:


2.1.1 `Public client not allowed to retrieve service account`


see [SO](https://stackoverflow.com/questions/72086736/python-keycloak-error-public-client-not-allowed-to-retrieve-service-account) 
  -> [site](https://www.appsdeveloperblog.com/keycloak-rest-api-create-a-new-user/)


![1](./.markdown/keycloak_1_sync_client_settings.png)

2.1.2 `Keycloak Get Users returns 403 forbidden`


[SO](https://stackoverflow.com/questions/66452108/keycloak-get-users-returns-403-forbidden/66454728#66454728)


![2](./.markdown/keycloak_2.1_enable_querying_users.png)
![3](./.markdown/keycloak_2.2_enable_querying_users.png)
![4](./.markdown/keycloak_2.3_enable_querying_users.png)
![5](./.markdown/keycloak_2.4_enable_querying_users_result.png)


2.2 Invoke them (do it fast, token expires after 60 sec)
```bash
# Get token (works thanks to imports file)
curl -Ss -X POST 'http://localhost:8484/realms/my_realm2/protocol/openid-connect/token' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'grant_type=client_credentials' \
--data-urlencode 'client_id=my_client2' \
--data-urlencode 'client_secret=z8cr0Nw2z8c7OpSvEix75GgZeDrWJi60' | jq

# Get users
curl -Ss -H 'Authorization: Bearer ey_PASTE_TOKEN' http://localhost:8484/admin/realms/my_realm2/users | jq
```

example of result
```json
[
  {
    "id": "959a4c72-9d37-474b-b718-1d3c9e18a462",
    "username": "user2",
    "firstName": "User",
    "lastName": "Second",
    "email": "user2@example.com",
    "emailVerified": false,
    "createdTimestamp": 1725603344594,
    "enabled": true,
    "totp": false,
    "disableableCredentialTypes": [],
    "requiredActions": [],
    "notBefore": 0,
    "access": {
      "manageGroupMembership": false,
      "view": true,
      "mapRoles": false,
      "impersonate": false,
      "manage": false
    }
  }
]
```

2.3 Getting users in role
To be able to invoke `/admin/realms/{realm_name}/roles/{role_name}/users` assign `view-realm` role, so the final look will be like:

![6](./.markdown/keycloak_2.5_enable_querying_users_in_group_result.png)


Getting all roles
```
curl -Ss -H 'Authorization: Bearer ey_PASTE_TOKEN' http://localhost:8484/admin/realms/my_realm2/roles | jq
```

Getting user-in-roles
```
curl -Ss -H 'Authorization: Bearer ey_PASTE_TOKEN' http://localhost:8484/admin/realms/my_realm2/roles/USER/users | jq
```

Wait for Get multiple users by Ids [#12025](https://github.com/keycloak/keycloak/issues/12025)


## Export data from Keycloak
### 1. exporting (not always importable)
from https://github.com/nkonev/videochat/tree/062aaf2ea58edcffadf6ddf768e289273801492a

```bash
docker compose exec keycloak bash
/opt/keycloak/bin/kc.sh export --file /tmp/realm-export.json --http-management-port 8180 --realm my_realm2
# --http-management-port 8180 to avoid annoying (but not critical) busy port
exit
```
next on host
```bash
docker cp $(docker ps --format {{.Names}} | grep keycloak):/tmp/realm-export.json ./docker/keycloak/realm-export.json
```


## Testing with Keycloak
To test add 3 environment varianbles:
```
spring.security.oauth2.client.registration.keycloak.client-id=my_client2
spring.security.oauth2.client.registration.keycloak.redirect-uri={baseUrl}/api/login/oauth2/code/{registrationId}
spring.security.oauth2.client.provider.keycloak.issuer-uri=http://localhost:8484/auth/realms/my_realm2
```

## Links
* https://habr.com/ru/amp/publications/552346/
* https://keycloak.discourse.group/t/issue-on-userinfo-endpoint-at-keycloak-20/18461/4

* Problem: Keycloak renders 'Invalid parameter: redirect_uri'
* Solution: Set proper [redirect url](https://stackoverflow.com/questions/45352880/keycloak-invalid-parameter-redirect-uri)


* Problem: Keycloak 24+, Unauthorized in browser after entering login and password on Keycloak's page
* Solution: From [here](https://keycloak.discourse.group/t/issue-on-userinfo-endpoint-at-keycloak-20/18461/12)


## spring-boot-keycloak
Using spring boot and Keycloak authorization server
https://habr.com/en/amp/post/552346/

Article about OpenID Connect
https://habr.com/en/post/422765/

Keycloak login/password - see in docker-compose.yml

Open user's keyclock page
http://localhost:8484/auth/realms/my_realm2/account

Open keyclock admin console
http://localhost:8484/ (keycloak_admin:admin_password)

Open protected page
http://localhost:8060/api2/user


# LDAP (Example with OpenDJ)

There is already configured OpenDJ in `docker-compose.opendj.yml`. Run
```bash
docker compose -f docker-compose.opendj.yml up -d
```

## Configuring OpenDJ to provide an example users
1. Download and unzip [OpenDJ](https://github.com/OpenIdentityPlatform/OpenDJ/releases/tag/4.8.0)
2. Run installation script
```bash
./setup 
```
![1](./.markdown/opendj_1.png)
![2](./.markdown/opendj_2.png)


At the end of setup run both server and GUI. Or you can start the server and control panel using
```bash
./bin/start-ds
./bin/control-panel
```
from the server installation folder.


Open users

![3.1](./.markdown/opendj_3.1.png)


Set the password `password` for an user `Aaccf Amar`

![3.2](./.markdown/opendj_3.2.png)

Add the group `MyGroup` and add `user.0` and `user.1` there

![3.3](./.markdown/opendj_3.3.png)

3. Run `aaa` with ldap opendj config
```bash
make run-with-ldap 
```
Full configuration is here: `src/test/resources/config/demo-ldap-opendj.yml`.

4. Login in videochat
![4](./.markdown/opendj_4.png)


## Exporting
Firstly, export ldif to /home/nkonev/example2.ldif

Then add read permission
```bash
chmod a+r /home/nkonev/example2.ldif
```

Then remove excess attributes
```bash
grep -v "entryUUID" /home/nkonev/example2.ldif > /tmp/temp && mv /tmp/temp /home/nkonev/example2.ldif
grep -v "pwdChangedTime" /home/nkonev/example2.ldif > /tmp/temp && mv /tmp/temp /home/nkonev/example2.ldif
grep -v "modifyTimestamp" /home/nkonev/example2.ldif > /tmp/temp && mv /tmp/temp /home/nkonev/example2.ldif
grep -v "modifiersName" /home/nkonev/example2.ldif > /tmp/temp && mv /tmp/temp /home/nkonev/example2.ldif
grep -v "createTimestamp" /home/nkonev/example2.ldif > /tmp/temp && mv /tmp/temp /home/nkonev/example2.ldif
grep -v "creatorsName" /home/nkonev/example2.ldif > /tmp/temp && mv /tmp/temp /home/nkonev/example2.ldif
```

```bash
docker run -it --rm -p 1389:1389 -p 1636:1636 -p 4444:4444 -e ROOT_PASSWORD=password2 -e ROOT_USER_DN='cn=Directory Manager' -e ADD_BASE_ENTRY='' -v /home/nkonev/example2.ldif:/opt/opendj/bootstrap/data/data.ldif openidentityplatform/opendj:4.8.0
```

# Test in browser
Open `http://localhost:8081/chat` in Firefox main and an Anonymous window;
Login as `admin:admin` in main window and as `nikita:password` in the Anonymous window.
Create chat in main window and add `nikita` there.

## Generating password
```bash
sudo yum install -y httpd-tools

# generate password
htpasswd -bnBC 10 "" password | tr -d ':'
```

# Minio
## Show minio content
```
mc stat local/files/chat/111/e4a37493-c6ff-4bd7-9d81-ffc9558af447/0a583bad-23c0-4c3d-8e8d-3a0591653603.jpg
```

## Migrating minio from official to bitnami
Remove 'command' tag from `docker-compose-infra.template.yml`
```
chown -R 1001 /mnt/chat-minio
```

## Rename minio bucket
... by [mirroring it](https://github.com/minio/mc/issues/2619#issuecomment-444545894)
```
mc mb myminio/newbucket
mc mirror myminio/oldbucket myminio/newbucket
mc rm -r --force myminio/oldbucket
```

## Show minio http logs
```
docker exec -it minio sh
mc admin trace -v local
```

## Minio events
```
mc event add local/files arn:minio:sqs::primary:amqp --event put,delete
mc event remove local/files --force
mc event list local/files
```

https://min.io/docs/minio/linux/administration/monitoring/publish-events-to-amqp.html#minio-bucket-notifications-publish-amqp
https://medium.com/@tiwari_nitish/lambda-computing-with-minio-and-kafka-de928897ccdf
```
# already done via config
mc admin config set local/ notify_amqp:PRIMARY \
  url="amqp://videoChat:videoChatPazZw0rd@rabbitmq:5672" \
  exchange="minio-events" \
  exchange_type="direct" \
  durable="on" \
  no_wait="off" \
  auto_deleted="off" \
  delivery_mode="2"
  
mc admin service restart local/
```

Why isn't environment variable MINIO_NOTIFY_KAFKA_ENABLE documented? [#8863](https://github.com/minio/minio/issues/8863)
https://github.com/minio/minio/pull/8864/files


## Attempt to fix (not working)
```
Feb 03 10:24:12 Chesnaught chat-storage[1036]: ERRO[2024-02-03T07:24:12Z]nkonev.name/storage/main.go:110 main.createCustomHTTPErrorHandler.func1() Unhandled error: XMinioStorageFull: Storage backend has reached its minimum free drive threshold. Please delete a few objects to proceed.
Feb 03 10:24:12 Chesnaught chat-storage[1036]:         status code: 507, request id: 17B0497B5CFA7E62, host id: dd9025bab4ad464b049177c95eb6ebf374d3b3fd1af9251148b658df7ac2e3e8  traceId=5c05ddd8d5b1ae02474b165bd3767d94

mc rm --recursive --force --dangerous --incomplete local/files
```


# PostgreSQL
## Open pgCenter
```bash
docker run --network=videochat_backend -it --rm lesovsky/pgcenter:latest pgcenter top -h videochat_postgresql_1 -U chat -d chat
```


## Pgadmin
https://hub.docker.com/r/dcagatay/pwless-pgadmin4

# Elasticsearch
* https://olivere.github.io/elastic/
* https://www.elastic.co/guide/en/elasticsearch/reference/7.17/index.html
* https://www.elastic.co/guide/en/elasticsearch/reference/7.17/explicit-mapping.html
* https://www.elastic.co/guide/en/elasticsearch/reference/7.17/mapping-types.html
```bash
curl 'http://127.0.0.1:28200/chat/_mapping' | jq '.'
curl 'http://127.0.0.1:28200/chat/_doc/3' | jq '.'
```

# Tracing
```
curl -i 'http://localhost:3100/sitemap.xml'
```

# Playwright
https://playwright.dev/docs/intro
```
npx playwright test

# in foreground
npx playwright test --headed --project=chromium --debug

npx playwright test --headed --project=chromium test/login.spec.mjs

npx playwright test --headed --project=chromium -g "login vkontakte and"

# https://habr.com/en/companies/otus/articles/757630/
npx playwright test --ui --project=chromium
```

[Auto-waiting](https://playwright.dev/docs/actionability)

# Droid cam
* https://askubuntu.com/questions/1235731/can-i-use-an-android-phone-as-webcam-for-an-ubuntu-device
* https://play.google.com/store/apps/details?id=com.dev47apps.droidcam

# GraphQL
## Generate GraphQL in go app
About subscriptions https://github.com/99designs/gqlgen/issues/953
```
go install github.com/99designs/gqlgen@v0.17.20
gqlgen generate
```

```bash
go run github.com/99designs/gqlgen --verbose generate
```

## Subscribing
```
subscription {
  subscribe(subscriber:"dodo")
}
```

You need to open it through Traefik
http://localhost:8081/event/playground

```
subscription{
  chatEvents(chatId: 1) {
    eventType
    messageEvent {
      id
      text
      chatId
      ownerId
    }
  }
}

subscription{
  globalEvents {
    eventType
    chatEvent {
      id
      name
      avatar
      avatarBig
      participantIds
      participants {
        id
        login
        avatar
        admin
      }
    }
  }
}
```

# Make a thumbnail using ffmpeg
https://codingdiksha.com/golang-script-to-generate-youtube-video-thumbnails-as-png-and-jpeg-images-in-browser/
https://www.bannerbear.com/blog/how-to-set-a-custom-thumbnail-for-a-video-file-using-ffmpeg/
https://superuser.com/questions/716538/use-ffmpeg-to-send-bmps-to-stdout/716590#716590
```
ffmpeg -y -i 'http://localhost:9000/files/chat/1/154b52fd-cb90-4952-89f3-81a6a119838e/d6e9f11c-b52f-40d5-9bcd-6df3d7231065.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=2SNM25VJ805V7RMK4TM0%2F20221227%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20221227T144627Z&X-Amz-Expires=604800&X-Amz-Security-Token=eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NLZXkiOiIyU05NMjVWSjgwNVY3Uk1LNFRNMCIsImV4cCI6MTY3MjE1NTk2OSwicGFyZW50IjoiQUtJQUlPU0ZPRE5ON0VYQU1QTEUifQ.2KM9yImoL2WWdI_yoSJ6JeB_uHz6PvGrKG1NXugbQAOtFbb7e_SKzM9FvxVFoYieMJoS6F31zI6C3HwsUX4KzA&X-Amz-SignedHeaders=host&versionId=null&X-Amz-Signature=40dc346427e9920af28c7485fe18a2314a7178a79a093b07844fef639945c061' -vf "thumbnail" -frames:v 1 /tmp/ss5.png

ffmpeg -y -i 'http://localhost:9000/files/chat/1/154b52fd-cb90-4952-89f3-81a6a119838e/d6e9f11c-b52f-40d5-9bcd-6df3d7231065.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=2SNM25VJ805V7RMK4TM0%2F20221227%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20221227T144627Z&X-Amz-Expires=604800&X-Amz-Security-Token=eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NLZXkiOiIyU05NMjVWSjgwNVY3Uk1LNFRNMCIsImV4cCI6MTY3MjE1NTk2OSwicGFyZW50IjoiQUtJQUlPU0ZPRE5ON0VYQU1QTEUifQ.2KM9yImoL2WWdI_yoSJ6JeB_uHz6PvGrKG1NXugbQAOtFbb7e_SKzM9FvxVFoYieMJoS6F31zI6C3HwsUX4KzA&X-Amz-SignedHeaders=host&versionId=null&X-Amz-Signature=40dc346427e9920af28c7485fe18a2314a7178a79a093b07844fef639945c061' -vf "thumbnail" -frames:v 1 -c:v bmp -f rawvideo -an - > /tmp/output.bmp

ffmpeg -y -i 'http://localhost:9000/files/chat/1/154b52fd-cb90-4952-89f3-81a6a119838e/d6e9f11c-b52f-40d5-9bcd-6df3d7231065.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=2SNM25VJ805V7RMK4TM0%2F20221227%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20221227T144627Z&X-Amz-Expires=604800&X-Amz-Security-Token=eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NLZXkiOiIyU05NMjVWSjgwNVY3Uk1LNFRNMCIsImV4cCI6MTY3MjE1NTk2OSwicGFyZW50IjoiQUtJQUlPU0ZPRE5ON0VYQU1QTEUifQ.2KM9yImoL2WWdI_yoSJ6JeB_uHz6PvGrKG1NXugbQAOtFbb7e_SKzM9FvxVFoYieMJoS6F31zI6C3HwsUX4KzA&X-Amz-SignedHeaders=host&versionId=null&X-Amz-Signature=40dc346427e9920af28c7485fe18a2314a7178a79a093b07844fef639945c061' -vf "thumbnail" -frames:v 1 -c:v png -f rawvideo -an - > /tmp/output.png
```


# Message embedding
## Case 1 - reply on message
```
update message_chat_1 set 
    embed_message_id = 845, 
    embed_message_type = 'reply_on'
where id = 734;

SELECT 
    m.id, 
    m.text, 
    m.owner_id,
    m.create_date_time, 
    m.edit_date_time, 
    m.file_item_uuid,
    me.id as embedded_message_id,
    me.text as embedded_message_text,
    me.owner_id as embedded_message_owner_id
FROM message_chat_1 m
LEFT JOIN message_chat_1 me 
ON m.embed_message_id = me.id
ORDER BY m.id DESC
LIMIT 10;
```

# Case 2 - message resending
```
insert into message_chat_2 values
    text = 'copy-paste of original',
    owner_id = <re_sender_id>
    embed_message_id = <original_message_id>, 
    embed_chat_id = <original_message_chat_id>, 
    embed_message_type = 'resent'
where id = 734;

using SQL above - message just will not found, but SQL returns all what we need
```

... and on frontend we will understand how to draw using embed_message_type

jsoup analogues for Golang
* https://github.com/PuerkitoBio/goquery
* https://github.com/saopayne/gsoup
* https://github.com/anaskhan96/soup

```
curl -i -X PUT -H 'Content-Type: application/json' -d '{"text": "<a>danger</a> or not"}' --url 'http://localhost:8081/api/chat/public/clean-html-tags'
```

```
curl -i 'http://localhost:8081/api/chat/114/message/8/pin?pin=true' -X PUT -H 'Cookie: VIDEOCHAT_SESSION=MjI3Y2MxOWEtMTcwNC00NDk4LTkzNTItNWI5MzkyMzY5OTY2; VIDEOCHAT_XSRF_TOKEN=15ec7af3-2b30-428f-8472-923a8627ce09'
curl -Ss 'http://localhost:8081/api/chat/114/message/pin' -H 'Cookie: VIDEOCHAT_SESSION=MjI3Y2MxOWEtMTcwNC00NDk4LTkzNTItNWI5MzkyMzY5OTY2; VIDEOCHAT_XSRF_TOKEN=15ec7af3-2b30-428f-8472-923a8627ce09' | jq '.'
```


```
curl -Ss --url 'http://localhost:8081/api/blog' -H 'Cookie: VIDEOCHAT_SESSION=YTRiMzhlNzktY2E2Yi00ZTFkLTkyMWEtNGJkZTIyMDBlYzQx' | jq '.'
```

# Materialdesignicons
https://pictogrammers.github.io/@mdi/font/7.0.96/


# Initial multi page in Vite
https://vitejs.dev/guide/build.html#multi-page-app
http://localhost:8081/front2/blog/

# Makefile
https://sflanders.net/2018/11/06/makefile-basics/
https://stackoverflow.com/questions/448910/what-is-the-difference-between-the-gnu-makefile-variable-assignments-a
https://stackoverflow.com/questions/25185607/whats-the-difference-between-parenthesis-and-curly-bracket-syntax-in-ma

# Testcase for remove incorrect data in video
```
docker exec -it videochat_redis_1 redis-cli -n 4
SADD dials_of_user:1 2
HSET user_call_state:2 userCallOwner 1
```

# Switching aaa onto JWT
* https://www.bezkoder.com/spring-boot-jwt-authentication/ (with changes 2->3)
* https://www.toptal.com/spring/spring-security-tutorial
* https://medium.com/code-with-farhan/spring-security-jwt-authentication-authorization-a2c6860be3cf

# Freemarker
still [does not support](https://stackoverflow.com/questions/76639412/freemarker-jsptaglib-issue-while-upgrading-to-spring-boot-3-and-java-17) `jakarta.*` namespace

It exists a resolved issue for that [FREEMARKER-218](https://issues.apache.org/jira/browse/FREEMARKER-218). It is solved but it is unknown then version 2.3.33 is released

having this, [spring security taglibs](https://docs.spring.io/spring-security/reference/servlet/integrations/jsp-taglibs.html) were removed


# Testing transfer ownership
```sql
insert into user_call_state (token_id, user_id, chat_id, token_taken, owner_token_id, owner_user_id, status)
values
('t1', 1, 100, true, null, null, 'inCall'),
('t2', 2, 100, true, 't1', 1, 'inCall'),
('t3', 3, 100, true, 't1', 1, 'inCall');

update user_call_state set (owner_token_id, owner_user_id) = (
	select owner_token_id, owner_user_id 
	from user_call_state 
	where chat_id = 100 and token_id != 't1' and user_id != 1 order by owner_user_id limit 1
) where chat_id = 100 and token_id != 't1' and user_id != 1;
```

# Mockery
* https://vektra.github.io/mockery/latest/#why-mockery
* https://medium.com/the-sixt-india-blog/mocking-with-mockery-in-golang-949794372e99
```bash
# go install github.com/vektra/mockery/v2@v2.45.1
go get github.com/vektra/mockery/v2/../
mockery --all

# or
go generate ./...
```

```bash
go install github.com/vektra/mockery/v3@v3.3.4
mockery init

```

# Admin's corner
* [jaeger](https://www.jaegertracing.io/docs/1.31/deployment/#ui-base-path)
* [rabbitmq](https://www.rabbitmq.com/docs/management#path-prefix)
* [minio](https://github.com/minio/minio/discussions/15551)
* [pgadmin](https://stackoverflow.com/questions/72567935/run-pgadmin4-on-a-custom-path-with-docker/72694786#72694786)
* [traefik](https://github.com/traefik/traefik/commit/0ec12c7aa7b1344648e8789c489e14ee6063dafc) starting from 3.3


# Opensearch
```bash
curl -Ss -X GET 'http://localhost:9200/_cat/indices'

# by trace id, without jq, but with pretty, to use inside from opensearch container
curl -Ss -X GET 'http://localhost:9200/log/_search?pretty' -H 'Content-Type: application/json' -d'
{
  "size": 1000,
  "query": {
      "match": {
        "trace_id": "6077d4a6a80eacc954d668b8e8edbf5e"
      }
  },
  "sort" : [
    { 
        "@timestamp" : "asc"
    }
  ]
}
'

# last 100 docs, without jq, but with pretty, to use inside from opensearch container
curl -Ss -X GET 'http://localhost:9200/log/_search?pretty' -H 'Content-Type: application/json' -d'
{
  "size": 100,
  "query": {
      "match_all": { }
  },
  "sort" : [
    { 
        "@timestamp" : "desc"
    }
  ]
}
'

# by date https://www.elastic.co/guide/en/elasticsearch/reference/current/api-conventions.html#api-date-math-index-names
docker exec -t $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_opensearch --filter desired-state=running -q)) curl -Ss -X GET 'http://localhost:9200/%3Clog-%7Bnow%2Fd-1d%7D%3E%2C%3Clog-%7Bnow%2Fd%7D%3E/_search?pretty' -H 'Content-Type: application/json' -d'
{
  "size": 10000,
  "query": {
      "match_all": { }
  },
  "sort" : [
    { 
        "@timestamp" : "asc"
    }
  ]
}
' > /tmp/logs.json
less /tmp/logs.json


curl -Ss -X GET 'http://localhost:9200/log-*/_mapping' | jq

# AUTO CLEANING 2
# create a policy to delete old idx
# https://opensearch.org/docs/latest/im-plugin/ism/api/#create-policy
# https://opensearch.org/docs/latest/im-plugin/ism/policies/#example-policy
curl -Ss -X PUT 'http://localhost:9200/_plugins/_ism/policies/delete_old_indexes_policy?pretty' -H 'Content-Type: application/json' -d'
{
  "policy": {
    "description": "delete old indexes",
    "default_state": "hot",
    "schema_version": 1,
    "states": [
      {
        "name": "hot",
        "transitions": [
          {
            "state_name": "delete",
            "conditions": {
              "min_index_age": "1d"
            }
          }
        ]
      },
      {
        "name": "delete",
        "actions": [
          {
            "delete": {}
          }
        ]
      }
    ],
    "ism_template": {
      "index_patterns": ["log-*"],
      "priority": 100
    }
  }
}
'

# AUTO CLEANING 3
# every 1 min (https://opensearch.org/docs/latest/im-plugin/ism/policies/#example-policy)
curl -Ss -X PUT 'http://localhost:9200/_cluster/settings?pretty=true' -H 'Content-Type: application/json' -d'
{
  "persistent" : {
    "plugins.index_state_management.job_interval" : 1
  }
}
'

# by count
curl -Ss -X PUT 'http://localhost:9200/_plugins/_ism/policies/delete_old_indexes_policy?pretty' -H 'Content-Type: application/json' -d'
{
  "policy": {
    "description": "delete old indexes",
    "default_state": "hot",
    "schema_version": 1,
    "states": [
      {
        "name": "hot",
        "transitions": [
          {
            "state_name": "delete",
            "conditions": {
              "min_doc_count": 10
            }
          }
        ]
      },
      {
        "name": "delete",
        "actions": [
          {
            "delete": {}
          }
        ]
      }
    ],
    "ism_template": {
      "index_patterns": ["log-*"],
      "priority": 100
    }
  }
}
'

curl -Ss -X GET 'http://localhost:9200/log/_count'

# also
curl -Ss 'http://localhost:9200/_plugins/_ism/policies?pretty'
curl -i -X PUT 'http://localhost:9200/.opendistro-ism-config'
curl -Ss -X DELETE 'http://localhost:9200/.opendistro-ism-config'
curl -Ss -X DELETE 'http://localhost:9200/_plugins/_ism/policies/delete_old_indexes_policy?pretty'
curl -Ss -X GET 'http://localhost:9200/_plugins/_ism/policies/delete_old_indexes_policy?pretty'

# AUTO CLEANING check, should be "enabled" : true
# explain
curl -Ss -X GET 'http://localhost:9200/_plugins/_ism/explain/log-2025.01.05?pretty'
curl -Ss -X GET 'http://localhost:9200/_plugins/_ism/explain/log-*?pretty'

# Warning!
# It can change from "enabled" : null to "enabled" : true after roughly 5 minutes ! 

# make yellow -> green (https://opster.com/guides/opensearch/opensearch-basics/yellow-status/)
curl -i -X PUT 'http://localhost:9200/log-2025.01.05/_settings' -H 'Content-Type: application/json' -d'
{
    "index" : {
        "number_of_replicas" : 0
    }
}
'

curl -i -X PUT 'http://localhost:9200/.opendistro-ism-config/_settings' -H 'Content-Type: application/json' -d'
{
    "index" : {
        "number_of_replicas" : 0
    }
}
'

# then health will show green
curl -i 'http://localhost:9200/_cluster/health/?level=shards&pretty'

# AUTO CLEANING 1
# should be created beforehand in order to make policy working (policy searches templates)
curl -i -X PUT 'http://localhost:9200/_index_template/log_template' -H 'Content-Type: application/json' -d'
{
  "index_patterns": [
    "log-*"
  ],
  "template": {
    "aliases": {
      "log": {}
    },
    "settings": {
      "number_of_shards": 1,
      "number_of_replicas": 0
    }
  }
}
'

# if an index videochat-2025.01.05 was created before, just remove it
curl -Ss -X DELETE 'http://localhost:9200/log-2025.01.05'

```

# Structured logging in JSON
There are the following mandatory fields every microservice should write
* `@timestamp`
* `message`
* `level` in lower case
* `service` in lower case, which is the name of microservice
* `trace_id`
* `span_id`

# Server bash shortcuts
```bash
journalctl -f -n 1000 CONTAINER_TAG=chat-opensearch
journalctl -f -n 1000 CONTAINER_TAG=chat-jaeger
docker exec -ti $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_redis --filter desired-state=running -q)) redis-cli -n 2
docker exec -ti $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_opensearch --filter desired-state=running -q)) bash

docker exec -ti $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_postgresql-citus-coordinator-1 --filter desired-state=running -q)) psql -U postgres -d postgres
docker exec -ti $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_postgresql-citus-worker-1 --filter desired-state=running -q)) psql -U postgres -d postgres
docker exec -ti $(docker inspect --format "{{.Status.ContainerStatus.ContainerID}}" $(docker service ps VIDEOCHATSTACK_postgresql-citus-worker-2 --filter desired-state=running -q)) psql -U postgres -d postgres
```

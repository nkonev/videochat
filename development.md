# Development

## Firewalld help
[Solve no route to host whe invoke host from container by add firewalld rich rule](https://forums.docker.com/t/no-route-to-host-network-request-from-container-to-host-ip-port-published-from-other-container/39063/6)
[Firewalld examples](https://www.rootusers.com/how-to-use-firewalld-rich-rules-and-zones-for-filtering-and-nat/)
```bash
firewall-cmd --permanent --zone=public --list-rich-rules
firewall-cmd --get-default-zone
```

# Add firewall exception on dev
```bash
firewall-cmd --zone=public --add-port=8081/tcp
```

# Open firewall on macOS
```
https://www.macworld.co.uk/how-to/how-open-specific-ports-in-os-x-1010-firewall-3616405/
```

# Add firewall exception on prod (not working, not need)
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

# Temporarily allow firewalld ports for usage in local network (not necessary in Fedora)
```
firewall-cmd --zone=public --add-port=8081/tcp
firewall-cmd --zone=public --add-port=3478/tcp
firewall-cmd --zone=public --add-port=3478/udp
firewall-cmd --zone=public --add-port=5000-5100/udp
```

[node check updates](https://www.npmjs.com/package/npm-check-updates)

[Error:java: invalid source release: 8](https://stackoverflow.com/a/26009627)

[Reactive, Security, Session MongoDb](https://medium.com/@hantsy/build-a-reactive-application-with-spring-boot-2-0-and-angular-de0ee5837fed)

# AAA Login
```
curl -i 'http://localhost:8060/api/login' \
  -H 'accept: application/json, text/plain, */*' \
  -H 'x-xsrf-token: aa0a1b63-7b5f-480d-9487-d62a48a32899' \
  -H 'content-type: application/x-www-form-urlencoded;charset=UTF-8' \
  -H 'cookie: XSRF-TOKEN=aa0a1b63-7b5f-480d-9487-d62a48a32899' \
  --data-raw 'username=admin&password=admin'
```


```
docker exec -t videochat_postgres_1 pg_dump -U aaa -b --create --column-inserts --serializable-deferrable
```

```
http://localhost:8081/api/user/list?userId=1&userId=-1
```


# Go

## Push down dummy go packages
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


# Firefox enable video on non-localhost
https://lists.mozilla.org/pipermail/dev-platform/2019-February/023590.html
about:config
media.devices.insecure.enabled

# Access to video camera (WebRTC) from local network without https from Mobile:
## Firefox
1. Install Firefox Beta (`about:config`it is working only in Beta releases and disabled in regular)
2. Open `about:config`
3. Set to true `media.devices.insecure.enabled` and `media.getusermedia.insecure.enabled`
![](./.markdown/mobile-ff-1.jpg)
![](./.markdown/mobile-ff-2.jpg)

## Chrome
1. Configure insecure origins treated as secure
![](./.markdown/mobile-ch-1.jpg)

# Validate turn server installation

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

## Egress
Article https://blog.livekit.io/livekit-universal-egress-launch/
```
curl -i -X PUT 'http://localhost:1237/video/1/record/start'
curl -i -X PUT 'http://localhost:1237/video/1/record/stop?egressId=EG_6Kf4zXaq7ood'
```



# ion-SFU FAQ (Obsoleted)
https://github.com/pion/ion-sfu/pull/496/files

## Explaination of two peer connections
https://github.com/pion/ion-sfu/issues/652#issuecomment-1078364761
> ion-sfu does not support perfect negotiation, becuase there were some issues on browser implementation, thats why it uses 2 pc, one to publish and other one to subscribe, that way negotiations would be always one sided.

### Tracing
You can match media stream id, media track id in console (F12) and in `about:webrtc`

Peer connection does not have id. [1](https://github.com/w3c/webrtc-pc/issues/1775), [2](https://github.com/pion/webrtc/issues/1250)


## Known issues

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


## Simulcast
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




# For Github CI
```
git diff --dirstat=files,0 HEAD~1 | sed 's/^[ 0-9.]\+% //g' | cut -d'/' -f1 | uniq
```

# Generate ports
```python
for x in range(35200, 35401):
    print("""
      - target: %d
        published: %d
        protocol: udp
        mode: host""" % (x, x))
```


# Keycloak

Problem: Keycloak renders 'Invalid parameter: redirect_uri'


Solution: Set proper [redirect url](https://stackoverflow.com/questions/45352880/keycloak-invalid-parameter-redirect-uri)

## spring-boot-keycloak
Using spring boot and Keycloak authorization server
https://habr.com/en/amp/post/552346/

Article about OpenID Connect
https://habr.com/en/post/422765/

Keycloak login/password - see in docker-compose.yml

Open user's keyclock page
http://localhost:8484/auth/realms/my_realm/account

Open keyclock admin console
http://localhost:8484/ (keycloak_admin:admin_password)

Open protected page
http://localhost:8060/api2/user

Configuring Keycloak - adding user:
1. Login as admin
2. Manage -> Users -> Add user
3. User's -> Credentials -> Set password, disable temporal
4. User's -> Role Mappings -> add 'USER' role

## Login as user1 (get 3 tokens)
```bash
curl -i -H 'Content-Type: application/x-www-form-urlencoded' 'http://localhost:8484/realms/my_realm/protocol/openid-connect/token' -d 'client_id=my_client&grant_type=password&scope=openid&username=user1&password=user_password'

curl -Ss -H 'Content-Type: application/x-www-form-urlencoded' 'http://localhost:8484/realms/my_realm/protocol/openid-connect/token' -d 'client_id=my_client&grant_type=password&scope=openid&username=user1&password=user_password' | jq '.'
```

## How to save added users to realm-export.json ?
from https://github.com/nkonev/videochat/tree/062aaf2ea58edcffadf6ddf768e289273801492a

## 1. exporting (not always importable)
```bash
docker-compose exec keycloak bash
/opt/keycloak/bin/kc.sh export --file /tmp/realm-export.json --realm my_realm
# don't worry about the busy port
exit
```
next on host
```bash
docker cp $(docker ps --format {{.Names}} | grep keycloak):/tmp/realm-export.json ./export2.json
```

## 2. Next find "users" JSON array. Then find required user's document by their name.

## 3. Append user's document to existing realm-export.json under "users" array.


# Testing with Keycloak
To test add 3 environment varianbles:
```
spring.security.oauth2.client.registration.keycloak.client-id=my_client
spring.security.oauth2.client.registration.keycloak.redirect-uri={baseUrl}/api/login/oauth2/code/{registrationId}
spring.security.oauth2.client.provider.keycloak.issuer-uri=http://localhost:8484/auth/realms/my_realm
```

# Architecture:

![Architecture](./.drawio/exported/app-Page-1.png "Title")


## Start docker-compose
```bash
docker-compose up -d
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

# Show minio content
```
mc stat local/files/chat/111/e4a37493-c6ff-4bd7-9d81-ffc9558af447/0a583bad-23c0-4c3d-8e8d-3a0591653603.jpg
```

# Migrating minio from official to bitnami
Remove 'command' tag from `docker-compose-infra.template.yml`
```
chown -R 1001 /mnt/chat-minio
```

# Rename minio bucket
... by [mirroring it](https://github.com/minio/mc/issues/2619#issuecomment-444545894)
```
mc mb myminio/newbucket
mc mirror myminio/oldbucket myminio/newbucket
mc rm -r --force myminio/oldbucket
```

# Open pgCenter
```bash
docker run --network=videochat_backend -it --rm lesovsky/pgcenter:latest pgcenter top -h videochat_postgresql_1 -U chat -d chat
```

# Working with Elasticsearch
* https://olivere.github.io/elastic/
* https://www.elastic.co/guide/en/elasticsearch/reference/7.17/index.html
* https://www.elastic.co/guide/en/elasticsearch/reference/7.17/explicit-mapping.html
* https://www.elastic.co/guide/en/elasticsearch/reference/7.17/mapping-types.html
```bash
curl 'http://127.0.0.1:28200/chat/_mapping' | jq '.'
curl 'http://127.0.0.1:28200/chat/_doc/3' | jq '.'
```

# Starting server for integration tests
```
cd aaa
export JAVA_HOME=/usr/lib/jvm/bellsoft-java17.x86_64

# https://docs.spring.io/spring-boot/docs/current/maven-plugin/reference/htmlsingle/#goals-run-parameters-details-mainClass
# https://stackoverflow.com/questions/40094423/spring-boot-usetestclasspath-throws-cannotloadbeanclassexception-and-classnotfou/43765880#43765880
./mvnw -Pintegration_test clean spring-boot:run

curl -i -X POST 'http://localhost:9080/recreate-oauth2-mocks'
```

# Playwright
https://playwright.dev/docs/intro
```
npx playwright test

# in foreground
npx playwright test --headed --project=chromium --debug

npx playwright test --headed --project=chromium test/login.spec.mjs

npx playwright test --headed --project=chromium -g "login vkontakte"
```

# Droid cam
* https://askubuntu.com/questions/1235731/can-i-use-an-android-phone-as-webcam-for-an-ubuntu-device
* https://play.google.com/store/apps/details?id=com.dev47apps.droidcam

# Generate GraphQL
About subscriptions https://github.com/99designs/gqlgen/issues/953
```
go install github.com/99designs/gqlgen@v0.17.20
gqlgen generate
```

# Subscribing
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

```
mc event add local/files arn:minio:sqs::primary:amqp --event put,delete
mc event remove local/files --force
mc event list local/files
```

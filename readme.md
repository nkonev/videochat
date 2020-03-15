# Building Go part

## (Re)generate go protobufs
```bash
rm -rf ./user-service/grpc
mkdir ./user-service/grpc || true
docker run -it --rm -v $PWD:/ws -w /ws znly/protoc:0.4.0 --go_out=plugins=grpc:user-service/grpc --plugin=protoc-gen-grpc=/usr/bin/protoc-gen-go -I./protobuf ./protobuf/*.proto
```

## Building
```
(cd frontend; npm i; npm run prod;)

cd user-service
go test ./...
go get github.com/gobuffalo/packr/v2/packr2@v2.0.1
packr2 build
```


# Prepare localhost

First of all set proper java 11
```bash
export JAVA_HOME=/usr/lib/jvm/jre-11
java -version
./mvnw clean package
```

Final architecture:

![Architecture](./.markdown/auth.png "Title")

## Add DNS names
`vim /etc/hosts`

```
127.0.0.1   auth.site.local
127.0.0.1   chat.site.local
127.0.0.1   site.local
```

## Allow container -> host

```bash
su -
firewall-cmd --permanent --zone=public --add-rich-rule='rule family=ipv4 source address="172.27.0.0/16" accept'
service firewalld restart
```

Check
```bash
docker-compose exec traefik sh
wget -O - http://chat.site.local:10000/chat
```

## Start docker-compose
```bash
docker-compose up -d
```

## Start Chat application from IDE
![alt text](./.markdown/chat.png "Title")

# Test auth - in browser

If all configured correctly - you will redirected to authentication page, then after successful authentication you
will get chat's html.

1. Open `http://site.local:8080/chat`
2. Use `tester:tester` for authentication.
3. Assert `Hello World!` is present
4. Click `Json` link
5. Assert `$.helloMessage` is `Hello Nikita Konev`

# Misc

## Firewalld help
[Solve no route to host whe invoke host from container by add firewalld rich rule](https://forums.docker.com/t/no-route-to-host-network-request-from-container-to-host-ip-port-published-from-other-container/39063/6)

[Firewalld examples](https://www.rootusers.com/how-to-use-firewalld-rich-rules-and-zones-for-filtering-and-nat/)
```bash
firewall-cmd --permanent --zone=public --list-rich-rules
firewall-cmd --get-default-zone
```
## Traefik
[dashboard](http://127.0.0.1:8010/dashboard/) (check 'file' tab, non 'docker')
![alt text](./.markdown/traefik.png "Title")

[file provider documentation](https://docs.traefik.io/v1.7/configuration/backends/file/)

## Keycloak
[Admin Console](http://auth.site.local:8844/auth/admin) (Use admin:admin for authentication)

## Exec cli
```bash
docker-compose exec keycloak /opt/jboss/keycloak/bin/jboss-cli.sh --connect
```

[OAuth 2.0 / OpenID Connect (ru)](https://habr.com/ru/post/281406/)

[how to apply authentication to-any web-service in-15 minutes using keycloak](https://medium.com/docker-hacks/how-to-apply-authentication-to-any-web-service-in-15-minutes-using-keycloak-and-keycloak-proxy-e4dd88bc1cd5)

[keycloak-gatekeeper: 'aud' claim and 'client_id' do not match](https://stackoverflow.com/questions/53550321/keycloak-gatekeeper-aud-claim-and-client-id-do-not-match/53627747#53627747)

[user authentication keycloak](https://scalac.io/user-authentication-keycloak-2/)

[Protect Kubernetes Dashboard with OpenID Connect](https://itnext.io/protect-kubernetes-dashboard-with-openid-connect-104b9e75e39c)

### exporting (not always importable)
```bash
docker-compose exec keycloak bash
cd /opt/jboss/keycloak/
bin/standalone.sh -Djboss.socket.binding.port-offset=100 -Dkeycloak.migration.action=export -Dkeycloak.migration.provider=singleFile -Dkeycloak.migration.file=/tmp/export.json
^C
exit
```
next on host
```bash
docker cp $(docker ps --format {{.Names}} | grep keycloak):/tmp/export.json ./docker/export2.json
```

### Token introspection
[token introspection](https://www.keycloak.org/docs/latest/authorization_services/index.html#_service_protection_token_introspection)

### Verify JWT (Id token)
1. Grab kc-access cookie
2. Optional `curl http://auth.site.local:8080/auth/realms/social-sender-realm/protocol/openid-connect/certs | jq`
3. Put JWT to [jwt.io debugger](https://jwt.io/)
4. Go to admin console, Realm Settings and click on Public key with `kid` as in header in `jwt.io`.
5. Add `-----BEGIN PUBLIC KEY-----` and append `-----END PUBLIC KEY-----` to this copied public key and paste resulting key to `Public key` field in `jwt.io`

# TODO
1. HowerFly ?


# Remove guice-cglib warning

Add
```
--add-opens java.base/java.lang=ALL-UNNAMED
```
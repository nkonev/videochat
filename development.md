# Misc

## Firewalld help
[Solve no route to host whe invoke host from container by add firewalld rich rule](https://forums.docker.com/t/no-route-to-host-network-request-from-container-to-host-ip-port-published-from-other-container/39063/6)
[Firewalld examples](https://www.rootusers.com/how-to-use-firewalld-rich-rules-and-zones-for-filtering-and-nat/)
```bash
firewall-cmd --permanent --zone=public --list-rich-rules
firewall-cmd --get-default-zone
```

# Development
[node check updates](https://www.npmjs.com/package/npm-check-updates)

[Error:java: invalid source release: 8](https://stackoverflow.com/a/26009627)

[Reactive, Security, Session MongoDb](https://medium.com/@hantsy/build-a-reactive-application-with-spring-boot-2-0-and-angular-de0ee5837fed)

# AAA Login
```
curl -v 'http://localhost:8060/api/login' -H 'Accept: application/json, text/plain, */*' -H 'Content-Type: application/x-www-form-urlencoded' --data 'username=admin&password=admin'
```


```
docker exec -t videochat_postgres_1 pg_dump -U aaa -b --create --column-inserts --serializable-deferrable
```

```
http://localhost:8081/api/user/list?userId=1&userId=-1
```



# Videochat
http://localhost:8081/public/index_old.html

Push down dummy go packages
```
go list -m -json all
```

Test:
```
go test ./... -count=1
```

# Update Go modules
https://github.com/golang/go/wiki/Modules
```bash
go get -u -t ./...
```

# Sanitizers for Go
* https://github.com/microcosm-cc/bluemonday
* https://github.com/kennygrant/sanitize
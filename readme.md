[![Build Status](https://github.com/nkonev/videochat/workflows/CI%20jobs/badge.svg)](https://github.com/nkonev/videochat/actions)


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

# Known issue
When Call started in th next sequence
* Firefox (1)
* Chrome (2)
* Firefox (3)
then Firefox (1) won't see video from Firefox (3). If we replace Chrome (2) with Firefox client then problem will be gone.

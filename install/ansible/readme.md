```bash
ansible-playbook -i hosts.ini playbook.yaml
```

Get [facts](https://www.digitalocean.com/community/tutorial-series/how-to-write-ansible-playbooks)
```bash
ansible all -i hosts.ini -m setup -a "filter=*ipv4*" -u root
```

Dry-run
```bash
ansible-playbook -i hosts.ini playbook.yaml --check
```

Manual apply
```bash
docker stack deploy --compose-file /opt/videochat/docker-compose-infra.yml VIDEOCHATSTACK
```

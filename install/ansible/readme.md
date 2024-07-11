```bash
pip3 install passlib
sudo dnf install ansible
cd install/ansible
mkdir ~/blog
# copy then edit this both vars and inventory
cp ./roles/initial_install/vars/main.yml.example ~/blog/vars.yml
cp ./inventory.ini.example ~/blog/inventory.ini
# check for community.docker
ansible-galaxy collection list
# run
ansible-playbook -i ~/blog/inventory.ini --extra-vars "@~/blog/vars.yml" playbook.yaml
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
journalctl -n 200 -f CONTAINER_TAG=chat-minio
```

Variables

https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_variables.html

Generate password
```bash
# https://passlib.readthedocs.io/en/stable/lib/passlib.hash.bcrypt.html
python
import passlib
from passlib.hash import bcrypt
bcrypt.using(rounds=10, salt="salt012345678901234567").hash("admin")
```

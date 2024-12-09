# Installation

Tested with Fedora Linux 39, 40 on the local machine and Rocky Linux 9.3, 9.5 on the server.

```bash
# install Ansible and passlib onto the local machine
pip3 install passlib
sudo dnf install ansible

cd install/ansible

# prepare configs
mkdir ~/videochat
# copy then edit this both vars and inventory
cp ./roles/install/vars/main.yml.example ~/videochat/vars.yml
cp ./inventory.ini.example ~/videochat/inventory.ini

# run the full installation on the server
ansible-playbook -i ~/videochat/inventory.ini --extra-vars "@~/videochat/vars.yml" --tags "services,videochat,continuous" playbook.yaml
```

# Just an update
```bash
# only videochat
ansible-playbook -i ~/videochat/inventory.ini --extra-vars "@~/videochat/vars.yml" --extra-vars "image_install_tag=changing" --tags "videochat" playbook.yaml
```

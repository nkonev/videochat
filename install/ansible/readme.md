# Installation

Tested on Rocky Linux 9.3.

```bash
# install Ansible and passlib onto local machine
pip3 install passlib
sudo dnf install ansible

cd install/ansible

# prepare configs
mkdir ~/blog
# copy then edit this both vars and inventory
cp ./roles/install/vars/main.yml.example ~/blog/vars.yml
cp ./inventory.ini.example ~/blog/inventory.ini

# run the full installation
ansible-playbook -i ~/blog/inventory.ini --extra-vars "@~/blog/vars.yml" --tags "services,videochat,continuous" playbook.yaml
```

# Just an update
```bash
# only videochat
ansible-playbook -i ~/blog/inventory.ini --extra-vars "@~/blog/vars.yml" --extra-vars "image_install_tag=changing" --tags "videochat" playbook.yaml
```

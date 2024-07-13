```bash
pip3 install passlib
sudo dnf install ansible
cd install/ansible
mkdir ~/blog
# copy then edit this both vars and inventory
cp ./roles/install/vars/main.yml.example ~/blog/vars.yml
cp ./inventory.ini.example ~/blog/inventory.ini
# run
ansible-playbook -i ~/blog/inventory.ini --extra-vars "@~/blog/vars.yml" --tags "services,videochat,continuous" playbook.yaml
```

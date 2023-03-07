This is the machine project built around a REST client/server model.

machined will run as a daemon and create a UNIX socket for the local client,
machine, to connect and interact with machined.

== Install Prerequisites ==

1. sudo add-apt-repository -y ppa:puzzleos/dev  # for swtpm
1. sudo apt install go || sudo snap install --classic go
1. sudo apt install -y \
        build-essential qemu-system-x86 qemu-utils spice-client-gtk soca
1. sudo usermod --append --groups kvm $USER
1. newgrp kvm  # or logout and login, run 'groups' command to confirm


== Build mcli ==

1. tar xzf mcli-v2-0.9.tar.gz
1. cd mcli-v2
1. make

== Run machined ==

=== Debugging/Testing ===

In a second shell/terminal

1. ./bin/machined

When done, control-c to stop daemon.


=== For hosting/running ===

In a second shell/terminal

1. groups | grep kvm || newgrp kvm
1. systemd-run --user --unit=machined.service --no-block bin/machined
1. systemctl --user status machined.service
1. journalctl --user --follow -u machined.service

When done, `systemctl stop --user machined.service`


== Run machine client ==

./bin machine list

== Starting your first VM ==

Download a live iso, like Ubuntu 22.04

https://releases.ubuntu.com/22.04.2/ubuntu-22.04.2-desktop-amd64.iso

```
$ cat >"vm1.yaml" <<EOF
name: vm1
type: kvm
ephemeral: false
description: A fresh VM booting Ubuntu LiveCD in SecureBoot mode with TPM
config:
  name: vm1
  boot: cdrom
  uefi: true
  tpm: true
  tpm-version: 2.0
  secure-boot: true
  cdrom: ubuntu-22.04.2-desktop-amd64.iso
  disks:
      - file: root-disk.qcow
        type: ssd
        size: 50GiB
EOF
$ ./bin/machine init <vm1.yaml
2023/03/06 22:27:10  info DoCreateMachine Name:rational-pig File:- Edit:false
2023/03/06 22:27:10  info Creating machine...
Got config:
name: vm1
type: kvm
ephemeral: false
description: A fresh VM booting Ubuntu LiveCD in SecureBoot mode with TPM
config:
  name: vm1
  boot: cdrom
  uefi: true
  tpm: true
  tpm-version: 2.0
  secure-boot: true
  cdrom: $HOME/ubuntu-22.04.2-desktop-amd64.iso
  disks:
      - file: root-disk.qcow
        type: ssd
        size: 50GiB
 200 OK
```

Then start and connect to the console or gui

```
$ bin/machine start vm1
200 OK
$ bin/machine gui vm1
```





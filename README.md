## pihole config
Create `/etc/dnsmasq.d/10-pxeserver.conf` with:
```
# inspect the vendor class string and match the text to set the tag
dhcp-vendorclass=UEFI,PXEClient:Arch:00007
dhcp-vendorclass=UEFI64,PXEClient:Arch:00009

# Set the boot file name based on the matching tag from the vendor class (above)
dhcp-boot=net:UEFI,efi64/syslinux.efi,,192.168.1.10
dhcp-boot=net:UEFI64,efi64/syslinux.efi,,192.168.1.10
```
and `pihole restartdns`

## Service (outside docker)

```
cat /etc/systemd/system/pxeserver.service
[Unit]
Description=PXE Server
After=network.target
StartLimitIntervalSec=0
[Service]
Type=simple
Restart=always
RestartSec=1
User=root
ExecStart=/usr/bin/pxeserver

[Install]
WantedBy=multi-user.target
```

(See Dockerfile.pxeserver for exact commands) Create `/static/` directory and
- download ubuntu iso
- extract vmlinuz and initrd
- install and copy bootloader
- copy `config.yaml` to `/etc/pxeserver/config.yaml` and edit hostname and password
- `GOOS=linux go build && scp pxeserver pi@pihole.papro.ca:.`, sudo cp ~pi/pxeserver/ /usr/bin/pxeserver`
- `sudo systemctl restart pxeserver`

## Reference
- https://tutorialedge.net/golang/go-websocket-tutorial/
- https://wiki.syslinux.org/wiki/index.php?title=Doc/pxelinux
- https://wiki.archlinux.org/title/dnsmasq#TFTP_server
- https://discourse.ubuntu.com/t/netbooting-the-live-server-installer/14510
- https://linuxconfig.org/how-to-configure-a-raspberry-pi-as-a-pxe-boot-server
- https://medium.com/@benmorel/creating-a-linux-service-with-systemd-611b5c8b91d6
- `sudo nmap --script broadcast-dhcp-discover`
- https://wiki.syslinux.org/wiki/index.php?title=PXELINUX
- https://docs.docker.com/samples/apt-cacher-ng/
- https://oofhours.com/2022/01/26/geeking-out-network-booting/
- https://forums.fogproject.org/topic/8726/advanced-dnsmasq-techniques
- https://getbootstrap.com/docs/5.2/components/progress/
- https://cloudinit.readthedocs.io/en/latest/topics/examples.html
- https://askubuntu.com/questions/135339/assign-highest-priority-to-my-local-repository/153408#153408
- https://ubuntu.com/server/docs/install/netboot-amd64
- https://ubuntu.com/server/docs/install/autoinstall-reference
- https://cloudinit.readthedocs.io/en/latest/topics/logging.html
- https://ubuntu.com/server/docs/install/autoinstall-reference#reporting
- >>>>>>>> https://askubuntu.com/questions/1235723/automated-20-04-server-installation-using-pxe-and-live-server-image
- https://chris-sanders.github.io/2018-02-02-maas-for-the-home/
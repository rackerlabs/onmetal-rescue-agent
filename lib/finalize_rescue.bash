#!/bin/bash
# Finalize the rescue process
# 
# Reads gzipped, base64 encoded iso file from stdin, and mounts
# as a config drive
#
set -e
set -x

cdMountpoint=${1}
rescueUsername=${2}
rescueHash=${3}

safeRescueUsername=$(printf '%s\n' "$rescueUsername" | sed 's/[[\.*^$/]/\\&/g')
safeRescueHash=$(printf '%s\n' "$rescueHash" | sed 's/[[\.*^$/]/\\&/g')

cdIsoFile=`mktemp`

# Read from stdin
base64 --decode | gzip -cd > ${cdIsoFile}
mkdir -p ${cdMountpoint}
mount -o loop ${cdIsoFile} ${cdMountpoint}
coreos-cloudinit --from-configdrive=${cdMountpoint} --convert-netconf="debian"
umount ${cdMountpoint}
sed -ie "s/^\(${safeRescueUsername}\):[^:]*:\(.*\)/\1:${safeRescueHash}:\2/" /etc/shadow
systemctl restart systemd-networkd.service

# Necessary to set up vlans on a bonded interface
sleep 10
systemctl restart systemd-networkd.service


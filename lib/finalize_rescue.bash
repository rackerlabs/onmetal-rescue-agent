#!/bin/bash
# Finalize the rescue process
#
# Reads gzipped, base64 encoded iso file from stdin, and mounts
# as a config drive
#
if [[ -z "$1" ]] || [[ "$1" = "-h" ]]
then
>&2 cat <<EOF
example usage:
  ./finalize_rescue.bash rescue_user rescue_hash [debug] < config_drive.iso.gz.b64

stdin:
  Accepts base64 encoded, gzipped config drive iso

args:
  rescue_user: Username to use for rescuing
  rescue_hash: Crypt hash to set for rescue_user in /etc/shadow
  debug: If included, do not clean up temp files
EOF
exit 1
fi

set -e
set -x

PATH="${PATH}:/usr/share/oem/bin"

rescueUsername=${1}
rescueHash=${2}
clean="true"
if [[ "${3}" == "debug" ]]
then
    clean="false"
fi

safeRescueUsername=$(printf '%s\n' "$rescueUsername" | sed 's/[[\.*^$/]/\\&/g')
safeRescueHash=$(printf '%s\n' "$rescueHash" | sed 's/[[\.*^$/]/\\&/g')

cdIsoFile=`mktemp --suffix=.cdIsoFile`
cdMountpoint=`mktemp --suffix=.cdMountpoint --directory`
cdWorkdir=`mktemp --suffix=.cdWorkdir --directory`


# Read base64ed/gzipped config drive, and mount somewhere in tmp
base64 --decode | gzip -cd > ${cdIsoFile}
mkdir -p ${cdMountpoint}
mount -o loop ${cdIsoFile} ${cdMountpoint}

# Prepare new config drive
mkdir -p ${cdWorkdir}/openstack/latest

# Copy all content (specificaly the interfaces file)
cp -r ${cdMountpoint}/openstack/content ${cdWorkdir}/openstack

# Select only the cloud config drive fields we want
jq '{network_config, name, hostname, uuid}' ${cdMountpoint}/openstack/latest/meta_data.json > ${cdWorkdir}/openstack/latest/meta_data.json

if [[ $clean == "true" ]]
then
	umount ${cdMountpoint}
	rm ${cdIsoFile}
	rmdir ${cdMountpoint}
fi

# Configure from customized config drive
coreos-cloudinit --from-configdrive=${cdWorkdir} --convert-netconf="debian"

if [[ $clean == "true" ]]
then
	rm -rf ${cdWorkdir}
fi

# Set rescue password based on hash
sed -ie "s/^\(${safeRescueUsername}\):[^:]*:\(.*\)/\1:${safeRescueHash}:\2/" /etc/shadow


# Activate network settings.  Performed twice as a work around for vlans on bonded inferface
systemctl restart systemd-networkd.service
sleep 10
systemctl restart systemd-networkd.service


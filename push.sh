#!/bin/bash

# tar up custom Mattermost distribution package and upload to S3 so the instances can fetch them

BUCKET="spinpunch-puppet"

MATTERMOST_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd $MATTERMOST_DIR

if ! make package; then
    echo "build error, aborting push"
    exit 1
fi

for DIST in dist/mattermost-*-bh*-linux-amd64.tar.gz; do
    UPLOADED_TAR_GZ="battlehouse-$(basename ${DIST})"
    # note: standard Mattermost Puppet install script uses curl for download, so file must be world-readable!
    aws s3 cp "${DIST}" "s3://${BUCKET}/${UPLOADED_TAR_GZ}" --acl=public-read
    echo "new full_url: https://s3.amazonaws.com/${BUCKET}/${UPLOADED_TAR_GZ}"
done

echo "Remember to update battlehouse-infra/puppet/bh_mattermost/manifests/init.pp with full_url!"

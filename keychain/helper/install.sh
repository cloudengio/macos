#!/usr/bin/env bash

if [[ -d /Applications/KeychainHelper.app ]] ; then
    rm -r /Applications/KeychainHelper.app
fi
cp -r ./bundles/KeychainHelper.app /Applications/

#SERVICE_NAME="io.cloudeng.KeychainHelper.xpc"
#XPC_BUNDLE="./bundles/${SERVICE_NAME}"

#mkdir -p ~/Library/XPCServices
#rm -rf ~/Library/XPCServices/"${SERVICE_NAME}"
#cp -r "${XPC_BUNDLE}" ~/Library/XPCServices/ 

#!/usr/bin/env bash
set -e # Exit immediately if a command exits with a non-zero status.

RELEASE="$1"
case "$RELEASE" in
    "debug"|"release")
        ;;
    *)
        if [[ -z "$RELEASE" ]]; then
            RELEASE="debug"
            echo "No release type specified. Defaulting to 'debug'."
        else
            echo "Invalid release type specified. Use 'debug' or 'release'."
            exit 1
        fi
        ;;
esac

echo "Building for release type: ${RELEASE}"
swift build -c "${RELEASE}"

if [[ -z "${SIGNING_IDENTITY}" ]]; then
    echo "No signing identity specified. Please set the SIGNING_IDENTITY environment variable."
    exit 1
fi

# Directory structure for the main app bundle.
BUNDLE_NAME="KeychainHelper.app"
BUNDLE_PATH="./bundles/${BUNDLE_NAME}"
CONTENTS_PATH="${BUNDLE_PATH}/Contents"
RESOURCES_PATH="${CONTENTS_PATH}/Resources"
XPC_SERVICE_PATH="${CONTENTS_PATH}/XPCServices/io.cloudeng.KeychainHelper.xpc"
XPC_SERVICE_CONTENTS_PATH="${XPC_SERVICE_PATH}/Contents"


# Directory structure for the XPC service bundle, it will be installed
# in ~/Library/XPCServices, the only difference is the Info.plist which
# allows using a mach port directly rather than being fully managed by
# launchd.
XPC_MACHPORT_BUNDLE_NAME="io.cloudeng.KeychainHelper.xpc"
XPC_MACHPORT_BUNDLE_PATH="./bundles/${XPC_MACHPORT_BUNDLE_NAME}"
XPC_MACHPORT_CONTENTS_PATH="${XPC_MACHPORT_BUNDLE_PATH}/Contents"


ENTITLEMENTS="keychain_helper.entitlements"
PROVISIONING_PROFILE="keychain_helper.provisionprofile"

# Executable names and paths.
# Location of build artifacts.
BUILD_ROOT=".build/arm64-apple-macosx/${RELEASE}"
BUILD_CLIENT_EXECUTABLE="${BUILD_ROOT}/keychainHelperClient"
BUILD_SERVICE_EXECUTABLE="${BUILD_ROOT}/keychainHelperService"


APP_INFO_PLIST="plists/Info.plist"
APP_ICON_PLIST="plists/AppIcon.plist"
XPC_SERVICE_PLIST="plists/XPCServiceInfo.plist"
XPC_MACHPORT_PLIST="plists/MachPortServiceInfo.plist"

KEYCHAIN_CLIENT_EXECUTABLE="KeychainHelper"
SERVICE_EXECUTABLE="KeychainHelperService"

# --- Clean and Prepare ---
rm -rf "${BUNDLE_PATH}" "${XPC_MACHPORT_BUNDLE_PATH}"
mkdir -p "${RESOURCES_PATH}"

icp() {
    mkdir -p "$(dirname "${2}")"
    echo "Copying ${1} to ${2}"
    cp "${1}" "${2}"
}

# --- Verification ---
plutil -lint "${ENTITLEMENTS}"


# --- Assemble App Bundle ---
echo "Assembling app bundle..."

icp "${APP_INFO_PLIST}" "${CONTENTS_PATH}/Info.plist"
icp "${BUILD_CLIENT_EXECUTABLE}" "${CONTENTS_PATH}/MacOS/${KEYCHAIN_CLIENT_EXECUTABLE}"

icp "${XPC_SERVICE_PLIST}" "${XPC_SERVICE_CONTENTS_PATH}/Info.plist"
icp "${BUILD_SERVICE_EXECUTABLE}" "${XPC_SERVICE_CONTENTS_PATH}/MacOS/${SERVICE_EXECUTABLE}"

#icp "${PROVISIONING_PROFILE}" "${CONTENTS_PATH}/embedded.provisionprofile"
#icp "${PROVISIONING_PROFILE}"  "${XPC_MACHPORT_CONTENTS_PATH}/embedded.provisionprofile"

actool --compile "${RESOURCES_PATH}"  \
    --output-partial-info-plist "${APP_ICON_PLIST}" \
    --platform macosx --minimum-deployment-target 15 \
    --app-icon AppIcon Assets.xcassets

/usr/libexec/PlistBuddy -c "Merge AppIcon.plist" "${CONTENTS_PATH}/Info.plist"

# -- Assemble XPC Service Bundle with mach port support ---
echo "Assembling launchd/XPC bundle..."

icp "${XPC_MACHPORT_PLIST}" "${XPC_MACHPORT_CONTENTS_PATH}/Info.plist"
icp "${BUILD_SERVICE_EXECUTABLE}" "${XPC_MACHPORT_CONTENTS_PATH}/MacOS/${SERVICE_EXECUTABLE}"

# --- Sign ---
for path in  "${BUNDLE_PATH}" "${XPC_MACHPORT_BUNDLE_PATH}"; do
    echo "Signing bundle ${path} with identity: ${SIGNING_IDENTITY}"
    codesign --entitlements "${ENTITLEMENTS}" \
        --force --deep -o runtime --sign "${SIGNING_IDENTITY}" "${path}"
done

# --- Verify ---
for path in  "${BUNDLE_PATH}" "${XPC_MACHPORT_BUNDLE_PATH}"; do
        echo "Verifying signature for ${path}..."
        codesign --verify --deep --display --verbose=4 "${path}"
done

# Need notarization
# -- Check XPC service signature
#spctl --assess --type execute --verbose "${BUNDLE_PATH}"

echo ""
echo "Keychain helper bundles built successfully at: ${BUNDLE_PATH} and ${XPC_MACHPORT_BUNDLE_PATH}"


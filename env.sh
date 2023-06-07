#!/bin/bash
#
export JAVA_HOME=/Users/wsong/Library/Java/JavaVirtualMachines/corretto-11.0.19/Contents/Home
export ANDROID_SDK_ROOT=${HOME}/Library/Android/sdk
export PATH=${ANDROID_SDK_ROOT}/platform-tools:${PATH}
export ANDROID_NDK_ROOT=/opt/homebrew/Caskroom/android-ndk/25c/AndroidNDK9519653.app/Contents/NDK

ANDROID_NDK=/opt/homebrew/share/android-ndk
NDK_BIN=${ANDROID_NDK}/toolchains/llvm/prebuilt/darwin-x86_64/bin/
export CGO_ENABLED=1 
export GOOS=android
export GOARCH=arm64
export CC=${NDK_BIN}/aarch64-linux-android21-clang

"$@"

// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

// JNI implementations of Java native callback methods.

import (
	"unsafe"

	"github.com/tailscale/tailscale-android/jni"
)

// #include <jni.h>
import "C"

var (
	// onVPNPrepared is notified when VpnService.prepare succeeds.
	onVPNPrepared = make(chan struct{}, 1)
	// onVPNClosed is notified when VpnService.prepare fails, or when
	// the a running VPN connection is closed.
	onVPNClosed = make(chan struct{}, 1)
	// onVPNRevoked is notified whenever the VPN service is revoked.
	onVPNRevoked = make(chan struct{}, 1)

	// onConnect receives global IPNService references when
	// a VPN connection is requested.
	onConnect = make(chan jni.Object)
	// onDisconnect receives global IPNService references when
	// disconnecting.
	onDisconnect = make(chan jni.Object)
	// onConnectivityChange is notified every time the network
	// conditions change.
	onConnectivityChange = make(chan bool, 1)

	// onWriteStorageGranted is notified when we are granted WRITE_STORAGE_PERMISSION.
	onWriteStorageGranted = make(chan struct{}, 1)
)

const (
	// Request codes for Android callbacks.
	// requestSignin is for Google Sign-In.
	requestSignin C.jint = 1000 + iota
	// requestPrepareVPN is for when Android's VpnService.prepare
	// completes.
	requestPrepareVPN
)

// resultOK is Android's Activity.RESULT_OK.
const resultOK = -1

//export Java_com_tailscale_ipn_App_onVPNPrepared
func Java_com_tailscale_ipn_App_onVPNPrepared(env *C.JNIEnv, class C.jclass) {
	notifyVPNPrepared()
}

//export Java_com_tailscale_ipn_App_onWriteStorageGranted
func Java_com_tailscale_ipn_App_onWriteStorageGranted(env *C.JNIEnv, class C.jclass) {
	select {
	case onWriteStorageGranted <- struct{}{}:
	default:
	}
}

func notifyVPNPrepared() {
	select {
	case onVPNPrepared <- struct{}{}:
	default:
	}
}

func notifyVPNRevoked() {
	select {
	case onVPNRevoked <- struct{}{}:
	default:
	}
}

func notifyVPNClosed() {
	select {
	case onVPNClosed <- struct{}{}:
	default:
	}
}

//export Java_com_tailscale_ipn_IPNService_connect
func Java_com_tailscale_ipn_IPNService_connect(env *C.JNIEnv, this C.jobject) {
	jenv := (*jni.Env)(unsafe.Pointer(env))
	onConnect <- jni.NewGlobalRef(jenv, jni.Object(this))
}

//export Java_com_tailscale_ipn_IPNService_disconnect
func Java_com_tailscale_ipn_IPNService_disconnect(env *C.JNIEnv, this C.jobject) {
	jenv := (*jni.Env)(unsafe.Pointer(env))
	onDisconnect <- jni.NewGlobalRef(jenv, jni.Object(this))
}

//export Java_com_tailscale_ipn_App_onConnectivityChanged
func Java_com_tailscale_ipn_App_onConnectivityChanged(env *C.JNIEnv, cls C.jclass, connected C.jboolean) {
	select {
	case <-onConnectivityChange:
	default:
	}
	onConnectivityChange <- connected == C.JNI_TRUE
}

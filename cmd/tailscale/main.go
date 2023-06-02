// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"gioui.org/app"
	"inet.af/netaddr"

	"github.com/tailscale/tailscale-android/jni"
	"tailscale.com/ipn"
	"tailscale.com/net/interfaces"
	"tailscale.com/tailcfg"
)

type App struct {
	jvm *jni.JVM
	// appCtx is a global reference to the com.tailscale.ipn.App instance.
	appCtx jni.Object

	store             *stateStore
	logIDPublicAtomic atomic.Value // of string

	// prefs receives new preferences from the backend.
	prefs chan *ipn.Prefs
	// browseURLs receives URLs when the backend wants to browse.
	browseURLs chan string
	// invalidates receives whenever the window should be refreshed.
	invalidates chan struct{}
}

type ExitStatus uint8

const (
	// No exit node selected.
	ExitNone ExitStatus = iota
	// Exit node selected and exists, but is offline or missing.
	ExitOffline
	// Exit node selected and online.
	ExitOnline
)

type Peer struct {
	Label  string
	Online bool
	ID     tailcfg.StableNodeID
}

// UIEvent is an event flowing from the UI to the backend.
type UIEvent interface{}

type RouteAllEvent struct {
	ID tailcfg.StableNodeID
}

type ConnectEvent struct {
	Enable bool
}

type CopyEvent struct {
	Text string
}

type SearchEvent struct {
	Query string
}

type OAuth2Event struct {
	Token *tailcfg.Oauth2Token
}
type SetLoginServerEvent struct {
	URL string
}

// serverOAuthID is the OAuth ID of the tailscale-android server, used
// by GoogleSignInOptions.Builder.requestIdToken.
const serverOAuthID = "744055068597-hv4opg0h7vskq1hv37nq3u26t8c15qk0.apps.googleusercontent.com"

// releaseCertFingerprint is the SHA-1 fingerprint of the Google Play Store signing key.
// It is used to check whether the app is signed for release.
const releaseCertFingerprint = "86:9D:11:8B:63:1E:F8:35:C6:D9:C2:66:53:BC:28:22:2F:B8:C1:AE"

func main() {
	a := &App{
		jvm:         (*jni.JVM)(unsafe.Pointer(app.JavaVM())),
		appCtx:      jni.Object(app.AppContext()),
		browseURLs:  make(chan string, 1),
		prefs:       make(chan *ipn.Prefs, 1),
		invalidates: make(chan struct{}, 1),
	}
	a.store = newStateStore(a.jvm, a.appCtx)
	interfaces.RegisterInterfaceGetter(a.getInterfaces)
	go func() {
		if err := a.runUI(); err != nil {
			fatalErr(err)
		}
	}()
	app.Main()
}

// openURI calls a.appCtx.getContentResolver().openFileDescriptor on uri and
// mode and returns the detached file descriptor.
func (a *App) openURI(uri, mode string) (*os.File, error) {
	var f *os.File
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.appCtx)
		openURI := jni.GetMethodID(env, cls, "openUri", "(Ljava/lang/String;Ljava/lang/String;)I")
		juri := jni.JavaString(env, uri)
		jmode := jni.JavaString(env, mode)
		fd, err := jni.CallIntMethod(env, a.appCtx, openURI, jni.Value(juri), jni.Value(jmode))
		if err != nil {
			return err
		}
		f = os.NewFile(uintptr(fd), "media-store")
		return nil
	})
	return f, err
}

func (a *App) isChromeOS() bool {
	var chromeOS bool
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.appCtx)
		m := jni.GetMethodID(env, cls, "isChromeOS", "()Z")
		b, err := jni.CallBooleanMethod(env, a.appCtx, m)
		chromeOS = b
		return err
	})
	if err != nil {
		panic(err)
	}
	return chromeOS
}

// hostname builds a hostname from android.os.Build fields, in place of a
// useless os.Hostname().
func (a *App) hostname() string {
	var hostname string
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.appCtx)
		getHostname := jni.GetMethodID(env, cls, "getHostname", "()Ljava/lang/String;")
		n, err := jni.CallObjectMethod(env, a.appCtx, getHostname)
		hostname = jni.GoString(env, jni.String(n))
		return err
	})
	if err != nil {
		panic(err)
	}
	return hostname
}

// osVersion returns android.os.Build.VERSION.RELEASE. " [nogoogle]" is appended
// if Google Play services are not compiled in.
func (a *App) osVersion() string {
	var version string
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.appCtx)
		m := jni.GetMethodID(env, cls, "getOSVersion", "()Ljava/lang/String;")
		n, err := jni.CallObjectMethod(env, a.appCtx, m)
		version = jni.GoString(env, jni.String(n))
		return err
	})
	if err != nil {
		panic(err)
	}
	return version
}

// modelName return the MANUFACTURER + MODEL from
// android.os.Build.
func (a *App) modelName() string {
	var model string
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.appCtx)
		m := jni.GetMethodID(env, cls, "getModelName", "()Ljava/lang/String;")
		n, err := jni.CallObjectMethod(env, a.appCtx, m)
		model = jni.GoString(env, jni.String(n))
		return err
	})
	if err != nil {
		panic(err)
	}
	return model
}

// updateNotification updates the foreground persistent status notification.
func (a *App) updateNotification(service jni.Object, state ipn.State, exitStatus ExitStatus, exit Peer) error {
	var msg, title string
	switch state {
	case ipn.Starting:
		title, msg = "Connecting...", ""
	case ipn.Running:
		title = "Connected"
		switch exitStatus {
		case ExitOnline:
			msg = fmt.Sprintf("Exit node: %s", exit.Label)
		default:
			msg = ""
		}
	default:
		return nil
	}
	return jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, service)
		update := jni.GetMethodID(env, cls, "updateStatusNotification", "(Ljava/lang/String;Ljava/lang/String;)V")
		jtitle := jni.JavaString(env, title)
		jmessage := jni.JavaString(env, msg)
		return jni.CallVoidMethod(env, service, update, jni.Value(jtitle), jni.Value(jmessage))
	})
}

// notifyExpiry notifies the user of imminent session expiry and
// returns a new timer that triggers when the user should be notified
// again.
func (a *App) notifyExpiry(service jni.Object, expiry time.Time) *time.Timer {
	if expiry.IsZero() {
		return nil
	}
	d := time.Until(expiry)
	var title string
	const msg = "Reauthenticate to maintain the connection to your network."
	var t *time.Timer
	const (
		aday = 24 * time.Hour
		soon = 5 * time.Minute
	)
	switch {
	case d <= 0:
		title = "Your authentication has expired!"
	case d <= soon:
		title = "Your authentication expires soon!"
		t = time.NewTimer(d)
	case d <= aday:
		title = "Your authentication expires in a day."
		t = time.NewTimer(d - soon)
	default:
		return time.NewTimer(d - aday)
	}
	if err := a.pushNotify(service, title, msg); err != nil {
		fatalErr(err)
	}
	return t
}

func (a *App) notifyFile(uri, msg string) error {
	return jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.appCtx)
		notify := jni.GetMethodID(env, cls, "notifyFile", "(Ljava/lang/String;Ljava/lang/String;)V")
		juri := jni.JavaString(env, uri)
		jmsg := jni.JavaString(env, msg)
		return jni.CallVoidMethod(env, a.appCtx, notify, jni.Value(juri), jni.Value(jmsg))
	})
}

func (a *App) pushNotify(service jni.Object, title, msg string) error {
	return jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, service)
		notify := jni.GetMethodID(env, cls, "notify", "(Ljava/lang/String;Ljava/lang/String;)V")
		jtitle := jni.JavaString(env, title)
		jmessage := jni.JavaString(env, msg)
		return jni.CallVoidMethod(env, service, notify, jni.Value(jtitle), jni.Value(jmessage))
	})
}

func (a *App) setPrefs(prefs *ipn.Prefs) {
	wantRunning := jni.Bool(prefs.WantRunning)
	if err := a.callVoidMethod(a.appCtx, "setTileStatus", "(Z)V", jni.Value(wantRunning)); err != nil {
		fatalErr(err)
	}
	select {
	case <-a.prefs:
	default:
	}
	a.prefs <- prefs
}

func (a *App) setURL(url string) {
	select {
	case <-a.browseURLs:
	default:
	}
	a.browseURLs <- url
}

func (a *App) runUI() error {
	var (
		// activity is the most recent Android Activity reference as reported
		// by Gio ViewEvents.
		activity jni.Object
		// files is list of files from the most recent file sharing intent.
	)
	deleteActivityRef := func() {
		if activity == 0 {
			return
		}
		jni.Do(a.jvm, func(env *jni.Env) error {
			jni.DeleteGlobalRef(env, activity)
			return nil
		})
		activity = 0
	}
	defer deleteActivityRef()
	for {
		select {
		case <-onVPNClosed:
		case <-a.prefs:
		case <-a.browseURLs:
		case <-onVPNPrepared:
		case <-onVPNRevoked:
		case <-a.invalidates:
		}
	}
}

func (a *App) isTV() bool {
	var istv bool
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.appCtx)
		m := jni.GetMethodID(env, cls, "isTV", "()Z")
		b, err := jni.CallBooleanMethod(env, a.appCtx, m)
		istv = b
		return err
	})
	if err != nil {
		fatalErr(err)
	}
	return istv
}

// isReleaseSigned reports whether the app is signed with a release
// signature.
func (a *App) isReleaseSigned() bool {
	var cert []byte
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.appCtx)
		m := jni.GetMethodID(env, cls, "getPackageCertificate", "()[B")
		str, err := jni.CallObjectMethod(env, a.appCtx, m)
		if err != nil {
			return err
		}
		cert = jni.GetByteArrayElements(env, jni.ByteArray(str))
		return nil
	})
	if err != nil {
		fatalErr(err)
	}
	h := sha1.New()
	h.Write(cert)
	fingerprint := h.Sum(nil)
	hex := fmt.Sprintf("%x", fingerprint)
	// Strip colons and convert to lower case to ease comparing.
	wantFingerprint := strings.ReplaceAll(strings.ToLower(releaseCertFingerprint), ":", "")
	return hex == wantFingerprint
}

// attachPeer registers an Android Fragment instance for
// handling onActivityResult callbacks.
func (a *App) attachPeer(act jni.Object) {
	err := a.callVoidMethod(a.appCtx, "attachPeer", "(Landroid/app/Activity;)V", jni.Value(act))
	if err != nil {
		fatalErr(err)
	}
}

func (a *App) prepareVPN(act jni.Object) error {
	return a.callVoidMethod(a.appCtx, "prepareVPN", "(Landroid/app/Activity;I)V",
		jni.Value(act), jni.Value(requestPrepareVPN))
}

func (a *App) invalidate() {
	select {
	case a.invalidates <- struct{}{}:
	default:
	}
}

// progressReader wraps an io.Reader to call a progress function
// on every non-zero Read.
type progressReader struct {
	r        io.Reader
	bytes    int64
	size     int64
	eof      bool
	progress func(n int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	// The request body may be read after http.Client.Do returns, see
	// https://github.com/golang/go/issues/30597. Don't update progress if the
	// file has been read.
	r.eof = r.eof || errors.Is(err, io.EOF)
	if !r.eof && r.bytes < r.size {
		r.progress(int64(n))
		r.bytes += int64(n)
	}
	return n, err
}

func (a *App) signOut() {
}

func (a *App) browseToURL(act jni.Object, url string) {
	if act == 0 {
		return
	}
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		jurl := jni.JavaString(env, url)
		return a.callVoidMethod(a.appCtx, "showURL", "(Landroid/app/Activity;Ljava/lang/String;)V", jni.Value(act), jni.Value(jurl))
	})
	if err != nil {
		fatalErr(err)
	}
}

func (a *App) callVoidMethod(obj jni.Object, name, sig string, args ...jni.Value) error {
	if obj == 0 {
		panic("invalid object")
	}
	return jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, obj)
		m := jni.GetMethodID(env, cls, name, sig)
		return jni.CallVoidMethod(env, obj, m, args...)
	})
}

// activityForView calls View.getContext and returns a global
// reference to the result.
func (a *App) contextForView(view jni.Object) jni.Object {
	if view == 0 {
		panic("invalid object")
	}
	var ctx jni.Object
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, view)
		m := jni.GetMethodID(env, cls, "getContext", "()Landroid/content/Context;")
		var err error
		ctx, err = jni.CallObjectMethod(env, view, m)
		ctx = jni.NewGlobalRef(env, ctx)
		return err
	})
	if err != nil {
		panic(err)
	}
	return ctx
}

// Report interfaces in the device in net.Interface format.
func (a *App) getInterfaces() ([]interfaces.Interface, error) {
	var ifaceString string
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.appCtx)
		m := jni.GetMethodID(env, cls, "getInterfacesAsString", "()Ljava/lang/String;")
		n, err := jni.CallObjectMethod(env, a.appCtx, m)
		ifaceString = jni.GoString(env, jni.String(n))
		return err

	})
	var ifaces []interfaces.Interface
	if err != nil {
		return ifaces, err
	}

	for _, iface := range strings.Split(ifaceString, "\n") {
		// Example of the strings we're processing:
		// wlan0 30 1500 true true false false true | fe80::2f60:2c82:4163:8389%wlan0/64 10.1.10.131/24
		// r_rmnet_data0 21 1500 true false false false false | fe80::9318:6093:d1ad:ba7f%r_rmnet_data0/64
		// mnet_data2 12 1500 true false false false false | fe80::3c8c:44dc:46a9:9907%rmnet_data2/64

		if strings.TrimSpace(iface) == "" {
			continue
		}

		fields := strings.Split(iface, "|")
		if len(fields) != 2 {
			log.Printf("getInterfaces: unable to split %q", iface)
			continue
		}

		var name string
		var index, mtu int
		var up, broadcast, loopback, pointToPoint, multicast bool
		_, err := fmt.Sscanf(fields[0], "%s %d %d %t %t %t %t %t",
			&name, &index, &mtu, &up, &broadcast, &loopback, &pointToPoint, &multicast)
		if err != nil {
			log.Printf("getInterfaces: unable to parse %q: %v", iface, err)
			continue
		}

		newIf := interfaces.Interface{
			Interface: &net.Interface{
				Name:  name,
				Index: index,
				MTU:   mtu,
			},
			AltAddrs: []net.Addr{}, // non-nil to avoid Go using netlink
		}
		if up {
			newIf.Flags |= net.FlagUp
		}
		if broadcast {
			newIf.Flags |= net.FlagBroadcast
		}
		if loopback {
			newIf.Flags |= net.FlagLoopback
		}
		if pointToPoint {
			newIf.Flags |= net.FlagPointToPoint
		}
		if multicast {
			newIf.Flags |= net.FlagMulticast
		}

		addrs := strings.Trim(fields[1], " \n")
		for _, addr := range strings.Split(addrs, " ") {
			ip, err := netaddr.ParseIPPrefix(addr)
			if err == nil {
				newIf.AltAddrs = append(newIf.AltAddrs, ip.IPNet())
			}
		}

		ifaces = append(ifaces, newIf)
	}

	return ifaces, nil
}

func fatalErr(err error) {
	// TODO: expose in UI.
	log.Printf("fatal error: %v", err)
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

const multipleUsersText = "Tailscale can't start due to an Android bug when multiple users are present on this device. " +
	"Please see https://tailscale.com/s/multi-user-bug for more information."

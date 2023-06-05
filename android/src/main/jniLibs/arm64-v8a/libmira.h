/* Code generated by cmd/cgo; DO NOT EDIT. */

/* package github.com/tailscale/tailscale-android/cmd/tailscale */


#line 1 "cgo-builtin-export-prolog"

#include <stddef.h>

#ifndef GO_CGO_EXPORT_PROLOGUE_H
#define GO_CGO_EXPORT_PROLOGUE_H

#ifndef GO_CGO_GOSTRING_TYPEDEF
typedef struct { const char *p; ptrdiff_t n; } _GoString_;
#endif

#endif

/* Start of preamble from import "C" comments.  */


#line 15 "callbacks.go"
 #include <jni.h>

#line 1 "cgo-generated-wrapper"

#line 30 "main.go"

#include <jni.h>
static jint jni_GetJavaVM(JNIEnv *env, JavaVM **jvm) {
	return (*env)->GetJavaVM(env, jvm);
}

#line 1 "cgo-generated-wrapper"


/* End of preamble from import "C" comments.  */


/* Start of boilerplate cgo prologue.  */
#line 1 "cgo-gcc-export-header-prolog"

#ifndef GO_CGO_PROLOGUE_H
#define GO_CGO_PROLOGUE_H

typedef signed char GoInt8;
typedef unsigned char GoUint8;
typedef short GoInt16;
typedef unsigned short GoUint16;
typedef int GoInt32;
typedef unsigned int GoUint32;
typedef long long GoInt64;
typedef unsigned long long GoUint64;
typedef GoInt64 GoInt;
typedef GoUint64 GoUint;
typedef size_t GoUintptr;
typedef float GoFloat32;
typedef double GoFloat64;
#ifdef _MSC_VER
#include <complex.h>
typedef _Fcomplex GoComplex64;
typedef _Dcomplex GoComplex128;
#else
typedef float _Complex GoComplex64;
typedef double _Complex GoComplex128;
#endif

/*
  static assertion to make sure the file is being used on architecture
  at least with matching size of GoInt.
*/
typedef char _check_for_64_bit_pointer_matching_GoInt[sizeof(void*)==64/8 ? 1:-1];

#ifndef GO_CGO_GOSTRING_TYPEDEF
typedef _GoString_ GoString;
#endif
typedef void *GoMap;
typedef void *GoChan;
typedef struct { void *t; void *v; } GoInterface;
typedef struct { void *data; GoInt len; GoInt cap; } GoSlice;

#endif

/* End of boilerplate cgo prologue.  */

#ifdef __cplusplus
extern "C" {
#endif

extern void Java_com_tailscale_ipn_App_onVPNPrepared(JNIEnv* env, jclass class);
extern void Java_com_tailscale_ipn_App_onWriteStorageGranted(JNIEnv* env, jclass class);
extern void Java_com_tailscale_ipn_IPNService_connect(JNIEnv* env, jobject this);
extern void Java_com_tailscale_ipn_IPNService_disconnect(JNIEnv* env, jobject this);
extern void Java_com_tailscale_ipn_App_onConnectivityChanged(JNIEnv* env, jclass cls, jboolean connected);
extern void Java_com_tailscale_ipn_App_initGO(JNIEnv* env, jobject ctx);
extern void Java_com_tailscale_ipn_IPNActivity_testJVM(JNIEnv* env, jobject ctx);

#ifdef __cplusplus
}
#endif

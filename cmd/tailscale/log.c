#include <android/log.h>
#define LOG_TAG "mylib"
#define LOGD(...) __android_log_print(ANDROID_LOG_DEBUG, LOG_TAG ,  __VA_ARGS__)

void mylogf(const char *cMsg)
{
	LOGD(cMsg);
}
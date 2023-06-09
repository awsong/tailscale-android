package com.tailscale.ipn.ddauth;

import android.app.Activity;
import android.content.Intent;
import android.os.Bundle;
import android.text.TextUtils;
import com.android.dingtalk.openauth.utils.DDAuthConstant;
import android.util.Log;

public class DDAuthActivity extends Activity {

   @Override
   public void onCreate(Bundle bstate) {
      super.onCreate(bstate);

      handleIntent();
      Intent intent = getIntent();
      String authCode = intent.getStringExtra(DDAuthConstant.CALLBACK_EXTRA_AUTH_CODE);
      String state = intent.getStringExtra(DDAuthConstant.CALLBACK_EXTRA_STATE);
      String error = intent.getStringExtra(DDAuthConstant.CALLBACK_EXTRA_ERROR);

      Log.i("Dingtalk", "authCode: " + authCode + " state: " + state);
      if (!TextUtils.isEmpty(authCode) && !TextUtils.isEmpty(state)) {
         // 授权成功
      } else {
         // 授权失败
      }
      finish();
   }

   @Override
   public void onNewIntent(Intent i) {
      setIntent(i);
      Log.i("Dingtalkk", "called from here");
      handleIntent();
   }

   private void handleIntent() {
      Intent it = getIntent();
      String act = it.getAction();
      String[] texts;
      // Uri[] uris;
      if ("com.tailscale.ipn.AUTH".equals(act)) {
         Log.i("Dingtalkk", "INTENT invokedddddd ");
      } else {
      }
   }
}

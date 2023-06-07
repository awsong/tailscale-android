import androidx.appcompat.app.AppCompatActivity;
import androidx.annotation.Nullable;
import android.content.Intent;
import android.os.Bundle;
import android.text.TextUtils;
import com.android.dingtalk.openauth.utils.DDAuthConstant;

public class DDAuthActivity extends AppCompatActivity {

   @Override
   protected void onCreate(@Nullable Bundle savedInstanceState) {
      super.onCreate(savedInstanceState);

      Intent intent = getIntent();
      String authCode = intent.getStringExtra(DDAuthConstant.CALLBACK_EXTRA_AUTH_CODE);
      String state = intent.getStringExtra(DDAuthConstant.CALLBACK_EXTRA_STATE);
      String error = intent.getStringExtra(DDAuthConstant.CALLBACK_EXTRA_ERROR);

      if (!TextUtils.isEmpty(authCode) && !TextUtils.isEmpty(state)) {
         // 授权成功
      } else {
         // 授权失败
      }
      finish();
   }
}

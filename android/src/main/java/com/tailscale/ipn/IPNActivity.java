// Copyright (c) 2021 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package com.tailscale.ipn;

import android.app.Activity;
import android.content.res.AssetFileDescriptor;
import android.content.res.Configuration;
import android.content.Intent;
import android.database.Cursor;
import android.os.Bundle;
import android.provider.OpenableColumns;
import android.net.Uri;
import android.content.pm.PackageManager;

import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.webkit.WebResourceRequest;
import android.util.Log;
import android.view.View;
import android.widget.Button;
import android.widget.TextView;

import java.util.List;
import java.util.ArrayList;

import org.gioui.GioView;

import com.android.dingtalk.openauth.AuthLoginParam;
import com.android.dingtalk.openauth.IDDAuthApi;
import com.android.dingtalk.openauth.DDAuthApiFactory;

public final class IPNActivity extends Activity {
	final static int WRITE_STORAGE_RESULT = 1000;

	private GioView view;

	@Override
	public void onCreate(Bundle state) {
		super.onCreate(state);
		// view = new GioView(this);
		// setContentView(view);
		setContentView(R.layout.activity_main);
		// loginDingtalk();

		Button button = findViewById(R.id.button);

		button.setOnClickListener(new View.OnClickListener() {
			public void onClick(View v) {
				WebView myWebView = (WebView) findViewById(R.id.webview);
				myWebView.setWebViewClient(new WebViewClient() {
					@Override
					public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) {
						Uri uri = request.getUrl();
						if (uri.toString().startsWith("https://www.example.com")) {
							Intent intent = new Intent("com.tailscale.ipn.AUTH", uri);
							if (intent.resolveActivity(getPackageManager()) != null) {
								startActivity(intent);
							} else {
								Log.d("SomeActivity", "No Activity found to handle Intent");
							}
							return true;
						}
						return false;
					}
				});
				myWebView.setVisibility(View.VISIBLE);
				myWebView.loadUrl("https://chi.matesafe.cn/test.html");
			}
		});

		handleIntent();
	}

	private void loginDingtalk() {
		AuthLoginParam.AuthLoginParamBuilder builder = AuthLoginParam.AuthLoginParamBuilder.newBuilder();
		builder.appId("dingmaup5nlhpi7ixgqz");
		builder.redirectUri("https://sdp.matesafe.cn/issuer/callback");
		builder.responseType("code");
		builder.scope("openid");
		builder.nonce("myNonce");
		builder.state("state");
		builder.prompt("consent");
		IDDAuthApi authApi = DDAuthApiFactory.createDDAuthApi(getApplicationContext(), builder.build());
		authApi.authLogin();
	}

	@Override
	public void onNewIntent(Intent i) {
		setIntent(i);
		handleIntent();
	}

	private void handleIntent() {
		Intent it = getIntent();
		String act = it.getAction();
		String[] texts;
		Uri[] uris;
		if (Intent.ACTION_SEND.equals(act)) {
			uris = new Uri[] { it.getParcelableExtra(Intent.EXTRA_STREAM) };
			texts = new String[] { it.getStringExtra(Intent.EXTRA_TEXT) };
		} else if (Intent.ACTION_SEND_MULTIPLE.equals(act)) {
			List<Uri> extraUris = it.getParcelableArrayListExtra(Intent.EXTRA_STREAM);
			uris = extraUris.toArray(new Uri[0]);
			texts = new String[uris.length];
		} else {
			return;
		}
		String mime = it.getType();
		int nitems = uris.length;
		String[] items = new String[nitems];
		String[] mimes = new String[nitems];
		int[] types = new int[nitems];
		String[] names = new String[nitems];
		long[] sizes = new long[nitems];
		int nfiles = 0;
		for (int i = 0; i < uris.length; i++) {
			String text = texts[i];
			Uri uri = uris[i];
			if (text != null) {
				types[nfiles] = 1; // FileTypeText
				names[nfiles] = "file.txt";
				mimes[nfiles] = mime;
				items[nfiles] = text;
				// Determined by len(text) in Go to eliminate UTF-8 encoding differences.
				sizes[nfiles] = 0;
				nfiles++;
			} else if (uri != null) {
				Cursor c = getContentResolver().query(uri, null, null, null, null);
				if (c == null) {
					// Ignore files we have no permission to access.
					continue;
				}
				int nameCol = c.getColumnIndex(OpenableColumns.DISPLAY_NAME);
				int sizeCol = c.getColumnIndex(OpenableColumns.SIZE);
				c.moveToFirst();
				String name = c.getString(nameCol);
				long size = c.getLong(sizeCol);
				types[nfiles] = 2; // FileTypeURI
				mimes[nfiles] = mime;
				items[nfiles] = uri.toString();
				names[nfiles] = name;
				sizes[nfiles] = size;
				nfiles++;
			}
		}
		App.onShareIntent(nfiles, types, mimes, items, names, sizes);
	}

	@Override
	public void onRequestPermissionsResult(int reqCode, String[] perms, int[] grants) {
		switch (reqCode) {
			case WRITE_STORAGE_RESULT:
				if (grants.length > 0 && grants[0] == PackageManager.PERMISSION_GRANTED) {
					App.onWriteStorageGranted();
				}
		}
	}

	/*
	 * @Override
	 * public void onDestroy() {
	 * view.destroy();
	 * super.onDestroy();
	 * }
	 * 
	 * @Override
	 * public void onStart() {
	 * super.onStart();
	 * view.start();
	 * }
	 * 
	 * @Override
	 * public void onStop() {
	 * view.stop();
	 * super.onStop();
	 * }
	 * 
	 * @Override
	 * public void onConfigurationChanged(Configuration c) {
	 * super.onConfigurationChanged(c);
	 * view.configurationChanged();
	 * }
	 * 
	 * @Override
	 * public void onLowMemory() {
	 * super.onLowMemory();
	 * view.onLowMemory();
	 * }
	 * 
	 * @Override
	 * public void onBackPressed() {
	 * if (!view.backPressed())
	 * super.onBackPressed();
	 * }
	 */
}

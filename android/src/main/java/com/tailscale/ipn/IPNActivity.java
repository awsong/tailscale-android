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

import android.os.AsyncTask;
import android.os.Bundle;
import android.view.View;
import android.widget.Button;
import android.widget.TextView;
//import androidx.appcompat.app.AppCompatActivity;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;

import java.util.List;
import java.util.ArrayList;

public final class IPNActivity extends Activity {
	final static int WRITE_STORAGE_RESULT = 1000;
	TextView label;

	@Override
	public void onCreate(Bundle state) {
		super.onCreate(state);

		// TODO: my UI entry
		setContentView(R.layout.activity_main);

		label = findViewById(R.id.label);
		Button button = findViewById(R.id.button);

		button.setOnClickListener(new View.OnClickListener() {
			public void onClick(View v) {
				new FetchContentTask().execute("https://www.baidu.com");
			}
		});
		handleIntent();
	}

	private class FetchContentTask extends AsyncTask<String, Void, String> {
		protected String doInBackground(String... urls) {
			try {
				URL url = new URL(urls[0]);
				HttpURLConnection urlConnection = (HttpURLConnection) url.openConnection();
				try {
					BufferedReader bufferedReader = new BufferedReader(
							new InputStreamReader(urlConnection.getInputStream()));
					StringBuilder stringBuilder = new StringBuilder();
					String line;
					while ((line = bufferedReader.readLine()) != null) {
						stringBuilder.append(line).append("\n");
					}
					bufferedReader.close();
					// testJVM();
					return getCacheDir() + stringBuilder.toString();
				} finally {
					urlConnection.disconnect();
				}
			} catch (Exception e) {
				e.printStackTrace();
			}
			return null;
		}

		protected void onPostExecute(String content) {
			label.setText(content);
		}
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

	// private native void testJVM();
}

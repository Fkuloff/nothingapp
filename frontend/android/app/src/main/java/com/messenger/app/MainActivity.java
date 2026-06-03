package com.messenger.app;

import android.content.Intent;
import android.os.Bundle;

import com.getcapacitor.BridgeActivity;
import com.getcapacitor.PluginHandle;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        // Register our custom plugins BEFORE super.onCreate so the Capacitor
        // bridge sees them during boot.
        registerPlugin(DownloadsPlugin.class);
        registerPlugin(UpdaterPlugin.class);
        registerPlugin(ShareTargetPlugin.class);
        super.onCreate(savedInstanceState);
    }

    // singleTask: shares arriving while running come here, not via onCreate.
    @Override
    public void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        if (getBridge() == null) return;
        PluginHandle handle = getBridge().getPlugin("ShareTarget");
        if (handle != null && handle.getInstance() instanceof ShareTargetPlugin) {
            ((ShareTargetPlugin) handle.getInstance()).handleIntent(intent);
        }
    }
}

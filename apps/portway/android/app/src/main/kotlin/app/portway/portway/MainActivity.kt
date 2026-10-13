package app.portway.portway

import android.content.Intent
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private var confirmationChannel: MethodChannel? = null
    private fun isConfirmation(intent: Intent?): Boolean =
        intent?.data?.toString() == "portway://auth/confirmed"

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        confirmationChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "app.portway/auth")
        confirmationChannel?.setMethodCallHandler { call, result ->
            if (call.method == "initialConfirmation") result.success(isConfirmation(intent))
            else result.notImplemented()
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        if (isConfirmation(intent)) confirmationChannel?.invokeMethod("confirmed", null)
    }
}

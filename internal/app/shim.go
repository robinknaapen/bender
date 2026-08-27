package app

// notificationShim is injected into every service webview before any page
// script runs. It replaces window.Notification with a stand-in that posts
// the notification to Go (which raises a native one) and reports the
// permission as granted, so sites don't gate on a prompt that native
// WebView2 would never show usefully. Service-worker notifications bypass
// this; the NotificationReceived COM event can cover those later.
const notificationShim = `(() => {
	const post = (title, body) =>
		window.chrome.webview.postMessage({ type: "notify", data: { title: String(title), body: String(body || "") } });
	class ShimNotification {
		constructor(title, options) {
			post(title, options && options.body);
			this.title = String(title);
		}
		close() {}
		addEventListener() {}
		removeEventListener() {}
		static requestPermission(callback) {
			if (callback) callback("granted");
			return Promise.resolve("granted");
		}
	}
	Object.defineProperty(ShimNotification, "permission", { get: () => "granted" });
	Object.defineProperty(ShimNotification, "maxActions", { get: () => 0 });
	window.Notification = ShimNotification;
})();`

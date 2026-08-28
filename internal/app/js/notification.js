(() => {
	const Native = window.Notification;
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
			// Forward to the real API so the origin gets genuinely
			// granted permission (auto-allowed by the host, persisted
			// per profile) — service-worker notifications need it.
			try { Native && Native.requestPermission().catch(() => {}); } catch (e) {}
			if (callback) callback("granted");
			return Promise.resolve("granted");
		}
	}
	Object.defineProperty(ShimNotification, "permission", { get: () => "granted" });
	Object.defineProperty(ShimNotification, "maxActions", { get: () => 0 });
	window.Notification = ShimNotification;
})();
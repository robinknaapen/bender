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
})();

/* Icon resolution: pick the page's best static icon and post it as a
   data URI. apple-touch-icon is preferred (large, and never the badged
   canvas favicon some services swap in — those are data: URIs, which
   are skipped). WebView2's own favicon event stays as the fallback. */
(() => {
	const post = (uri) => window.chrome.webview.postMessage({ type: "icon", data: { uri } });
	const resolve = () => {
		const links = [...document.querySelectorAll(
			'link[rel~="icon"], link[rel="apple-touch-icon"], link[rel="apple-touch-icon-precomposed"]',
		)].filter((l) => l.href && !l.href.startsWith("data:"));
		if (!links.length) return;
		const score = (l) => {
			let s = l.rel.toLowerCase().includes("apple-touch-icon") ? 1000 : 0;
			const m = /(\d+)x/.exec((l.sizes && l.sizes.value) || "");
			return s + (m ? Math.min(+m[1], 512) : 0);
		};
		links.sort((a, b) => score(b) - score(a));
		fetch(links[0].href)
			.then((r) => (r.ok ? r.blob() : Promise.reject(new Error(r.status))))
			.then((b) => new Promise((res, rej) => {
				const fr = new FileReader();
				fr.onload = () => res(fr.result);
				fr.onerror = rej;
				fr.readAsDataURL(b);
			}))
			.then((uri) => { if (uri.startsWith("data:image/")) post(uri); })
			.catch(() => {});
	};
	addEventListener("load", () => {
		resolve();
		// SPAs often install their real icon links well after load.
		setTimeout(resolve, 10000);
	});
})();`

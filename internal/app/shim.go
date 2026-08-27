package app

// notificationShim is injected into every service webview before any page
// script runs. It replaces window.Notification with a stand-in that posts
// the notification to Go (which raises a native one) and reports the
// permission as granted, so sites don't gate on a prompt that native
// WebView2 would never show usefully. Service-worker notifications bypass
// this; the NotificationReceived COM event can cover those later.
const notificationShim = `(() => {
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
})();`

// iconResolver is injected into every service webview.
const iconResolver = `/* Icon resolution: pick the page's best static icon and post it as a
   data URI. The web-app manifest usually carries the largest art
   (192/512px), then apple-touch-icon, then the biggest link icon.
   data: favicons are skipped — that's the badged canvas favicon some
   services swap in. WebView2's own favicon event stays as fallback. */
(() => {
	const post = (uri) => window.chrome.webview.postMessage({ type: "icon", data: { uri } });
	const fromManifest = async () => {
		const link = document.querySelector('link[rel="manifest"]');
		if (!link || !link.href) return null;
		const m = await fetch(link.href).then((r) => (r.ok ? r.json() : null)).catch(() => null);
		if (!m || !Array.isArray(m.icons)) return null;
		const size = (i) => Math.max(0, ...String(i.sizes || "").split(" ").map((s) => parseInt(s) || 0));
		const best = [...m.icons].sort((a, b) => size(b) - size(a))[0];
		return best && best.src ? new URL(best.src, link.href).href : null;
	};
	const fromLinks = () => {
		const links = [...document.querySelectorAll(
			'link[rel~="icon"], link[rel="apple-touch-icon"], link[rel="apple-touch-icon-precomposed"]',
		)].filter((l) => l.href && !l.href.startsWith("data:"));
		const score = (l) => {
			let s = l.rel.toLowerCase().includes("apple-touch-icon") ? 1000 : 0;
			const m = /(\d+)x/.exec((l.sizes && l.sizes.value) || "");
			return s + (m ? Math.min(+m[1], 512) : 0);
		};
		return links.sort((a, b) => score(b) - score(a)).map((l) => l.href);
	};
	const resolve = async () => {
		// Best first; a candidate can fail to fetch (CORS on CDN-hosted
		// manifest icons), so fall through until one succeeds.
		const candidates = [await fromManifest(), ...fromLinks()]
			.filter((u) => u && !u.startsWith("data:"));
		for (const url of candidates) {
			try {
				const blob = await fetch(url).then((r) => (r.ok ? r.blob() : Promise.reject(new Error(r.status))));
				// Rasterize through a canvas: normalizes every format the
				// browser can decode (ico included) to PNG for the host.
				const bmp = await createImageBitmap(blob);
				const n = Math.min(256, Math.max(bmp.width, bmp.height));
				if (!n) continue;
				const canvas = document.createElement("canvas");
				canvas.width = canvas.height = n;
				canvas.getContext("2d").drawImage(bmp, 0, 0, n, n);
				bmp.close();
				post(canvas.toDataURL("image/png"));
				return;
			} catch (e) {}
		}
	};
	addEventListener("load", () => {
		resolve();
		// SPAs often install their real icon links well after load.
		setTimeout(resolve, 10000);
	});
})();`

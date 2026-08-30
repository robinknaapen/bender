// The only script in bender's own pages: receive rendered HTML from Go,
// and report interactions back. Clicked elements carry their message in
// data-msg; forms declare a type in data-msg-type and post their fields.
// Everything else is Go's problem.
(() => {
	const host = window.chrome.webview;
	const mount = document.getElementById("mount");
	const modal = document.getElementById("modal-mount");

	host.addEventListener("message", (e) => {
		const m = e.data;
		if (!m) return;
		if (m.type === "render") mount.innerHTML = m.data.html;
		if (m.type === "render-modal") modal.innerHTML = m.data.html;
	});

	// Modal conveniences the markup can't express: Esc and clicking the
	// backdrop itself (not its children) dismiss.
	document.addEventListener("keydown", (e) => {
		if (e.key === "Escape" && modal.childElementCount > 0) {
			host.postMessage({ type: "close-settings" });
		}
	});
	document.addEventListener("mousedown", (e) => {
		if (e.target.id === "modal-backdrop") {
			host.postMessage({ type: "close-settings" });
		}
	});

	document.addEventListener("click", (e) => {
		const el = e.target.closest("[data-msg]");
		if (!el) return;
		e.preventDefault();
		host.postMessage(JSON.parse(el.dataset.msg));
	});

	document.addEventListener("submit", (e) => {
		const form = e.target.closest("form[data-msg-type]");
		if (!form) return;
		e.preventDefault();
		host.postMessage({
			type: form.dataset.msgType,
			data: Object.fromEntries(new FormData(form)),
		});
	});

	host.postMessage({ type: "ready" });
})();

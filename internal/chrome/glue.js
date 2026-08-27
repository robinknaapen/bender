// The only script in the chrome page: receive rendered sidebar HTML from
// Go, and report clicks back. Everything else is Go's problem.
(() => {
	const host = window.chrome.webview;
	const mount = document.getElementById("mount");

	host.addEventListener("message", (e) => {
		const m = e.data;
		if (m && m.type === "render") mount.innerHTML = m.data.html;
	});

	document.addEventListener("click", (e) => {
		const el = e.target.closest("[data-service-id]");
		if (!el) return;
		e.preventDefault();
		host.postMessage({ type: "activate", data: { serviceId: Number(el.dataset.serviceId) } });
	});

	host.postMessage({ type: "ready" });
})();

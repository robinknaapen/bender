(() => {
	const post = (count, dot) =>
		window.chrome.webview.postMessage({ type: "badge", data: { count, dot } });
	const readers = [
		() => { // Mattermost: unread channels get .unread, mentions a numeric .badge
			const root = document.getElementById("sidebar-left") || document.getElementById("SidebarContainer");
			if (!root || !root.querySelector(".SidebarChannel")) return null;
			let count = 0;
			for (const b of root.querySelectorAll(".SidebarChannel .badge"))
				count += parseInt(b.textContent, 10) || 0;
			const dot = !!root.querySelector(".SidebarChannel.unread, .unread-title");
			return { count, dot };
		},
	];
	let last = "";
	const scan = () => {
		for (const read of readers) {
			let b = null;
			try { b = read(); } catch (e) {}
			if (!b) continue;
			const key = b.count + ":" + b.dot;
			if (key !== last) { last = key; post(b.count, b.dot); }
			return;
		}
	};
	addEventListener("load", () => setInterval(scan, 2000));
})();
package app

// testServicePage is the built-in "Test" service shown in debug builds:
// buttons to exercise the notification pipe (instant and delayed, so you
// can switch away first) and the title-based badge parsing.
const testServicePage = `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><title>Test</title><style>
	body { font-family: system-ui, sans-serif; background: #18181b; color: #e4e4e7;
		display: flex; flex-direction: column; gap: 12px; align-items: flex-start; padding: 24px; }
	button { font: inherit; background: #3f3f46; color: inherit; border: 0; border-radius: 8px;
		padding: 10px 16px; cursor: pointer; }
	button:hover { background: #52525b; }
	small { color: #a1a1aa; }
</style></head><body>
	<h1>bender test service</h1>
	<button onclick='new Notification("bender", {body: "instant notification"})'>Notify now</button>
	<button onclick='setTimeout(() => new Notification("bender", {body: "delayed notification"}), 5000)'>
		Notify in 5s <small>(switch away first)</small></button>
	<button onclick='document.title = "(" + (Math.floor(Math.random()*98)+1) + ") Test"'>Set badge</button>
	<button onclick='document.title = "Test"'>Clear badge</button>
	<small>Notification.permission = <span id="p"></span></small>
	<script>document.getElementById("p").textContent = Notification.permission + " (" + Notification.name + ")";</script>
</body></html>`

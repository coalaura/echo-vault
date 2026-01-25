(() => {
	const StorageKey = "echo_vault_token",
		VolumeKey = "echo_vault_volume",
		VideoExtensions = ["mp4", "webm", "mov", "m4v", "mkv"],
		Resolutions = [
			[4320, "8K"],
			[2880, "5K"],
			[2160, "4K"],
			[2048, "2K-DCI"],
			[1600, "WQXGA"],
			[1536, "QXGA"],
			[1440, "QHD"],
			[1200, "WUXGA"],
			[1080, "FHD"],
			[960, "qHD"],
			[900, "900P"],
			[800, "WXGA"],
			[768, "XGA"],
			[720, "HD"],
			[600, "SVGA"],
			[576, "PAL"],
			[480, "SD"],
			[240, "NTSC"],
		];

	const $loginView = document.getElementById("login-view"),
		$dashboardView = document.getElementById("dashboard-view"),
		$modalView = document.getElementById("modal-view"),
		$modalTag = document.getElementById("modal-tag"),
		$loginForm = document.getElementById("login-form"),
		$apiToken = document.getElementById("api-token"),
		$loginError = document.getElementById("login-error"),
		$gallery = document.getElementById("gallery"),
		$emptyState = document.getElementById("empty-state"),
		$loader = document.getElementById("loader"),
		$dropOverlay = document.getElementById("drop-overlay"),
		$searchWrapper = document.getElementById("search-wrapper"),
		$searchInput = document.getElementById("search-input"),
		$fileInput = document.getElementById("file-input"),
		$uploadBtn = document.getElementById("upload-trigger"),
		$logoutBtn = document.getElementById("logout-btn"),
		$totalSize = document.getElementById("total-size"),
		$totalCount = document.getElementById("total-count"),
		$versionTags = document.querySelectorAll(".version-tag"),
		$modalViewContent = $modalView.querySelector(".modal-content"),
		$modalViewBackdrop = $modalView.querySelector(".modal-backdrop"),
		$modalTagBackdrop = $modalTag.querySelector(".modal-backdrop"),
		$tagSelect = document.getElementById("tag-select"),
		$tagCloseBtn = document.getElementById("tag-close-btn"),
		$tagUpdateBtn = document.getElementById("tag-update-btn");

	let $notifyArea;

	const State = {
		alive: true,
		token: localStorage.getItem(StorageKey),
		volume: parseFloat(localStorage.getItem(VolumeKey)),
		page: 1,
		busy: 0,
		hasMore: true,
		query: "",
		cache: new Map(),
		stats: {
			size: 0,
			count: 0,
		},
		config: {
			noBlur: false,
			ignoreSafety: [],
		},
		controllers: {
			query: null,
			sse: null,
		},
	};

	if (!Number.isFinite(State.volume) || State.volume < 0 || State.volume > 1) {
		State.volume = 0.75;
	}

	$searchInput.value = "";

	function generateId() {
		return Math.random().toString(16).substring(2, 8);
	}

	async function init() {
		$notifyArea = document.createElement("div");

		$notifyArea.id = "notification-area";

		document.body.appendChild($notifyArea);

		await fetchServerInfo();

		if (State.token) {
			verifyToken(State.token);
		} else {
			switchView("login");
		}

		setupEvents();
	}

	function switchView(viewName) {
		$loginView.classList.add("hidden");
		$dashboardView.classList.add("hidden");

		if (viewName === "login") {
			$loginView.classList.remove("hidden");
		} else {
			$dashboardView.classList.remove("hidden");
		}
	}

	function showNotification(message, type = "info") {
		const toast = document.createElement("div");

		toast.className = `notification ${type}`;
		toast.textContent = message;

		$notifyArea.appendChild(toast);

		requestAnimationFrame(() => {
			setTimeout(() => {
				toast.classList.add("fade-out");

				toast.addEventListener("animationend", () => toast.remove());
			}, 3000);
		});
	}

	function updateStatsUI() {
		$totalSize.textContent = formatBytes(State.stats.size);
		$totalSize.title = formatBytesFull(State.stats.size);
		$totalCount.textContent = State.stats.count.toLocaleString();
	}

	function renderBatch(items, method = "append") {
		const fragment = document.createDocumentFragment(),
			list = Array.isArray(items) ? items : [items];

		list.forEach(item => {
			State.cache.set(item.hash, item);

			const existingNode = document.getElementById(`echo-${item.hash}`);

			if (existingNode) {
				updateEchoNode(existingNode, item);

				return;
			}

			const node = createEchoNode(item);

			updateEchoNode(node, item);

			fragment.appendChild(node);
		});

		if (method === "prepend") {
			$gallery.prepend(fragment);
		} else {
			$gallery.appendChild(fragment);
		}
	}

	function makeActionButton(hash, text, action, cls = "") {
		const btn = document.createElement("button");

		btn.className = `action-btn ${cls}`;
		btn.dataset.action = action;
		btn.dataset.hash = hash;
		btn.textContent = text;

		return btn;
	}

	function createEchoNode(item) {
		const isVideo = VideoExtensions.includes(item.extension);

		const card = document.createElement("div");

		card.id = `echo-${item.hash}`;
		card.className = "echo-card";
		card.dataset.hash = item.hash;

		const link = document.createElement("a");

		link.href = item.url;
		link.target = "_blank";
		link.className = "echo-link";

		const loader = document.createElement("div");

		loader.className = "media-loader";
		loader.innerHTML = '<span class="spinner"></span>';

		link.appendChild(loader);

		let media;

		const onLoad = () => {
			loader.remove();

			media.classList.add("loaded");
		};

		const onError = () => {
			loader.remove();

			const err = document.createElement("div");

			err.className = "media-error";
			err.textContent = "FAILED";

			link.appendChild(err);
		};

		if (isVideo) {
			media = document.createElement("video");

			media.className = "echo-media";
			media.muted = true;
			media.loop = true;
			media.src = item.url;

			const badge = document.createElement("div");

			badge.className = "type-badge";
			badge.textContent = "▶";

			card.appendChild(badge);

			media.addEventListener("loadeddata", () => {
				badge.textContent = formatDuration(media.duration);

				onLoad();
			});

			card.addEventListener("mouseenter", () => media.play());

			card.addEventListener("mouseleave", () => {
				media.pause();

				media.currentTime = 0;
			});
		} else {
			media = document.createElement("img");

			media.className = "echo-media";
			media.loading = "lazy";
			media.src = item.url;

			media.addEventListener("load", onLoad);
		}

		media.addEventListener("error", onError);

		link.appendChild(media);

		card.appendChild(link);

		const actions = document.createElement("div");

		actions.className = "echo-actions";

		actions.append(makeActionButton(item.hash, "TAG", "tag"), makeActionButton(item.hash, "COPY", "copy"), makeActionButton(item.hash, "DEL", "delete", "delete"));

		const info = document.createElement("div");

		info.className = "echo-info";

		const dateSpan = document.createElement("span");

		dateSpan.className = "meta-date";

		const sizeSpan = document.createElement("span");

		sizeSpan.className = "meta-size";

		info.append(dateSpan, sizeSpan);

		card.append(actions, info);

		return card;
	}

	function createUploadingNode(file, uploadId) {
		const isImage = file.type.startsWith("image/"),
			isVideo = file.type.startsWith("video/");

		const card = document.createElement("div");

		card.id = `uploading-${uploadId}`;
		card.className = "echo-card uploading";

		const link = document.createElement("div");

		link.className = "echo-link";

		const overlay = document.createElement("div");

		overlay.className = "upload-overlay";
		overlay.innerHTML = '<span class="spinner"></span><span class="upload-text">UPLOADING...</span>';

		link.appendChild(overlay);

		if (isImage || isVideo) {
			const objectUrl = URL.createObjectURL(file);

			let media;

			if (isVideo) {
				media = document.createElement("video");

				media.className = "echo-media loaded";
				media.muted = true;
				media.src = objectUrl;

				const badge = document.createElement("div");

				badge.className = "type-badge";
				badge.textContent = "▶";

				card.appendChild(badge);
			} else {
				media = document.createElement("img");

				media.className = "echo-media loaded";
				media.src = objectUrl;
			}

			media.dataset.objectUrl = objectUrl;

			link.appendChild(media);
		} else {
			const placeholder = document.createElement("div");

			placeholder.className = "upload-placeholder";
			placeholder.textContent = file.name.split(".").pop()?.toUpperCase() || "FILE";

			link.appendChild(placeholder);
		}

		card.appendChild(link);

		const info = document.createElement("div");

		info.className = "echo-info";

		const nameSpan = document.createElement("span");

		nameSpan.className = "meta-date";
		nameSpan.textContent = "Uploading...";

		const sizeSpan = document.createElement("span");

		sizeSpan.className = "meta-size";
		sizeSpan.textContent = formatBytes(file.size);

		info.append(nameSpan, sizeSpan);

		card.appendChild(info);

		return card;
	}

	function removeUploadingNode(uploadId) {
		const node = document.getElementById(`uploading-${uploadId}`);

		if (!node) {
			return;
		}

		const media = node.querySelector("[data-object-url]");

		if (media?.dataset.objectUrl) {
			URL.revokeObjectURL(media.dataset.objectUrl);
		}

		node.remove();
	}

	function replaceUploadingNode(uploadId, item) {
		const placeholder = document.getElementById(`uploading-${uploadId}`);

		State.cache.set(item.hash, item);

		const existingNode = document.getElementById(`echo-${item.hash}`);

		if (existingNode) {
			updateEchoNode(existingNode, item);

			if (placeholder) {
				removeUploadingNode(uploadId);
			}

			return;
		}

		const realNode = createEchoNode(item);

		updateEchoNode(realNode, item);

		if (!placeholder) {
			$gallery.prepend(realNode);

			return;
		}

		const blobMedia = placeholder.querySelector(".echo-media"),
			realMedia = realNode.querySelector(".echo-media");

		if (blobMedia && realMedia && blobMedia.tagName === "IMG" && realMedia.tagName === "IMG") {
			const blobUrl = blobMedia.dataset.objectUrl || blobMedia.src,
				serverUrl = realMedia.src;

			realMedia.src = blobUrl;

			const loader = realNode.querySelector(".media-loader");

			if (loader) {
				loader.remove();
			}

			realMedia.classList.add("loaded");

			placeholder.replaceWith(realNode);

			const tmp = new Image();

			tmp.onload = () => {
				realMedia.src = serverUrl;

				URL.revokeObjectURL(blobUrl);
			};

			tmp.onerror = () => {
				URL.revokeObjectURL(blobUrl);
			};

			tmp.src = serverUrl;
		} else {
			placeholder.replaceWith(realNode);

			removeUploadingNode(uploadId);
		}
	}

	function updateEchoNode(node, item) {
		const shouldBlur = !State.config.noBlur && item.safety && item.safety !== "ok" && !State.config.ignoreSafety.includes(item.safety);

		node.classList.remove("blurred", "safety-suggestive", "safety-explicit", "safety-violence", "safety-sensitive");

		if (shouldBlur) {
			node.classList.add("blurred", `safety-${item.safety}`);
		}

		let simBadge = node.querySelector(".echo-similarity");

		if (item.similarity) {
			if (!simBadge) {
				simBadge = document.createElement("div");

				simBadge.className = "echo-similarity";

				node.appendChild(simBadge);
			}

			simBadge.textContent = `${Math.round(item.similarity * 100)}% MATCH`;
		} else if (simBadge) {
			simBadge.remove();
		}

		const dateSpan = node.querySelector(".meta-date");

		if (dateSpan) {
			dateSpan.textContent = formatDate(item.timestamp);
		}

		const sizeSpan = node.querySelector(".meta-size");

		if (sizeSpan) {
			sizeSpan.textContent = `${formatBytes(item.upload_size)} 🡒 ${formatBytes(item.size)}`;
		}
	}

	function openModal(hash) {
		const item = State.cache.get(hash);

		if (!item) {
			return;
		}

		const isVideo = VideoExtensions.includes(item.extension);

		$modalViewContent.innerHTML = "";
		$modalViewContent.style.width = "";

		const infoBtn = document.createElement("button");

		infoBtn.className = "media-info-btn";
		infoBtn.textContent = "INFO";
		infoBtn.title = "View description";

		const descPanel = document.createElement("div");

		descPanel.className = "media-description closed";

		infoBtn.addEventListener("click", async event => {
			event.stopPropagation();

			if (!descPanel.classList.contains("closed")) {
				descPanel.classList.add("closed");

				infoBtn.classList.remove("active");

				return;
			}

			if (descPanel.dataset.loaded === "true") {
				descPanel.classList.remove("closed");

				infoBtn.classList.add("active");

				return;
			}

			infoBtn.classList.add("loading");

			try {
				const response = await fetchWithAuth(`/echo/${hash}`);

				if (!response.ok) {
					throw new Error("Failed to load");
				}

				const data = await response.json();

				if (data?.description) {
					descPanel.textContent = data.description;
					descPanel.dataset.loaded = "true";

					descPanel.classList.remove("closed");

					infoBtn.classList.add("active");
				} else {
					showNotification("No description available", "info");
				}
			} catch {
				showNotification("Failed to load description", "error");
			} finally {
				infoBtn.classList.remove("loading");
			}
		});

		let media;

		const meta = document.createElement("div");

		meta.className = "media-meta";

		const updateMeta = () => {
			const nw = isVideo ? media.videoWidth : media.naturalWidth,
				nh = isVideo ? media.videoHeight : media.naturalHeight,
				rect = media.getBoundingClientRect();

			if (rect.width > 0) {
				$modalViewContent.style.width = `${Math.ceil(rect.width) + 2}px`;
			}

			if (nw && nh) {
				const tag = getResolutionTag(nw, nh);

				meta.textContent = `${tag ? `${tag} // ` : ""}${nw}x${nh} // ${formatBytes(item.size)}`;
			}
		};

		if (isVideo) {
			media = document.createElement("video");

			media.src = item.url;
			media.volume = State.volume;
			media.controls = true;
			media.autoplay = true;

			media.addEventListener("volumechange", () => {
				State.volume = media.volume;

				localStorage.setItem(VolumeKey, State.volume);
			});

			media.addEventListener("loadeddata", updateMeta);
		} else {
			media = document.createElement("img");

			media.src = item.url;

			media.addEventListener("load", updateMeta);
		}

		media.addEventListener("click", () => {
			descPanel.classList.add("closed");

			infoBtn.classList.remove("active");
		});

		$modalViewContent.append(media, meta, infoBtn, descPanel);

		$modalView.classList.remove("hidden");
	}

	function closeModals() {
		$modalTag.classList.add("hidden");
		$modalView.classList.add("hidden");

		$modalViewContent.innerHTML = "";
	}

	function viewTagModal(hash) {
		const item = State.cache.get(hash),
			node = document.getElementById(`echo-${hash}`);

		if (!item || node?.classList.contains("processing")) {
			return;
		}

		$modalTag.classList.remove("hidden");
		$modalTag.dataset.hash = hash;
		$tagSelect.value = item.safety || "auto";
	}

	async function fetchServerInfo() {
		try {
			const response = await fetch("/info"),
				data = await response.json();

			if (!data?.version) {
				throw new Error("invalid response");
			}

			$versionTags.forEach(tag => {
				tag.textContent = data.version;
			});

			if (data.queries) {
				$searchWrapper.classList.remove("hidden");
			}

			if (!data.blur) {
				State.config.noBlur = true;
			}

			if (data.ignore?.length) {
				State.config.ignoreSafety = data.ignore;
			}
		} catch (err) {
			console.error(`Failed to fetch info: ${err}`);
		}
	}

	async function verifyToken(token) {
		try {
			const response = await fetchWithAuth("/verify", token);

			if (response.status !== 200) {
				throw new Error("Unauthorized");
			}

			State.token = token;

			localStorage.setItem(StorageKey, token);

			switchView("dashboard");

			loadEchos().then(setupSSE);
		} catch {
			logout(false);

			$loginError.classList.remove("hidden");
		}
	}

	function logout(clearUi) {
		State.token = null;
		State.page = 1;
		State.hasMore = true;
		State.query = "";
		State.cache.clear();

		$searchInput.value = "";

		localStorage.removeItem(StorageKey);

		if (clearUi) {
			$gallery.innerHTML = "";

			$loginError.classList.add("hidden");
		}

		switchView("login");
	}

	async function loadEchos() {
		if (!State.hasMore || State.busy > 0) {
			return;
		}

		State.controllers.query?.abort();

		const controller = new AbortController();

		State.controllers.query = controller;

		$loader.classList.remove("hidden");

		let aborted = false;

		try {
			const endpoint = State.query ? `/query/${State.page}?q=${encodeURIComponent(State.query)}` : `/echos/${State.page}`;

			const response = await fetchWithAuth(endpoint, null, {
				signal: controller.signal,
			});

			if (!response.ok) {
				throw new Error(await parseResponseError(response));
			}

			const data = await response.json();

			if (data?.echos?.length) {
				$emptyState.classList.add("hidden");

				State.stats.size = data.size || 0;
				State.stats.count = data.count || 0;

				updateStatsUI();

				renderBatch(data.echos, "append");

				State.page++;
			} else {
				State.hasMore = false;

				if (State.page === 1) {
					$emptyState.classList.remove("hidden");
				}
			}
		} catch (error) {
			aborted = controller.signal.aborted;

			if (!aborted) {
				showNotification(error.message, "error");
			}
		} finally {
			if (!aborted) {
				State.controllers.query = null;

				$loader.classList.add("hidden");
			}
		}
	}

	async function handleUpload(file) {
		const uploadId = generateId();

		const uploadingNode = createUploadingNode(file, uploadId);

		$emptyState.classList.add("hidden");

		$gallery.prepend(uploadingNode);

		const formData = new FormData();

		formData.append("upload", file);

		try {
			const response = await fetchWithAuth(`/upload?id=${uploadId}`, null, {
				method: "POST",
				body: formData,
			});

			if (!response.ok) {
				throw new Error(await parseResponseError(response));
			}

			const data = await response.json();

			if (!data) {
				throw new Error("invalid response");
			}

			// Handle potentially multiple uploads in response, though typically one
			const items = Array.isArray(data.echo) ? data.echo : [data.echo];

			if (items.length > 0) {
				replaceUploadingNode(uploadId, items[0]);

				if (items.length > 1) {
					renderBatch(items.slice(1), "prepend");
				}
			} else {
				removeUploadingNode(uploadId);
			}

			showNotification("Upload complete", "success");
		} catch (err) {
			removeUploadingNode(uploadId);

			if ($gallery.children.length === 0) {
				$emptyState.classList.remove("hidden");
			}

			showNotification(err.message, "error");
		}
	}

	async function updateTag(hash, tag) {
		const node = document.getElementById(`echo-${hash}`);

		if (!node || node.classList.contains("processing")) {
			return;
		}

		closeModals();

		State.busy++;

		node.classList.add("processing");

		let body;

		if (tag === "auto") {
			body = {
				action: "re_tag",
			};
		} else {
			body = {
				action: "set_safety",
				safety: tag,
			};
		}

		try {
			const response = await fetchWithAuth(`/echos/${hash}`, null, {
				method: "PUT",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify(body),
			});

			if (!response.ok) {
				throw new Error(await parseResponseError(response));
			}

			const data = await response.json();

			if (!data) {
				throw new Error("invalid response");
			}

			renderBatch(data.echo, "update");

			const newTag = data.echo.safety || "ok",
				ignored = newTag !== "ok" && (State.config.noBlur || State.config.ignoreSafety.includes(newTag));

			showNotification(`Re-Tag: ${newTag} ${ignored ? "(ignored)" : ""}`, "success");
		} catch (err) {
			showNotification(err.message, "error");
		} finally {
			node.classList.remove("processing");

			State.busy--;
		}
	}

	async function deleteEcho(hash, noConfirm) {
		const node = document.getElementById(`echo-${hash}`);

		if (!node || node.classList.contains("processing")) {
			return;
		}

		if (!noConfirm && !confirm("Delete this echo?")) {
			return;
		}

		State.busy++;

		node.classList.add("processing");

		try {
			const response = await fetchWithAuth(`/echos/${hash}`, null, {
				method: "DELETE",
			});

			if (!response.ok) {
				throw new Error(await parseResponseError(response));
			}

			removeLocalEcho(hash);

			showNotification("Echo deleted", "success");
		} catch (error) {
			node.classList.remove("processing");

			showNotification(error.message, "error");
		} finally {
			State.busy--;
		}
	}

	function removeLocalEcho(hash) {
		const item = State.cache.get(hash),
			node = document.getElementById(`echo-${hash}`);

		if (item) {
			State.cache.delete(hash);
		}

		if (node) {
			node.remove();
		}

		if ($gallery.children.length === 0) {
			$emptyState.classList.remove("hidden");
		}
	}

	function handleServerEvent(data) {
		if (!data || typeof data.type === "undefined") {
			return;
		}

		const { type, echo, hash, size, count, id } = data,
			targetHash = hash || echo?.hash;

		if (typeof size === "number" && typeof count === "number") {
			State.stats.size = data.size || 0;
			State.stats.count = data.count || 0;

			updateStatsUI();
		}

		switch (type) {
			case 0: // Create
				if (State.controllers.query || State.query) {
					return;
				}

				if (echo) {
					if (id) {
						const placeholder = document.getElementById(`uploading-${id}`);

						if (placeholder) {
							replaceUploadingNode(id, echo);

							return;
						}
					}

					const exists = document.getElementById(`echo-${echo.hash}`);

					if (!exists) {
						renderBatch(echo, "prepend");
					}
				}

				break;

			case 1: // Update
				if (State.cache.has(targetHash) && echo) {
					renderBatch(echo, "update");
				}

				break;

			case 2: // Delete
				removeLocalEcho(targetHash);

				break;
		}
	}

	async function setupSSE() {
		State.controllers.sse?.abort();

		const controller = new AbortController();

		State.controllers.sse = controller;

		try {
			const response = await fetchWithAuth("/echo", null, {
				signal: controller.signal,
			});

			if (!response.ok) {
				throw new Error(response.statusText);
			}

			const reader = response.body.pipeThrough(new TextDecoderStream()).getReader();

			let buffer = "";

			while (true) {
				const { value, done } = await reader.read();

				if (done) {
					break;
				}

				buffer += value;

				const lines = buffer.split("\n");

				buffer = lines.pop();

				for (const line of lines) {
					if (!line || line === "ping") {
						continue;
					}

					try {
						handleServerEvent(JSON.parse(line));
					} catch (err) {
						console.warn(`SSE Parse error: ${err}`);
					}
				}
			}
		} catch (err) {
			if (!controller.signal.aborted) {
				State.controllers.sse = null;
			}

			if (!State.alive) {
				return;
			}

			console.warn(`SSE error: ${err}, retrying...`);

			setTimeout(setupSSE, 3000);
		}
	}

	async function fetchWithAuth(url, tokenOverride, options = {}) {
		const token = tokenOverride || State.token,
			headers = options.headers || {};

		headers["Authorization"] = `Bearer ${token}`;

		options.headers = headers;

		return fetch(url, options);
	}

	async function parseResponseError(res) {
		try {
			const data = await res.json();

			return data.error || res.statusText;
		} catch {
			return res.statusText || "unknown network error";
		}
	}

	function formatBytes(bytes) {
		if (bytes === 0) {
			return "0 B";
		}
		const k = 1000,
			sizes = ["B", "kB", "MB", "GB", "TB"];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
	}

	function formatBytesFull(bytes) {
		if (bytes === 0) {
			return "0 B";
		}
		const k = 1000,
			sizes = ["B", "kB", "MB", "GB", "TB"];

		const parts = [];

		for (let x = sizes.length - 1; x >= 0; x--) {
			const multiplier = Math.pow(k, x);

			if (bytes < multiplier) {
				continue;
			}

			const amount = Math.floor(bytes / multiplier);

			bytes -= amount * multiplier;

			parts.push(`${amount} ${sizes[x]}`);
		}

		return parts.join(" ");
	}

	function formatDate(timestamp) {
		if (!timestamp) {
			return "";
		}

		return new Date(timestamp * 1000).toISOString().split("T")[0];
	}

	function formatDuration(sec) {
		if (!Number.isFinite(sec) || sec <= 0) {
			return "0s";
		}

		sec = Math.floor(sec);

		const m = Math.floor(sec / 60),
			s = sec % 60;

		return m > 0 ? `${m}m${s}s` : `${s}s`;
	}

	function getResolutionTag(w, h) {
		const long = Math.max(w, h),
			short = Math.min(w, h);

		if (short === 1440) {
			if (long >= 5120) {
				return "DQHD";
			}

			if (long >= 3440) {
				return "UWQHD";
			}
		}

		if (short === 1080 && long >= 2560) {
			return "UW-HD";
		}

		if (short === 1600 && long >= 3840) {
			return "UW-4K";
		}

		const match = Resolutions.find(([val]) => short >= val);

		return match ? match[1] : "";
	}

	async function copyLink(hash, btn) {
		const item = State.cache.get(hash);

		if (!item) {
			return;
		}

		try {
			await navigator.clipboard.writeText(item.url);

			if (btn) {
				const txt = btn.textContent;

				btn.textContent = "COPIED";

				setTimeout(() => {
					btn.textContent = txt;
				}, 1000);
			}
		} catch {
			showNotification("Failed to copy", "error");
		}
	}

	function setupEvents() {
		// Auth
		$loginForm.addEventListener("submit", event => {
			event.preventDefault();

			verifyToken($apiToken.value);
		});

		$logoutBtn.addEventListener("click", () => logout(true));

		// Search
		let debounceTimer;

		const executeSearch = val => {
			const query = val.trim();

			if (query === State.query) {
				return;
			}

			$gallery.innerHTML = "";

			State.query = query;
			State.page = 1;
			State.hasMore = true;
			State.cache.clear();

			loadEchos();
		};

		$searchInput.addEventListener("keydown", event => {
			if (event.key === "Enter") {
				clearTimeout(debounceTimer);

				executeSearch(event.target.value);

				event.target.blur();
			} else if (event.key === "Escape") {
				clearTimeout(debounceTimer);

				$searchInput.value = "";

				executeSearch("");

				event.target.blur();
			}
		});

		$searchInput.addEventListener("input", () => {
			clearTimeout(debounceTimer);

			if (State.busy > 0) {
				return;
			}

			debounceTimer = setTimeout(() => executeSearch($searchInput.value), 500);
		});

		// Gallery Delegation
		$gallery.addEventListener("click", event => {
			const target = event.target,
				btn = target.closest("button"),
				link = target.closest(".echo-link");

			if (btn) {
				event.preventDefault();
				event.stopPropagation();

				const { action, hash } = btn.dataset;

				if (action === "tag") {
					event.shiftKey ? updateTag(hash, "auto") : viewTagModal(hash);
				} else if (action === "copy") {
					copyLink(hash, btn);
				} else if (action === "delete") {
					deleteEcho(hash, event.shiftKey);
				}

				return;
			}

			if (link) {
				if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
					return;
				}

				event.preventDefault();

				const card = link.closest(".echo-card");

				if (card?.classList.contains("uploading")) {
					return;
				}

				openModal(card.dataset.hash);
			}
		});

		// Scroll (Infinite Load)
		window.addEventListener("scroll", () => {
			if (!State.token || !$dashboardView.classList.contains("hidden") === false) {
				return;
			}

			const { scrollTop, scrollHeight, clientHeight } = document.documentElement;

			if (scrollTop + clientHeight >= scrollHeight - 300) {
				loadEchos();
			}
		});

		// Uploads
		$uploadBtn.addEventListener("click", () => $fileInput.click());

		$fileInput.addEventListener("change", event => {
			if (!event.target.files.length) {
				return;
			}

			Array.from(event.target.files).forEach(handleUpload);

			$fileInput.value = "";
		});

		document.addEventListener("paste", event => {
			if (event.target.tagName === "INPUT") {
				return;
			}

			const items = event.clipboardData?.items;

			if (!items) {
				return;
			}

			let hasFile = false;

			for (const item of items) {
				if (item.kind === "file") {
					const file = item.getAsFile();

					if (file) {
						hasFile = true;

						handleUpload(file);
					}
				}
			}

			if (hasFile) {
				event.preventDefault();
			}
		});

		// Drag & Drop
		let dragCounter = 0;

		window.addEventListener("dragenter", event => {
			event.preventDefault();

			dragCounter++;

			if (dragCounter === 1) {
				$dropOverlay.classList.remove("hidden");
			}
		});

		window.addEventListener("dragleave", event => {
			event.preventDefault();

			dragCounter--;

			if (dragCounter <= 0) {
				dragCounter = 0;

				$dropOverlay.classList.add("hidden");
			}
		});

		window.addEventListener("dragover", event => event.preventDefault());

		window.addEventListener("drop", event => {
			event.preventDefault();

			dragCounter = 0;

			$dropOverlay.classList.add("hidden");

			if (event.dataTransfer.files.length) {
				Array.from(event.dataTransfer.files).forEach(handleUpload);
			}
		});

		// Modals
		$modalViewBackdrop.addEventListener("click", closeModals);

		$modalTagBackdrop.addEventListener("click", closeModals);

		$tagCloseBtn.addEventListener("click", closeModals);

		document.addEventListener("keydown", event => {
			if (event.key !== "Escape") {
				return;
			}

			closeModals();
		});

		// Tag update
		$tagUpdateBtn.addEventListener("click", () => {
			if ($modalTag.classList.contains("hidden")) {
				return;
			}

			updateTag($modalTag.dataset.hash, $tagSelect.value);
		});

		// Window unload
		window.addEventListener("beforeunload", () => {
			State.alive = false;

			State.controllers.sse?.abort();
		});
	}

	init();
})();

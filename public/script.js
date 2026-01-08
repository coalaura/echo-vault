(() => {
	const StorageKey = "echo_vault_token",
		VolumeKey = "echo_vault_volume",
		VideoExtensions = ["mp4", "webm", "mov", "m4v", "mkv"];

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

	let authToken = localStorage.getItem(StorageKey),
		globalVolume = parseFloat(localStorage.getItem(VolumeKey)),
		currentPage = 1,
		isLoading = false,
		hasMore = true,
		dragCounter = 0,
		echoCache = new Map(),
		totalSize = 0,
		totalCount = 0,
		currentQuery = "",
		noBlur = false,
		ignoreSafety = [];

	if (typeof globalVolume !== "number" || !Number.isFinite(globalVolume) || globalVolume < 0 || globalVolume > 1) {
		globalVolume = 0.75;
	}

	$searchInput.value = "";

	let $notifyArea;

	async function init() {
		$notifyArea = document.createElement("div");

		$notifyArea.id = "notification-area";

		document.body.appendChild($notifyArea);

		await fetchInfo();

		if (authToken) {
			verifyToken(authToken);
		} else {
			switchView("login");
		}

		setupEventListeners();
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
			const mult = Math.pow(k, x);

			if (bytes < mult) {
				continue;
			}

			const amount = Math.floor(bytes / mult);

			bytes -= amount * mult;

			parts.push(`${amount} ${sizes[x]}`);
		}

		return parts.join(" ");
	}

	function formatDate(timestamp) {
		if (!timestamp) {
			return "";
		}

		const date = new Date(timestamp * 1000);

		return date.toISOString().split("T")[0];
	}

	function formatDuration(seconds) {
		if (!Number.isFinite(seconds) || seconds <= 0) {
			return "0s";
		}

		seconds = Math.floor(seconds);

		const m = Math.floor(seconds / 60),
			s = seconds % 60;

		if (m > 0) {
			return `${m}m${s}s`;
		}

		return `${s}s`;
	}

	function showNotification(message, type = "info") {
		const toast = document.createElement("div");

		toast.className = `notification ${type}`;
		toast.textContent = message;

		$notifyArea.appendChild(toast);

		setTimeout(() => {
			toast.classList.add("fade-out");

			toast.addEventListener("animationend", () => {
				toast.remove();
			});
		}, 3000);
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

	function updateTotalSize() {
		$totalSize.textContent = formatBytes(totalSize);
		$totalSize.title = formatBytesFull(totalSize);

		$totalCount.textContent = totalCount.toLocaleString();
	}

	async function fetchInfo() {
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
				searchEnabled = true;

				$searchWrapper.classList.remove("hidden");
			}

			if (!data.blur) {
				noBlur = true;
			}

			if (data.ignore?.length) {
				ignoreSafety = data.ignore;
			}
		} catch (err) {
			console.error(`Failed to fetch version: ${err}`);
		}
	}

	async function verifyToken(token) {
		try {
			const response = await fetchWithAuth("/verify", token);

			if (response.status !== 200) {
				throw new Error("Unauthorized");
			}

			authToken = token;

			localStorage.setItem(StorageKey, token);

			switchView("dashboard");

			loadEchos();
		} catch {
			logout(false);

			$loginError.classList.remove("hidden");
		}
	}

	function logout(clearUi) {
		authToken = null;
		currentPage = 1;
		hasMore = true;
		currentQuery = "";
		$searchInput.value = "";
		echoCache.clear();

		localStorage.removeItem(StorageKey);

		if (clearUi) {
			$gallery.innerHTML = "";

			$loginError.classList.add("hidden");
		}

		switchView("login");
	}

	async function fetchWithAuth(url, tokenOverride, options) {
		const token = tokenOverride || authToken,
			opts = options || {},
			headers = opts.headers || {};

		headers["Authorization"] = `Bearer ${token}`;

		opts.headers = headers;

		return fetch(url, opts);
	}

	async function parseResponseError(response) {
		try {
			const data = await response.json();

			return data.error || response.statusText;
		} catch {
			return response.statusText || "unknown network error";
		}
	}

	async function loadEchos() {
		if (isLoading || !hasMore) {
			return;
		}

		isLoading = true;

		$searchInput.setAttribute("disabled", true);

		$loader.classList.remove("hidden");

		try {
			let endpoint;

			if (currentQuery) {
				endpoint = `/query/${currentPage}?q=${encodeURIComponent(currentQuery)}`;
			} else {
				endpoint = `/echos/${currentPage}`;
			}

			const response = await fetchWithAuth(endpoint);

			if (!response.ok) {
				const msg = await parseResponseError(response);

				throw new Error(msg);
			}

			const data = await response.json();

			if (!data || !data?.echos?.length) {
				hasMore = false;

				if (currentPage === 1) {
					$emptyState.classList.remove("hidden");
				}
			} else {
				$emptyState.classList.add("hidden");

				totalSize = data.size || 0;
				totalCount = data.count || 0;

				updateTotalSize();

				renderItems(data.echos, false);

				currentPage++;
			}
		} catch (error) {
			showNotification(error.message, "error");
		} finally {
			isLoading = false;

			$searchInput.removeAttribute("disabled");

			$loader.classList.add("hidden");
		}
	}

	function renderItems(items, prepend) {
		const fragment = document.createDocumentFragment(),
			list = Array.isArray(items) ? items : [items];

		list.forEach(item => {
			const id = `echo-${item.hash}`,
				ext = item.extension,
				url = item.url,
				isVideo = VideoExtensions.includes(ext);

			// Card container
			const card = document.createElement("div");

			card.id = id;
			card.className = "echo-card";
			card.dataset.hash = item.hash;

			if (!noBlur && item.tag?.safety && item.tag?.safety !== "ok" && !ignoreSafety.includes(item.tag?.safety)) {
				card.classList.add("blurred", `safety-${item.tag?.safety}`);
			}

			// Link container
			const link = document.createElement("a");

			link.href = url;
			link.target = "_blank";
			link.className = "echo-link";

			// Loader
			const loader = document.createElement("div");

			loader.className = "media-loader";

			const spinner = document.createElement("span");

			spinner.className = "spinner";

			loader.appendChild(spinner);

			link.appendChild(loader);

			const onLoad = () => {
				if (loader.parentNode) {
					loader.remove();
				}

				media.classList.add("loaded");
			};

			const onError = () => {
				if (loader.parentNode) {
					loader.remove();
				}

				const errorState = document.createElement("div");

				errorState.className = "media-error";

				errorState.textContent = "FAILED";

				link.appendChild(errorState);
			};

			// Media element
			let media;

			if (isVideo) {
				media = document.createElement("video");

				media.className = "echo-media";
				media.muted = true;
				media.loop = true;

				// Video Badge
				const badge = document.createElement("div");

				badge.className = "type-badge";
				badge.textContent = "▶";

				card.appendChild(badge);

				media.addEventListener("loadeddata", () => {
					badge.textContent = formatDuration(media.duration);

					onLoad();
				});

				media.addEventListener("error", onError);

				card.addEventListener("mouseenter", () => {
					media.play();
				});

				card.addEventListener("mouseleave", () => {
					media.pause();
					media.currentTime = 0;
				});

				media.src = url;
			} else {
				media = document.createElement("img");

				media.className = "echo-media";
				media.loading = "lazy";

				media.addEventListener("load", onLoad);
				media.addEventListener("error", onError);

				media.src = url;
			}

			link.appendChild(media);

			card.appendChild(link);

			// Similarity
			if (item.tag?.similarity) {
				const similarity = document.createElement("div"),
					percentage = Math.round(item.tag.similarity * 100);

				similarity.className = "echo-similarity";

				similarity.textContent = `${percentage}% MATCH`;

				card.appendChild(similarity);
			}

			// Actions container
			const actions = document.createElement("div");

			actions.className = "echo-actions";

			const tagBtn = document.createElement("button");

			tagBtn.className = "action-btn";

			tagBtn.dataset.action = "tag";
			tagBtn.dataset.hash = item.hash;

			tagBtn.textContent = "TAG";

			const copyBtn = document.createElement("button");

			copyBtn.className = "action-btn";

			copyBtn.dataset.action = "copy";
			copyBtn.dataset.hash = item.hash;

			copyBtn.textContent = "COPY";

			const deleteBtn = document.createElement("button");

			deleteBtn.className = "action-btn delete";
			deleteBtn.dataset.action = "delete";
			deleteBtn.dataset.hash = item.hash;

			deleteBtn.textContent = "DEL";

			actions.append(tagBtn, copyBtn, deleteBtn);

			// Info container
			const info = document.createElement("div");

			info.className = "echo-info";

			// Date
			const dateSpan = document.createElement("span");

			dateSpan.textContent = formatDate(item.timestamp);

			// Size
			const sizeSpan = document.createElement("span");

			sizeSpan.textContent = `${formatBytes(item.upload_size)} 🡒 ${formatBytes(item.size)}`;

			info.append(dateSpan, sizeSpan);

			// Assemble card
			card.append(actions, info);

			const exists = document.getElementById(id);

			if (exists) {
				exists.replaceWith(card);
			} else {
				fragment.appendChild(card);
			}

			item.el = card;

			echoCache.set(item.hash, item);
		});

		if (prepend) {
			$gallery.prepend(fragment);
		} else {
			$gallery.appendChild(fragment);
		}
	}

	function openModal(hash) {
		const item = echoCache.get(hash);

		if (!item) {
			return;
		}

		const ext = item.extension,
			url = item.url,
			isVideo = VideoExtensions.includes(ext);

		$modalViewContent.innerHTML = "";

		if (isVideo) {
			const vid = document.createElement("video");

			vid.src = url;
			vid.volume = globalVolume;
			vid.controls = true;
			vid.autoplay = true;

			vid.addEventListener("volumechange", () => {
				localStorage.setItem(VolumeKey, vid.volume);
			});

			$modalViewContent.appendChild(vid);
		} else {
			const img = document.createElement("img");

			img.src = url;

			$modalViewContent.appendChild(img);
		}

		$modalView.classList.remove("hidden");
	}

	function closeModals() {
		$modalTag.classList.add("hidden");

		$modalView.classList.add("hidden");
		$modalViewContent.innerHTML = "";
	}

	function handleSearch(query) {
		const normalized = query.trim();

		if (normalized === currentQuery) {
			return;
		}

		currentQuery = normalized;

		currentPage = 1;
		hasMore = true;
		$gallery.innerHTML = "";
		echoCache.clear();

		loadEchos();
	}

	async function copyLink(hash, btnElement) {
		const item = echoCache.get(hash);

		if (!item) {
			return;
		}

		const url = item.url;

		try {
			await navigator.clipboard.writeText(url);

			if (btnElement) {
				const originalText = btnElement.textContent;

				btnElement.textContent = "COPIED";

				setTimeout(() => {
					btnElement.textContent = originalText;
				}, 1000);
			}
		} catch {
			showNotification("Failed to copy", "error");
		}
	}

	async function handleUpload(file) {
		const formData = new FormData();

		formData.append("upload", file);

		$uploadBtn.textContent = "UPLOADING...";
		$uploadBtn.disabled = true;

		try {
			const response = await fetchWithAuth("/upload?return", null, {
				method: "POST",
				body: formData,
			});

			if (!response.ok) {
				const msg = await parseResponseError(response);

				throw new Error(msg);
			}

			const newEcho = (await response.json())?.echo;

			if (!newEcho) {
				throw new Error("invalid response");
			}

			totalSize += newEcho.size;
			totalCount++;

			updateTotalSize();

			$emptyState.classList.add("hidden");

			renderItems(newEcho, true);

			showNotification("Upload complete", "success");
		} catch (err) {
			showNotification(err.message, "error");
		} finally {
			$uploadBtn.textContent = "UPLOAD_FILE";
			$uploadBtn.disabled = false;

			$fileInput.value = "";
		}
	}

	async function viewTagModal(hash) {
		const item = echoCache.get(hash);

		if (!item || item.el.classList.contains("processing")) {
			return;
		}

		$modalTag.classList.remove("hidden");
		$modalTag.dataset.hash = hash;

		if (item.tag?.sensitive) {
			$tagSelect.value = item.tag.sensitive;
		} else {
			$tagSelect.value = "auto";
		}
	}

	async function updateTag(hash, tag) {
		const item = hash ? echoCache.get(hash) : null;

		if (!item || item.el.classList.contains("processing")) {
			return;
		}

		closeModals();

		item.el.classList.add("processing");

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
				const msg = await parseResponseError(response);

				throw new Error(msg);
			}

			const data = await response.json();

			if (!data) {
				throw new Error("invalid response");
			}

			totalSize = data.size || 0;
			totalCount = data.count || 0;

			updateTotalSize();

			renderItems(data.echo, false);

			const newTag = data.echo.tag?.safety || "ok";

			showNotification(`Re-Tag complete: ${newTag} ${newTag !== "ok" && (noBlur || ignoreSafety.includes(newTag)) ? "(ignored)" : ""}`, "success");
		} catch (err) {
			showNotification(err.message, "error");
		} finally {
			item.el.classList.remove("processing");
		}
	}

	async function deleteEcho(hash, noConfirm = false) {
		const item = echoCache.get(hash);

		if (!item || item.el.classList.contains("processing")) {
			return;
		}

		if (!noConfirm && !confirm("Delete this echo?")) {
			return;
		}

		item.el.classList.add("processing");

		try {
			const response = await fetchWithAuth(`/echos/${hash}`, null, {
				method: "DELETE",
			});

			if (!response.ok) {
				const msg = await parseResponseError(response);

				throw new Error(msg);
			}

			item.el.remove();

			totalSize -= item.size;
			totalCount--;

			updateTotalSize();

			echoCache.delete(hash);

			if ($gallery.children.length === 0) {
				$emptyState.classList.remove("hidden");
			}

			showNotification("Echo deleted", "success");
		} catch (error) {
			showNotification(error.message, "error");
		} finally {
			item.el.classList.remove("processing");
		}
	}

	function setupEventListeners() {
		$loginForm.addEventListener("submit", event => {
			event.preventDefault();

			verifyToken($apiToken.value);
		});

		$logoutBtn.addEventListener("click", () => {
			logout(true);
		});

		$searchInput.addEventListener("keydown", event => {
			if (event.key === "Enter") {
				handleSearch(event.target.value);

				$searchInput.blur();
			} else if (event.key === "Escape") {
				$searchInput.value = "";

				handleSearch("");

				$searchInput.blur();
			}
		});

		$gallery.addEventListener("click", async event => {
			const target = event.target,
				btn = target.closest("button"),
				link = target.closest(".echo-link");

			if (btn) {
				event.preventDefault();
				event.stopPropagation();

				const action = btn.dataset.action,
					hash = btn.dataset.hash;

				if (action === "tag") {
					if (event.shiftKey) {
						updateTag(hash, "auto");
					} else {
						viewTagModal(hash);
					}
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

				openModal(card.dataset.hash);
			}
		});

		$modalViewBackdrop.addEventListener("click", closeModals);
		$modalTagBackdrop.addEventListener("click", closeModals);

		document.addEventListener("keydown", event => {
			if (event.key === "Escape") {
				closeModals();
			}
		});

		window.addEventListener("scroll", () => {
			if (authToken && !$dashboardView.classList.contains("hidden")) {
				const doc = document.documentElement;

				const scrollTop = doc.scrollTop,
					scrollHeight = doc.scrollHeight,
					clientHeight = doc.clientHeight;

				if (scrollTop + clientHeight >= scrollHeight - 300) {
					loadEchos();
				}
			}
		});

		$uploadBtn.addEventListener("click", () => {
			$fileInput.click();
		});

		$fileInput.addEventListener("change", event => {
			const files = event.target.files;

			if (files.length > 0) {
				handleUpload(files[0]);
			}
		});

		$tagCloseBtn.addEventListener("click", () => {
			closeModals();
		});

		$tagUpdateBtn.addEventListener("click", async () => {
			if ($modalTag.classList.contains("hidden")) {
				return;
			}

			updateTag($modalTag.dataset.hash, $tagSelect.value);
		});

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

		window.addEventListener("dragover", event => {
			event.preventDefault();
		});

		window.addEventListener("drop", event => {
			event.preventDefault();

			dragCounter = 0;

			$dropOverlay.classList.add("hidden");

			const files = event.dataTransfer.files;

			if (files.length > 0) {
				handleUpload(files[0]);
			}
		});
	}

	init();
})();

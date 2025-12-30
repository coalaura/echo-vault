(() => {
	const StorageKey = "echo_vault_token";

	const $loginView = document.getElementById("login-view"),
		$dashboardView = document.getElementById("dashboard-view"),
		$modalView = document.getElementById("modal-view"),
		$loginForm = document.getElementById("login-form"),
		$apiToken = document.getElementById("api-token"),
		$loginError = document.getElementById("login-error"),
		$gallery = document.getElementById("gallery"),
		$emptyState = document.getElementById("empty-state"),
		$loader = document.getElementById("loader"),
		$dropOverlay = document.getElementById("drop-overlay"),
		$fileInput = document.getElementById("file-input"),
		$uploadBtn = document.getElementById("upload-trigger"),
		$logoutBtn = document.getElementById("logout-btn"),
		$versionTags = document.querySelectorAll(".version-tag"),
		$modalContent = document.querySelector(".modal-content"),
		$modalBackdrop = document.querySelector(".modal-backdrop");

	let authToken = localStorage.getItem(StorageKey),
		currentPage = 1,
		isLoading = false,
		hasMore = true,
		dragCounter = 0,
		echoCache = new Map();

	let $notifyArea;

	function init() {
		$notifyArea = document.createElement("div");

		$notifyArea.id = "notification-area";

		document.body.appendChild($notifyArea);

		fetchVersion();

		if (authToken) {
			verifyToken(authToken);
		} else {
			switchView("login");
		}

		setupEventListeners();
	}

	function showNotification(message, type = "info") {
		const toast = document.createElement("div");

		toast.className = `notification ${type}`;
		toast.innerText = message;

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

	async function fetchVersion() {
		try {
			const response = await fetch("/info"),
				data = await response.json();

			if (!data?.version) {
				throw new Error("invalid response");
			}

			$versionTags.forEach(tag => {
				tag.innerText = data.version;
			});
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

		$loader.classList.remove("hidden");

		try {
			const response = await fetchWithAuth(`/echos/${currentPage}`);

			if (!response.ok) {
				const msg = await parseResponseError(response);

				throw new Error(msg);
			}

			const data = await response.json();

			if (!data || data.length === 0) {
				hasMore = false;

				if (currentPage === 1) {
					$emptyState.classList.remove("hidden");
				}
			} else {
				$emptyState.classList.add("hidden");

				renderItems(data, false);

				currentPage++;
			}
		} catch (error) {
			showNotification(error.message, "error");
		} finally {
			isLoading = false;

			$loader.classList.add("hidden");
		}
	}

	function renderItems(items, prepend) {
		const fragment = document.createDocumentFragment(),
			list = Array.isArray(items) ? items : [items];

		list.forEach(item => {
			echoCache.set(item.hash, item);

			const ext = item.ext || item.extension,
				url = `${window.location.origin}/i/${item.hash}.${ext}`,
				isVideo = ext === "mp4" || ext === "webm";

			const card = document.createElement("div");

			card.className = "echo-card";
			card.dataset.hash = item.hash;

			let media;

			if (isVideo) {
				media = `<video src="${url}" class="echo-media" muted loop onmouseover="this.play()" onmouseout="this.pause()"></video>`;
			} else {
				media = `<img src="${url}" class="echo-media" loading="lazy">`;
			}

			card.innerHTML = `
                ${media}
                <div class="echo-actions">
                    <button class="action-btn" data-action="copy" data-hash="${item.hash}">COPY</button>
                    <button class="action-btn delete" data-action="delete" data-hash="${item.hash}">DEL</button>
                </div>
            `;

			fragment.appendChild(card);
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

		const ext = item.ext || item.extension,
			url = `${window.location.origin}/i/${item.hash}.${ext}`,
			isVideo = ext === "mp4" || ext === "webm";

		$modalContent.innerHTML = "";

		if (isVideo) {
			const vid = document.createElement("video");

			vid.src = url;
			vid.controls = true;
			vid.autoplay = true;

			$modalContent.appendChild(vid);
		} else {
			const img = document.createElement("img");

			img.src = url;

			$modalContent.appendChild(img);
		}

		$modalView.classList.remove("hidden");
	}

	function closeModal() {
		$modalView.classList.add("hidden");
		$modalContent.innerHTML = "";
	}

	async function copyLink(hash, btnElement) {
		const item = echoCache.get(hash);

		if (!item) {
			return;
		}

		const ext = item.ext || item.extension,
			url = `${window.location.origin}/i/${item.hash}.${ext}`;

		try {
			await navigator.clipboard.writeText(url);

			if (btnElement) {
				const originalText = btnElement.innerText;

				btnElement.innerText = "COPIED";

				setTimeout(() => {
					btnElement.innerText = originalText;
				}, 1000);
			}
		} catch {
			showNotification("Failed to copy", "error");
		}
	}

	async function handleUpload(file) {
		const formData = new FormData();

		formData.append("upload", file);

		$uploadBtn.innerText = "UPLOADING...";
		$uploadBtn.disabled = true;

		try {
			const response = await fetchWithAuth("/upload", null, {
				method: "POST",
				body: formData,
			});

			if (!response.ok) {
				const msg = await parseResponseError(response);

				throw new Error(msg);
			}

			const newEcho = await response.json();

			$emptyState.classList.add("hidden");

			renderItems(newEcho, true);

			showNotification("Upload complete", "success");
		} catch (err) {
			showNotification(err.message, "error");
		} finally {
			$uploadBtn.innerText = "UPLOAD_FILE";
			$uploadBtn.disabled = false;

			$fileInput.value = "";
		}
	}

	async function deleteEcho(hash) {
		if (!confirm("Delete this echo?")) {
			return;
		}

		try {
			const response = await fetchWithAuth(`/echos/${hash}`, null, {
				method: "DELETE",
			});

			if (!response.ok) {
				const msg = await parseResponseError(response);

				throw new Error(msg);
			}

			const card = document.querySelector(`.echo-card[data-hash="${hash}"]`);

			if (card) {
				card.remove();
			}

			echoCache.delete(hash);

			if ($gallery.children.length === 0) {
				$emptyState.classList.remove("hidden");
			}

			showNotification("Echo deleted", "success");
		} catch (error) {
			showNotification(error.message, "error");
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

		$gallery.addEventListener("click", async event => {
			const target = event.target,
				btn = target.closest("button"),
				card = target.closest(".echo-card");

			if (btn) {
				const action = btn.dataset.action,
					hash = btn.dataset.hash;

				if (action === "copy") {
					copyLink(hash, btn);
				} else if (action === "delete") {
					deleteEcho(hash);
				}

				return;
			}

			if (card) {
				openModal(card.dataset.hash);
			}
		});

		$modalBackdrop.addEventListener("click", closeModal);

		document.addEventListener("keydown", event => {
			if (event.key === "Escape" && !$modalView.classList.contains("hidden")) {
				closeModal();
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

/* Nexo Chess: all game rules and legal moves come from the local server. */
(() => {
  "use strict";

  const $ = (id) => document.getElementById(id);
  const files = "abcdefgh";
  const pieceNames = { p: "peón", n: "caballo", b: "alfil", r: "torre", q: "dama", k: "rey" };
  const promotionNames = { q: "Dama", r: "Torre", b: "Alfil", n: "Caballo" };
  const sessionKey = "nexo-session";
  let token = "";
  try { token = sessionStorage.getItem(sessionKey) || ""; } catch (_) { /* Private browsing can deny storage. */ }

  const state = {
    room: null, snapshot: null, rooms: [], info: null,
    selected: null, focusSquare: "e2", review: null, flipped: false,
    online: false, busy: false, ready: false, promotion: null,
    stopped: false, lobbySignature: "", historySignature: ""
  };
  const themeQuery = window.matchMedia("(prefers-color-scheme: dark)");
  let chosenTheme = null;
  let requestQueue = Promise.resolve();
  let pollTimer = null;
  let pollPromise = null;
  let confirmAction = null;
  let routing = false;
  const squares = [];

  function setTheme() {
    const dark = chosenTheme ? chosenTheme === "dark" : themeQuery.matches;
    document.documentElement.dataset.theme = dark ? "dark" : "light";
    $("theme-toggle").setAttribute("aria-label", `Cambiar al tema ${dark ? "claro" : "oscuro"}`);
    $("theme-toggle").title = `Cambiar al tema ${dark ? "claro" : "oscuro"}`;
  }
  $("theme-toggle").addEventListener("click", () => {
    chosenTheme = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
    setTheme();
  });
  if (themeQuery.addEventListener) themeQuery.addEventListener("change", () => { if (!chosenTheme) setTheme(); });
  setTheme();

  function notify(message, kind = "info") {
    $("notice-text").textContent = message;
    $("notice").dataset.kind = kind;
    $("notice").dataset.network = String(kind === "error" && message.startsWith("No se puede conectar con Nexo."));
    $("notice").hidden = false;
  }
  $("notice-close").addEventListener("click", () => { $("notice").hidden = true; });
  const announce = (message) => { $("screen-reader-status").textContent = message; };

  function rememberToken(value) {
    token = value;
    try { sessionStorage.setItem(sessionKey, value); } catch (_) { /* The active tab still works. */ }
  }

  async function rawRequest(path, method, data, authenticated) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 10000);
    const headers = { "X-Nexo-Client": "1" };
    if (authenticated && token) headers.Authorization = `Bearer ${token}`;
    if (method === "POST") headers["Content-Type"] = "application/json";
    try {
      const response = await fetch(path, {
        method, headers, credentials: "omit", cache: "no-store",
        body: method === "POST" ? JSON.stringify(data || {}) : undefined,
        signal: controller.signal
      });
      state.online = true;
      if ($("notice").dataset.network === "true") $("notice").hidden = true;
      renderConnection();
      let result;
      try { result = await response.json(); }
      catch (_) { throw new Error("Nexo ha recibido una respuesta inesperada. Inténtalo de nuevo."); }
      if (!response.ok) {
        const error = new Error(result.error || "No se pudo completar la petición.");
        error.status = response.status;
        throw error;
      }
      return result;
    } catch (error) {
      if (error.name === "AbortError" || error instanceof TypeError) {
        state.online = false;
        renderConnection();
        renderControls();
        renderBoard();
        const networkError = new Error("No se puede conectar con Nexo. Comprueba que sigue abierto y que estáis en la misma red. Se volverá a intentar automáticamente.");
        networkError.status = 0;
        throw networkError;
      }
      throw error;
    } finally { clearTimeout(timeout); }
  }

  async function newSession() {
    const result = await rawRequest("/api/session", "POST", {}, false);
    if (!result.token) throw new Error("No se ha podido abrir una sesión. Recarga la página.");
    rememberToken(result.token);
  }

  // Every request joins one queue: a poll and a click can never run two fetches at once.
  function api(path, method = "GET", data) {
    const task = requestQueue.then(async () => {
      if (!token) await newSession();
      try { return await rawRequest(path, method, data, true); }
      catch (error) {
        if (error.status !== 401) throw error;
        await newSession();
        return rawRequest(path, method, data, true);
      }
    });
    requestQueue = task.catch(() => {});
    return task;
  }

  function renderConnection() {
    $("connection").className = `connection ${state.online ? "connected" : state.ready ? "disconnected" : ""}`;
    $("connection-label").textContent = state.stopped ? "Nexo cerrado" : state.online ? "En tu red" : state.ready ? "Reconectando" : "Conectando";
  }

  function stopPolling() {
    if (pollTimer !== null) clearTimeout(pollTimer);
    pollTimer = null;
  }

  function schedulePoll() {
    stopPolling();
    if (!state.stopped) pollTimer = setTimeout(poll, 1000);
  }

  async function poll() {
    pollTimer = null;
    if (state.stopped) return;
    if (state.busy || routing || pollPromise) { schedulePoll(); return; }
    const room = state.room;
    pollPromise = (async () => {
      try {
        if (!state.info) { state.info = await api("/api/info"); renderInfo(); }
        if (room) {
          const snapshot = await api(`/api/state?room=${encodeURIComponent(room)}`);
          if (state.room === room) applySnapshot(snapshot);
        } else await refreshLobby();
        state.ready = true;
        renderControls();
      } catch (error) {
        if (state.room === room && room && (error.status === 403 || error.status === 404)) {
          showLobby();
          prepareJoin(room);
          notify(error.message, "error");
        } else if (error.status !== 0) notify(error.message, "error");
        renderControls();
      }
    })();
    try { await pollPromise; }
    finally { pollPromise = null; schedulePoll(); }
  }

  function roomFromHash() {
    return (new URLSearchParams(location.hash.slice(1)).get("room") || "").trim().toUpperCase();
  }

  function setRoomHash(room) {
    history.replaceState(null, "", `${location.pathname}${location.search}${room ? `#room=${encodeURIComponent(room)}` : ""}`);
  }

  function switchTab(tab) {
    const creating = tab === "create";
    $("create-form").hidden = !creating;
    $("join-form").hidden = creating;
    for (const [id, active] of [["tab-create", creating], ["tab-join", !creating]]) {
      $(id).classList.toggle("active", active);
      $(id).setAttribute("aria-selected", String(active));
      $(id).tabIndex = active ? 0 : -1;
    }
  }
  $("tab-create").addEventListener("click", () => switchTab("create"));
  $("tab-join").addEventListener("click", () => switchTab("join"));
  for (const tab of [$("tab-create"), $("tab-join")]) {
    tab.addEventListener("keydown", (event) => {
      if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
      event.preventDefault();
      const target = tab.id === "tab-create" ? "tab-join" : "tab-create";
      switchTab(target === "tab-create" ? "create" : "join");
      $(target).focus();
    });
  }

  function prepareJoin(room, color = "w") {
    switchTab("join");
    $("room-code").value = room;
    const option = document.querySelector(`input[name="join-color"][value="${color}"]`);
    if (option) option.checked = true;
  }

  function showLobby() {
    state.room = null;
    state.snapshot = null;
    state.review = null;
    state.selected = null;
    state.promotion = null;
    if ($("promotion-dialog").open) $("promotion-dialog").close();
    $("lobby-view").hidden = false;
    $("game-view").hidden = true;
    document.documentElement.scrollTop = 0;
    document.body.scrollTop = 0;
    document.title = "Nexo Chess · Ajedrez en tu red";
    renderControls();
  }

  async function routeFromHash() {
    const room = roomFromHash();
    if (state.room && room !== state.room) {
      setRoomHash(state.room);
      notify("Ya estás en una sala. Usa «Salir de la sala» antes de entrar en otra.");
      return;
    }
    if (routing || state.busy) return;
    routing = true;
    stopPolling();
    try {
      if (!room) { showLobby(); await refreshLobby(); }
      else if (room !== state.room) {
        try { applySnapshot(await api(`/api/state?room=${encodeURIComponent(room)}`)); }
        catch (error) {
          showLobby();
          prepareJoin(room);
          await refreshLobby();
          if (error.status === 403) notify("Invitación preparada. Escribe tu nombre, elige tus piezas o entra como espectador.");
          else notify(error.message, "error");
        }
      }
    } finally { routing = false; schedulePoll(); }
  }
  window.addEventListener("hashchange", () => { routeFromHash().catch((error) => notify(error.message, "error")); });

  function validPlayerName() {
    const input = $("player-name");
    const value = input.value.trim();
    if (!value) { notify("Escribe tu nombre para entrar en la partida."); input.focus(); return null; }
    if ([...value].length > 32) { notify("Tu nombre puede tener hasta 32 caracteres.", "error"); input.focus(); return null; }
    return value;
  }

  async function mutation(work) {
    if (state.busy || !state.online || state.stopped) return;
    state.busy = true;
    stopPolling();
    renderControls();
    try { await work(); }
    catch (error) {
      notify(error.message, "error");
      // A rejected action may mean the other player has changed the position.
      // Refresh once; never repeat a mutation after an uncertain response.
      if (state.room && error.status) {
        const room = state.room;
        try { applySnapshot(await api(`/api/state?room=${encodeURIComponent(room)}`)); }
        catch (refreshError) {
          if (refreshError.status === 403 || refreshError.status === 404) {
            showLobby(); prepareJoin(room);
          }
        }
      }
    }
    finally { state.busy = false; renderControls(); renderBoard(); schedulePoll(); }
  }

  $("create-form").addEventListener("submit", (event) => {
    event.preventDefault();
    const playerName = validPlayerName();
    if (!playerName) return;
    const name = $("room-name").value.trim() || "Una partida entre amigos";
    const color = document.querySelector('input[name="create-color"]:checked').value;
    mutation(async () => {
      applySnapshot(await api("/api/rooms", "POST", { name, playerName, color }));
      setRoomHash(state.room);
      $("notice").hidden = true;
      announce("Sala creada. Comparte el enlace para invitar a tu rival.");
    });
  });

  $("join-form").addEventListener("submit", (event) => {
    event.preventDefault();
    const playerName = validPlayerName();
    if (!playerName) return;
    const room = $("room-code").value.trim().toUpperCase();
    if (!room) { notify("Introduce el código de la sala."); $("room-code").focus(); return; }
    const color = document.querySelector('input[name="join-color"]:checked').value;
    mutation(async () => {
      applySnapshot(await api("/api/join", "POST", { room, playerName, color }));
      setRoomHash(state.room);
      $("notice").hidden = true;
      announce(color === "spectator" ? "Has entrado como espectador." : `Juegas con ${color === "w" ? "blancas" : "negras"}.`);
    });
  });

  const playerLabel = (player) => typeof player === "string" ? player : player && player.name || "Libre";

  async function refreshLobby() {
    const result = await api("/api/lobby");
    state.rooms = Array.isArray(result.rooms) ? result.rooms : [];
    // If a create/join committed but its response was lost, recover membership
    // from this authenticated field rather than replaying the mutation.
    if (!state.room && typeof result.activeRoom === "string" && result.activeRoom) {
      try {
        applySnapshot(await api(`/api/state?room=${encodeURIComponent(result.activeRoom)}`));
        setRoomHash(state.room);
        notify("Tu sala se ha recuperado. Puedes continuar la partida.");
      } catch (error) {
        if (error.status !== 403 && error.status !== 404) throw error;
      }
    }
    if (!state.room) renderLobby();
  }

  function renderLobby() {
    $("room-count").textContent = state.rooms.length;
    const signature = JSON.stringify(state.rooms);
    if (signature === state.lobbySignature && $("rooms-list").children.length) return;
    state.lobbySignature = signature;
    const list = $("rooms-list");
    const focusedRoom = document.activeElement && document.activeElement.dataset.room;
    list.replaceChildren();
    if (!state.rooms.length) {
      list.innerHTML = '<div class="empty-rooms"><span class="empty-icon" aria-hidden="true">＋</span><div><strong>El tablero está listo.</strong><p>Crea la primera sala y reúne a tu rival.</p></div></div>';
      return;
    }
    for (const room of state.rooms) {
      const card = document.createElement("article");
      card.className = "room-card";
      card.innerHTML = '<div class="room-card-top"><h3></h3><span class="room-card-code"></span></div><div class="room-players"><span class="side-token white-token" aria-hidden="true"></span><span class="room-player-name white-name"></span><span aria-hidden="true">·</span><span class="side-token black-token" aria-hidden="true"></span><span class="room-player-name black-name"></span></div><div class="room-card-bottom"><span class="room-card-status"></span><button class="button secondary room-enter" type="button">Entrar <span aria-hidden="true">↗</span></button></div>';
      card.querySelector("h3").textContent = room.name || "Partida";
      card.querySelector(".room-card-code").textContent = room.id;
      card.querySelector(".white-name").textContent = playerLabel(room.white);
      card.querySelector(".black-name").textContent = playerLabel(room.black);
      const finished = ["1-0", "0-1", "1/2-1/2", "finished", "Terminada"].includes(room.status);
      card.querySelector(".room-card-status").textContent = finished ? "Partida terminada" : !room.white || !room.black ? "Hay un sitio libre" : `${Math.ceil((room.ply || 0) / 2)} jugadas · En curso`;
      const button = card.querySelector("button");
      button.dataset.room = room.id;
      button.dataset.color = !room.white ? "w" : !room.black ? "b" : "spectator";
      button.setAttribute("aria-label", `Entrar en ${room.name || "la partida"}, sala ${room.id}`);
      list.append(card);
    }
    if (focusedRoom) {
      const focused = [...list.querySelectorAll("button[data-room]")].find((button) => button.dataset.room === focusedRoom);
      if (focused) focused.focus();
    }
    renderControls();
  }
  $("rooms-list").addEventListener("click", (event) => {
    const button = event.target.closest("button[data-room]");
    if (!button || state.busy) return;
    prepareJoin(button.dataset.room, button.dataset.color);
    if (!$("player-name").value.trim()) { $("player-name").focus(); notify("Escribe tu nombre para entrar en esta sala."); }
    else $("join-form").requestSubmit();
  });

  function renderInfo() {
    if (!state.info) return;
    $("app-version").textContent = state.info.version ? `v${state.info.version}` : "";
    $("shutdown").hidden = !state.info.isHost;
    const urls = Array.isArray(state.info.urls) ? state.info.urls : [];
    const container = $("network-urls");
    container.replaceChildren();
    if (!urls.length) {
      const text = document.createElement("span");
      text.className = "quiet-label";
      text.textContent = "No hay otra dirección de red disponible.";
      container.append(text);
    }
    for (const url of urls) {
      const row = document.createElement("div");
      row.className = "network-address";
      const input = document.createElement("input");
      input.type = "text";
      input.readOnly = true;
      input.value = url;
      input.setAttribute("aria-label", "Dirección de Nexo en tu red");
      input.addEventListener("click", () => input.select());
      const button = document.createElement("button");
      button.type = "button";
      button.className = "icon-button";
      button.setAttribute("aria-label", `Copiar ${url}`);
      button.title = "Copiar dirección";
      button.innerHTML = '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="8" y="8" width="12" height="13" rx="2"/><path d="M16 8V3H3v13h5"/></svg>';
      button.addEventListener("click", () => copyValue(input));
      row.append(input, button);
      container.append(row);
    }
    updateInvitation();
  }

  async function copyValue(input) {
    try {
      if (!navigator.clipboard || !window.isSecureContext) throw new Error("selection");
      await navigator.clipboard.writeText(input.value);
      notify("Enlace copiado. Compártelo con alguien de tu red.");
    } catch (_) {
      input.focus();
      input.select();
      input.setSelectionRange(0, input.value.length);
      notify("Enlace seleccionado. Usa Ctrl+C o mantén pulsado para copiarlo.");
    }
  }

  function updateInvitation() {
    if (!state.room) return;
    const urls = state.info && Array.isArray(state.info.urls) ? state.info.urls : [];
    let base = location.origin;
    if (state.info && state.info.isHost) {
      const lan = urls.find((value) => !/^https?:\/\/(localhost|127\.|\[::1\])/.test(value));
      if (lan) base = lan.replace(/\/$/, "");
    }
    $("share-link").value = `${base}/#room=${encodeURIComponent(state.room)}`;
  }
  $("copy-invite").addEventListener("click", () => copyValue($("share-link")));
  $("share-link").addEventListener("click", () => $("share-link").select());

  function boardFromFen(fen) {
    const board = {};
    const ranks = (fen || "").split(" ")[0].split("/");
    ranks.forEach((row, index) => {
      let file = 0;
      for (const char of row) {
        if (/^[1-8]$/.test(char)) file += Number(char);
        else { if (files[file]) board[`${files[file]}${8 - index}`] = char; file++; }
      }
    });
    return board;
  }
  const colorOfPiece = (piece) => !piece ? null : piece === piece.toUpperCase() ? "w" : "b";

  function canAct() {
    const s = state.snapshot;
    return Boolean(s && s.you !== "spectator" && s.white && s.black && state.online && !state.busy && state.review === null && !state.stopped);
  }
  function canMove() {
    const s = state.snapshot;
    return Boolean(canAct() && s.outcome === "*" && s.you === s.turn);
  }

  function applySnapshot(snapshot) {
    if (!snapshot || !snapshot.id || !snapshot.fen) throw new Error("La sala ha devuelto un estado inesperado.");
    snapshot.history = Array.isArray(snapshot.history) ? snapshot.history : [];
    snapshot.legal = Array.isArray(snapshot.legal) ? snapshot.legal : [];
    const sameRoom = state.room === snapshot.id;
    if (sameRoom && state.snapshot && snapshot.version < state.snapshot.version) return;
    const previous = state.snapshot;
    if (!sameRoom) {
      state.selected = null;
      state.review = null;
      state.flipped = snapshot.you === "b";
      state.focusSquare = snapshot.you === "b" ? "e7" : "e2";
      state.historySignature = "";
      document.documentElement.scrollTop = 0;
      document.body.scrollTop = 0;
    }
    state.room = snapshot.id;
    state.snapshot = snapshot;
    if (state.review !== null) state.review = Math.min(state.review, snapshot.history.length);
    if (state.selected && (!canMove() || colorOfPiece(boardFromFen(snapshot.fen)[state.selected]) !== snapshot.you)) state.selected = null;
    if (state.promotion && !snapshot.legal.some((move) => move.from === state.promotion.moves[0].from && move.to === state.promotion.moves[0].to)) {
      state.promotion = null;
      $("promotion-dialog").close();
    }
    $("lobby-view").hidden = true;
    $("game-view").hidden = false;
    document.title = `${snapshot.name || "Partida"} · Nexo Chess`;
    renderRoom();
    if (sameRoom && previous && previous.history.length !== snapshot.history.length) {
      if (snapshot.history.length > previous.history.length) {
        const last = snapshot.history[snapshot.history.length - 1];
        announce(`Jugada ${Math.ceil(snapshot.history.length / 2)}: ${last.san}. ${snapshot.check ? "Jaque. " : ""}${statusText(snapshot)}`);
      } else announce("La posición de la partida se ha actualizado.");
    }
  }

  function statusText(snapshot) {
    if (snapshot.outcome === "1-0") return "Ganan las blancas";
    if (snapshot.outcome === "0-1") return "Ganan las negras";
    if (snapshot.outcome === "1/2-1/2") return "Tablas";
    if (!snapshot.white || !snapshot.black) return "Esperando a tu rival";
    if (snapshot.check) return `Jaque a las ${snapshot.turn === "w" ? "blancas" : "negras"}`;
    return `Turno de las ${snapshot.turn === "w" ? "blancas" : "negras"}`;
  }

  function renderRoom() {
    const s = state.snapshot;
    if (!s) return;
    $("game-room-name").textContent = s.name || "Partida";
    $("game-room-code").textContent = s.id;
    $("role-badge").textContent = s.you === "w" ? "Juegas con blancas" : s.you === "b" ? "Juegas con negras" : "Espectador";
    $("game-phase").textContent = s.outcome !== "*" ? "PARTIDA TERMINADA" : !s.white || !s.black ? "EL TABLERO OS ESPERA" : "PARTIDA EN CURSO";
    $("game-status").textContent = statusText(s);
    $("game-substatus").textContent = s.outcome !== "*" ? `${s.reason || "Partida finalizada"} · ${s.outcome}` : !s.white || !s.black ? "Comparte el enlace y empieza en buena compañía." : s.you === "spectator" ? "Estás viendo la partida como espectador." : s.you === s.turn ? "Tu turno. Encuentra tu siguiente gran jugada." : "Tu rival está pensando. El tablero también es tuyo.";
    renderPlayers();
    renderHistory();
    renderPending();
    renderControls();
    renderBoard();
    updateInvitation();
  }

  function renderPlayers() {
    if (!state.snapshot) return;
    renderPlayer($("player-top"), state.flipped ? "w" : "b");
    renderPlayer($("player-bottom"), state.flipped ? "b" : "w");
  }
  function renderPlayer(element, color) {
    const s = state.snapshot;
    const player = color === "w" ? s.white : s.black;
    element.innerHTML = '<div class="player-avatar" aria-hidden="true"></div><div class="player-description"><div class="player-name"></div><div class="player-details"></div></div><span class="player-turn"></span>';
    element.querySelector(".player-avatar").innerHTML = window.NexoPieces.svg(color === "w" ? "K" : "k");
    element.querySelector(".player-name").textContent = player ? player.name : `Esperando a las ${color === "w" ? "blancas" : "negras"}`;
    const details = element.querySelector(".player-details");
    const dot = document.createElement("span");
    dot.className = `status-dot ${player && player.online ? "online-dot" : ""}`;
    dot.setAttribute("aria-hidden", "true");
    details.append(dot, document.createTextNode(`${color === "w" ? "Blancas" : "Negras"} · ${player ? player.online ? "Conectado" : "Ausente" : "Sitio libre"}${s.you === color ? " · Tú" : ""}`));
    const turn = element.querySelector(".player-turn");
    const active = s.outcome === "*" && s.turn === color && s.white && s.black;
    turn.classList.toggle("inactive", !active);
    turn.textContent = s.outcome !== "*" ? s.outcome === "1/2-1/2" ? "Tablas" : (s.outcome === "1-0" && color === "w") || (s.outcome === "0-1" && color === "b") ? "Victoria" : "Finalizada" : active ? s.you === color ? "Tu turno" : "En turno" : player ? "En espera" : "Por llegar";
  }

  function buildBoard() {
    for (let index = 0; index < 64; index++) {
      const square = document.createElement("button");
      square.type = "button";
      square.className = "square";
      square.setAttribute("role", "gridcell");
      square.tabIndex = -1;
      square.innerHTML = '<span class="square-piece" aria-hidden="true"></span><span class="square-rank" aria-hidden="true"></span><span class="square-file" aria-hidden="true"></span><span class="legal-marker" aria-hidden="true" hidden></span>';
      squares.push(square);
      $("board").append(square);
    }
  }
  buildBoard();

  function renderBoard() {
    const s = state.snapshot;
    if (!s) return;
    const reviewing = state.review !== null;
    const fen = reviewing ? state.review === 0 ? s.initialFen : s.history[state.review - 1].fen : s.fen;
    const board = boardFromFen(fen);
    const legal = canMove() && state.selected ? s.legal.filter((move) => move.from === state.selected) : [];
    const last = reviewing ? state.review ? s.history[state.review - 1] : null : s.history[s.history.length - 1];
    const checkedKing = !reviewing && s.check ? Object.keys(board).find((square) => board[square] === (s.turn === "w" ? "K" : "k")) : null;
    const moveAllowed = canMove();
    squares.forEach((button, index) => {
      const row = Math.floor(index / 8);
      const column = index % 8;
      const file = state.flipped ? 7 - column : column;
      const rank = state.flipped ? row + 1 : 8 - row;
      const square = `${files[file]}${rank}`;
      const piece = board[square] || "";
      const target = legal.some((move) => move.to === square);
      const capture = Boolean(piece || (state.selected && board[state.selected] && board[state.selected].toLowerCase() === "p" && state.selected[0] !== square[0]));
      button.dataset.square = square;
      button.className = `square${(file + rank) % 2 === 1 ? " dark" : ""}${last && (last.from === square || last.to === square) ? " last-move" : ""}${state.selected === square ? " selected" : ""}${checkedKing === square ? " checked" : ""}`;
      if (button.dataset.piece !== piece) {
        button.querySelector(".square-piece").innerHTML = window.NexoPieces.svg(piece);
        button.dataset.piece = piece;
      }
      button.querySelector(".square-rank").textContent = column === 0 ? rank : "";
      button.querySelector(".square-file").textContent = row === 7 ? files[file] : "";
      const marker = button.querySelector(".legal-marker");
      marker.hidden = !target;
      marker.className = `legal-marker ${capture ? "legal-ring" : "legal-dot"}`;
      button.setAttribute("aria-selected", String(state.selected === square));
      button.setAttribute("aria-disabled", String(!moveAllowed));
      button.setAttribute("aria-label", `${square}${piece ? `, ${pieceNames[piece.toLowerCase()]} ${colorOfPiece(piece) === "w" ? "de blancas" : "de negras"}` : ", vacía"}${target ? ", destino legal" : ""}${checkedKing === square ? ", en jaque" : ""}${state.selected === square ? ", seleccionada" : ""}`);
      button.tabIndex = square === state.focusSquare ? 0 : -1;
    });
    $("board").setAttribute("aria-label", `Tablero de ajedrez, ${state.flipped ? "negras" : "blancas"} abajo${reviewing ? `, posición tras ${state.review} medias jugadas` : ", en directo"}`);
    $("board-instructions").textContent = !state.online ? "Sin conexión. Espera a que Nexo vuelva a conectar." : reviewing ? "Estás revisando. Vuelve al directo para jugar." : s.you === "spectator" ? "Observa la partida. Usa el historial para revisarla." : s.outcome !== "*" ? "Partida terminada. Podéis acordar una nueva." : !s.white || !s.black ? "Invita a tu rival para empezar la partida." : s.you !== s.turn ? "Tu rival está pensando. Puedes revisar el historial." : state.selected ? "Elige uno de los destinos marcados." : "Selecciona una pieza. Flechas para recorrer el tablero.";
  }

  $("board").addEventListener("focusin", (event) => {
    const button = event.target.closest("button[data-square]");
    if (!button) return;
    state.focusSquare = button.dataset.square;
    squares.forEach((square) => { square.tabIndex = square === button ? 0 : -1; });
  });
  $("board").addEventListener("keydown", (event) => {
    const button = event.target.closest("button[data-square]");
    if (!button) return;
    let index = squares.indexOf(button);
    const key = event.key;
    if (key === "Escape") { state.selected = null; renderBoard(); return; }
    if (!["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown", "Home", "End"].includes(key)) return;
    event.preventDefault();
    if (key === "ArrowLeft" && index % 8 > 0) index--;
    if (key === "ArrowRight" && index % 8 < 7) index++;
    if (key === "ArrowUp" && index >= 8) index -= 8;
    if (key === "ArrowDown" && index < 56) index += 8;
    if (key === "Home") index = Math.floor(index / 8) * 8;
    if (key === "End") index = Math.floor(index / 8) * 8 + 7;
    squares[index].focus();
  });
  $("board").addEventListener("click", (event) => {
    const button = event.target.closest("button[data-square]");
    if (!button || !canMove() || state.promotion) return;
    const square = button.dataset.square;
    state.focusSquare = square;
    const s = state.snapshot;
    if (state.selected) {
      const moves = s.legal.filter((move) => move.from === state.selected && move.to === square);
      if (moves.length) {
        if (moves.some((move) => move.promotion)) showPromotion(moves);
        else movePiece(moves[0]);
        return;
      }
    }
    const board = boardFromFen(s.fen);
    if (colorOfPiece(board[square]) === s.you) {
      state.selected = state.selected === square ? null : square;
      if (state.selected) announce(`${pieceNames[board[square].toLowerCase()]} en ${square}, seleccionada. ${new Set(s.legal.filter((move) => move.from === square).map((move) => move.to)).size} destinos legales.`);
    } else state.selected = null;
    renderBoard();
  });
  $("flip-board").addEventListener("click", () => {
    state.flipped = !state.flipped;
    renderPlayers();
    renderBoard();
    announce(`Tablero girado. ${state.flipped ? "Negras" : "Blancas"} abajo.`);
  });

  function showPromotion(moves) {
    const version = state.snapshot.version;
    state.promotion = { moves, version };
    const container = $("promotion-choices");
    container.replaceChildren();
    for (const piece of ["q", "r", "b", "n"]) {
      const move = moves.find((candidate) => String(candidate.promotion).toLowerCase() === piece);
      const button = document.createElement("button");
      button.type = "button";
      button.className = "promotion-choice";
      button.disabled = !move;
      button.setAttribute("aria-label", `Coronar a ${promotionNames[piece].toLowerCase()}`);
      button.innerHTML = window.NexoPieces.svg(state.snapshot.you === "w" ? piece.toUpperCase() : piece);
      const label = document.createElement("span");
      label.textContent = promotionNames[piece];
      button.append(label);
      button.addEventListener("click", () => {
        if (!move || !canMove()) return;
        state.promotion = null;
        $("promotion-dialog").close();
        movePiece(move, version);
      });
      container.append(button);
    }
    $("promotion-dialog").showModal();
  }
  $("promotion-cancel").addEventListener("click", () => $("promotion-dialog").close());
  $("promotion-dialog").addEventListener("close", () => { state.promotion = null; });

  function movePiece(move, version = state.snapshot.version) {
    if (!canMove()) return;
    state.selected = null;
    mutation(async () => {
      const payload = { room: state.room, action: "move", from: move.from, to: move.to, version };
      if (move.promotion) payload.promotion = move.promotion;
      applySnapshot(await api("/api/action", "POST", payload));
    });
  }

  function renderHistory() {
    const s = state.snapshot;
    if (!s) return;
    $("move-counter").textContent = `${Math.ceil(s.history.length / 2)} ${Math.ceil(s.history.length / 2) === 1 ? "jugada" : "jugadas"}`;
    const signature = JSON.stringify([s.id, s.initialFen, s.history]);
    const list = $("history-list");
    if (signature !== state.historySignature) {
      const focusedPly = document.activeElement && document.activeElement.dataset.ply;
      state.historySignature = signature;
      list.replaceChildren();
      if (!s.history.length) list.innerHTML = '<p class="history-empty">Cada buena partida empieza<br>con una primera jugada.</p>';
      for (let index = 0; index < s.history.length; index += 2) {
        const row = document.createElement("div");
        row.className = "history-row";
        const number = document.createElement("span");
        number.className = "history-number";
        number.textContent = `${index / 2 + 1}.`;
        row.append(number);
        for (let offset = 0; offset < 2; offset++) {
          const move = s.history[index + offset];
          if (!move) { row.append(document.createElement("span")); continue; }
          const button = document.createElement("button");
          button.className = "history-move";
          button.type = "button";
          button.dataset.ply = index + offset + 1;
          button.textContent = move.san;
          button.setAttribute("aria-label", `Ver posición tras ${index / 2 + 1}, ${offset === 0 ? "blancas" : "negras"}: ${move.san}`);
          row.append(button);
        }
        list.append(row);
      }
      if (focusedPly) {
        const focused = [...list.querySelectorAll("button[data-ply]")].find((button) => button.dataset.ply === focusedPly);
        if (focused) focused.focus();
        else $("history-live").focus();
      }
      if (state.review === null) list.scrollTop = list.scrollHeight;
    }
    const activePly = state.review === null ? s.history.length : state.review;
    list.querySelectorAll("button[data-ply]").forEach((button) => {
      const active = Number(button.dataset.ply) === activePly;
      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", String(active));
    });
    $("history-live").classList.toggle("active", state.review === null);
    $("history-live").setAttribute("aria-pressed", String(state.review === null));
    $("review-note").hidden = state.review === null;
  }

  function reviewPosition(ply) {
    if (!state.snapshot || state.busy) return;
    state.review = ply === null ? null : Math.max(0, Math.min(ply, state.snapshot.history.length));
    state.selected = null;
    renderHistory(); renderControls(); renderBoard();
    announce(state.review === null ? "Tablero en directo." : state.review === 0 ? "Posición inicial." : `Posición tras ${state.review} medias jugadas.`);
  }
  $("history-list").addEventListener("click", (event) => {
    const button = event.target.closest("button[data-ply]");
    if (button) reviewPosition(Number(button.dataset.ply));
  });
  $("history-start").addEventListener("click", () => reviewPosition(0));
  $("history-back").addEventListener("click", () => { if (state.snapshot) reviewPosition((state.review === null ? state.snapshot.history.length : state.review) - 1); });
  $("history-forward").addEventListener("click", () => { if (state.snapshot && state.review !== null) reviewPosition(state.review + 1); });
  $("history-live").addEventListener("click", () => reviewPosition(null));

  function renderPending() {
    const s = state.snapshot;
    const pending = s && s.pending;
    $("pending-panel").hidden = !pending;
    if (!pending) return;
    const verbs = { undo: "deshacer la última jugada", restart: "empezar una nueva partida", draw: "acordar tablas" };
    const fromYou = pending.by === s.you;
    const player = pending.by === "w" ? s.white : s.black;
    $("pending-text").textContent = fromYou ? `Has propuesto ${verbs[pending.kind] || "un cambio"}. Esperando a tu rival.` : `${player ? player.name : "Tu rival"} propone ${verbs[pending.kind] || "un cambio"}.`;
    $("pending-actions").hidden = fromYou || s.you === "spectator";
  }

  function renderControls() {
    const s = state.snapshot;
    const connected = state.online && !state.busy && !state.stopped;
    $("create-submit").disabled = !connected;
    $("join-submit").disabled = !connected;
    $("create-submit").firstChild.textContent = state.busy && !s ? "Creando… " : "Crear partida ";
    $("join-submit").firstChild.textContent = state.busy && !s ? "Entrando… " : "Entrar en la sala ";
    $("rooms-list").querySelectorAll("button").forEach((button) => { button.disabled = !connected; });
    $("shutdown").disabled = !connected;
    if (!s) return;
    const actor = canAct();
    const active = s.outcome === "*";
    $("player-actions").hidden = s.you === "spectator";
    $("undo-action").disabled = !actor || !s.history.length || !!s.pending;
    $("restart-action").disabled = !actor || !!s.pending;
    $("draw-action").disabled = !actor || !active || !!s.pending;
    $("resign-action").disabled = !actor || !active;
    $("claim-action").hidden = !s.canClaim || s.you === "spectator";
    $("claim-action").disabled = !actor || !active || !s.canClaim;
    const responder = actor && s.pending && s.pending.by !== s.you;
    $("accept-proposal").disabled = !responder;
    $("reject-proposal").disabled = !responder;
    $("leave-room").disabled = !connected;
    $("history-start").disabled = state.busy || state.review === 0;
    $("history-back").disabled = state.busy || (state.review === null ? s.history.length : state.review) === 0;
    $("history-forward").disabled = state.busy || state.review === null || state.review >= s.history.length;
    $("history-live").disabled = state.busy;
    $("download-pgn").disabled = !s.pgn;
    $("flip-board").disabled = state.busy;
  }

  function gameAction(action, fields = {}) {
    if (!canAct()) return;
    const version = state.snapshot.version;
    mutation(async () => {
      applySnapshot(await api("/api/action", "POST", { room: state.room, action, version, ...fields }));
    });
  }
  $("undo-action").addEventListener("click", () => gameAction("undo"));
  $("restart-action").addEventListener("click", () => gameAction("restart"));
  $("draw-action").addEventListener("click", () => gameAction("draw"));
  $("claim-action").addEventListener("click", () => gameAction("claim"));
  $("accept-proposal").addEventListener("click", () => gameAction("respond", { accept: true }));
  $("reject-proposal").addEventListener("click", () => gameAction("respond", { accept: false }));

  function confirm(title, description, label, action, eyebrow = "TU PARTIDA") {
    $("confirm-title").textContent = title;
    $("confirm-description").textContent = description;
    $("confirm-accept").textContent = label;
    $("confirm-eyebrow").textContent = eyebrow;
    confirmAction = action;
    $("confirm-dialog").showModal();
    $("confirm-cancel").focus();
  }
  $("confirm-cancel").addEventListener("click", () => $("confirm-dialog").close());
  $("confirm-dialog").addEventListener("close", () => { confirmAction = null; });
  $("confirm-accept").addEventListener("click", () => {
    const action = confirmAction;
    $("confirm-dialog").close();
    if (action) action();
  });
  $("resign-action").addEventListener("click", () => {
    if (!canAct()) return;
    confirm("¿Abandonar esta partida?", "La partida terminará con tu rendición. Podréis acordar una nueva partida después.", "Abandonar partida", () => gameAction("resign"));
  });

  function leaveRoom() {
    if (!state.room || state.busy || !state.online) return;
    const room = state.room;
    mutation(async () => {
      await api("/api/action", "POST", { room, action: "leave" });
      showLobby(); setRoomHash(null); await refreshLobby();
      notify("Has salido de la sala. Tu sitio vuelve a estar disponible.");
    });
  }
  function askLeave() {
    if (!state.snapshot) return;
    if (state.snapshot.you === "spectator") { leaveRoom(); return; }
    confirm("¿Salir de la sala?", "Dejarás tu sitio libre. La partida seguirá en la sala y podrás volver a entrar si hay sitio.", "Salir de la sala", leaveRoom);
  }
  $("leave-room").addEventListener("click", askLeave);
  $("brand-home").addEventListener("click", (event) => {
    event.preventDefault();
    if (state.snapshot) askLeave();
    else { switchTab("create"); setRoomHash(null); }
  });

  $("shutdown").addEventListener("click", () => {
    if (!state.info || !state.info.isHost || state.busy) return;
    confirm("¿Cerrar Nexo Chess?", "Se cerrará la aplicación y terminarán todas las salas de esta red. Las partidas que no hayas descargado se perderán.", "Cerrar Nexo", () => {
      mutation(async () => {
        await api("/api/shutdown", "POST", {});
        state.stopped = true; state.online = false;
        stopPolling(); renderConnection(); renderControls();
        notify("Nexo Chess está cerrado. Puedes cerrar esta pestaña.");
      });
    }, "EN ESTE ORDENADOR");
  });

  $("download-pgn").addEventListener("click", () => {
    if (!state.snapshot || !state.snapshot.pgn) return;
    const blob = new Blob([state.snapshot.pgn], { type: "application/x-chess-pgn;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `Nexo-Chess-${state.room}.pgn`;
    document.body.append(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  });

  if (window.NexoPieces) $("hero-piece").innerHTML = window.NexoPieces.svg("N");
  switchTab("create");
  renderControls();
  async function start() {
    try {
      state.info = await api("/api/info");
      renderInfo();
      await routeFromHash();
      state.ready = true;
      renderControls();
    } catch (error) {
      state.ready = true;
      renderConnection();
      notify(error.message, "error");
      schedulePoll();
    }
  }
  start();
})();

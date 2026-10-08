/* Pure Node/JSDOM contract tests. No browser, server, socket, or real fetch is used.
 * Install JSDOM separately from the portable app, then run:
 *   node --test tests/frontend.test.cjs
 * NODE_PATH or NEXO_JSDOM_PATH may point at an existing JSDOM installation.
 */
"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");

function loadJSDOM() {
  const candidates = [process.env.NEXO_JSDOM_PATH, "jsdom", "D:/Temp/ajedrez-build/dom/node_modules/jsdom"].filter(Boolean);
  for (const candidate of candidates) {
    try { return require(candidate).JSDOM; } catch (error) {
      if (error.code !== "MODULE_NOT_FOUND") throw error;
    }
  }
  throw new Error("JSDOM is a development-only prerequisite. Set NEXO_JSDOM_PATH to its installed module path.");
}

const JSDOM = loadJSDOM();
const web = path.join(__dirname, "../web");
const html = fs.readFileSync(path.join(web, "index.html"), "utf8");
const pieces = fs.readFileSync(path.join(web, "pieces.js"), "utf8");
const app = fs.readFileSync(path.join(web, "app.js"), "utf8");
const clone = (value) => JSON.parse(JSON.stringify(value));
const INITIAL = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1";
const AFTER_E4 = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1";
const AFTER_E5 = "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2";
const OPENING = [
  { san: "e4", from: "e2", to: "e4", fen: AFTER_E4 },
  { san: "e5", from: "e7", to: "e5", fen: AFTER_E5 }
];

function snapshot(fields = {}) {
  return {
    id: "TEST123", name: "Partida de prueba", you: "w",
    white: { name: "Manu", online: true }, black: { name: "Rival", online: true },
    fen: INITIAL, initialFen: INITIAL, turn: "w", check: false,
    outcome: "*", reason: "", legal: [
      { from: "e2", to: "e3" }, { from: "e2", to: "e4" },
      { from: "g1", to: "f3" }, { from: "g1", to: "h3" }
    ], history: [], version: 5, pending: null, canClaim: false,
    pgn: '[Event "Partida de prueba"]\n\n*', ...fields
  };
}

async function settle() {
  // Flush asynchronous handlers without running app poll/abort timers.
  for (let index = 0; index < 5; index++) await new Promise(setImmediate);
}

async function harness(initial = snapshot(), options = {}) {
  const dom = new JSDOM(html, {
    url: `http://nexo.invalid/${options.lobby ? "" : `#room=${initial.id}`}`,
    runScripts: "outside-only"
    // No `resources`: scripts/styles in the document never load.
  });
  const { window } = dom;
  const document = window.document;
  const logs = [];
  const errors = [];
  const timers = new Map();
  const handlers = new Map();
  let timerId = 0;
  let serverSnapshot = clone(initial);
  let sessionCount = 0;
  let activeRequests = 0;
  let maximumRequests = 0;
  window.setTimeout = (fn, delay) => { const id = ++timerId; timers.set(id, { fn, delay }); return id; };
  window.clearTimeout = (id) => timers.delete(id);
  window.requestAnimationFrame = (fn) => window.setTimeout(() => fn(0), 16);
  window.cancelAnimationFrame = window.clearTimeout;
  const media = { matches: Boolean(options.dark), addEventListener() {}, removeEventListener() {} };
  window.matchMedia = () => media;
  window.HTMLDialogElement.prototype.showModal = function () { this.open = true; };
  window.HTMLDialogElement.prototype.close = function () {
    if (!this.open) return;
    this.open = false;
    this.dispatchEvent(new window.Event("close"));
  };
  window.addEventListener("error", (event) => { errors.push(event.error || event.message); });
  if (options.token) window.sessionStorage.setItem("nexo-session", options.token);
  Object.defineProperty(window, "isSecureContext", { value: false });
  let clipboardCalls = 0;
  Object.defineProperty(window.navigator, "clipboard", { value: { async writeText() { clipboardCalls++; } } });
  window.fetch = async (url, request = {}) => {
    const endpoint = String(url).split("?")[0];
    const body = request.body ? JSON.parse(request.body) : null;
    const log = { url: String(url), endpoint, method: request.method, headers: clone(request.headers), body };
    logs.push(log);
    activeRequests++;
    maximumRequests = Math.max(maximumRequests, activeRequests);
    try {
      let payload;
      let status = 200;
      if (handlers.has(endpoint)) {
        const result = await handlers.get(endpoint)(log);
        if (result && result.status) { status = result.status; payload = result.payload; }
        else payload = result;
      } else if (endpoint === "/api/session") payload = { token: `TEST-TOKEN-${++sessionCount}` };
      else if (endpoint === "/api/info") payload = { name: "Nexo Chess", version: "test", urls: ["http://192.168.1.20:8765"], isHost: false };
      else if (endpoint === "/api/lobby") payload = { rooms: [] };
      else if (endpoint === "/api/state") payload = serverSnapshot;
      else if (endpoint === "/api/action" || endpoint === "/api/rooms" || endpoint === "/api/join") payload = serverSnapshot;
      else throw new Error(`Unexpected mock request: ${endpoint}`);
      const copy = clone(payload);
      return { ok: status >= 200 && status < 300, status, async json() { return copy; } };
    } finally { activeRequests--; }
  };
  window.eval(pieces);
  window.eval(app);
  await settle();
  assert.deepEqual(errors, [], "initialization must not throw DOM errors");

  const h = {
    window, document, logs, errors, media,
    $(id) { return document.getElementById(id); },
    square(coordinate) { return document.querySelector(`#board button[data-square="${coordinate}"]`); },
    click(id) { this.$(id).click(); },
    actions() { return logs.filter((entry) => entry.endpoint === "/api/action"); },
    on(endpoint, fn) { handlers.set(endpoint, fn); },
    setSnapshot(next) { serverSnapshot = clone(next); },
    async poll(next) {
      if (next) this.setSnapshot(next);
      const poll = [...timers].find(([, value]) => value.delay === 1000);
      assert.ok(poll, "a single poll must be scheduled");
      timers.delete(poll[0]); poll[1].fn(); await settle();
    },
    pendingPolls() { return [...timers.values()].filter((timer) => timer.delay === 1000).length; },
    get maximumRequests() { return maximumRequests; },
    get clipboardCalls() { return clipboardCalls; },
    close() { dom.window.close(); },
    check() { assert.deepEqual(errors, [], "frontend must not throw DOM errors"); }
  };
  return h;
}

async function withHarness(initial, work, options) {
  const h = await harness(initial, options);
  try { await work(h); h.check(); } finally { h.close(); }
}

test("white and black auto-orientations preserve all coordinates and a1 dark", async () => {
  for (const color of ["w", "b"]) {
    await withHarness(snapshot({ you: color }), async (h) => {
      const cells = [...h.$("board").querySelectorAll("button")];
      assert.equal(cells.length, 64);
      assert.equal(new Set(cells.map((cell) => cell.dataset.square)).size, 64);
      assert.equal(cells[0].dataset.square, color === "w" ? "a8" : "h1");
      assert.equal(cells[63].dataset.square, color === "w" ? "h1" : "a8");
      assert.ok(h.square("a1").classList.contains("dark"));
      assert.ok(!h.square("h1").classList.contains("dark"));
      assert.ok(!h.square("a8").classList.contains("dark"));
      assert.match(h.$("player-bottom").textContent, color === "w" ? /Manu.*Blancas/ : /Rival.*Negras/);
      assert.equal(h.square("e1").dataset.piece, "K");
      assert.equal(h.square("e8").dataset.piece, "k");
      assert.equal(h.square("e1").querySelectorAll("svg.piece-white").length, 1);
      assert.equal(h.square("e8").querySelectorAll("svg.piece-black").length, 1);
      h.click("flip-board");
      assert.equal(h.$("board").firstChild.dataset.square, color === "w" ? "h1" : "a8");
      assert.ok(h.square("a1").classList.contains("dark"));
      assert.equal(h.square("a1").querySelector(".square-file").textContent, color === "w" ? "" : "a");
      assert.equal(h.actions().length, 0);
    });
  }
});

test("server legal moves alone become markers; moves carry authentication and version", async () => {
  await withHarness(snapshot(), async (h) => {
    h.square("e2").click();
    assert.equal(h.square("e2").getAttribute("aria-selected"), "true");
    assert.equal(h.square("e3").querySelector(".legal-marker").hidden, false);
    assert.equal(h.square("e4").querySelector(".legal-marker").hidden, false);
    assert.equal(h.square("e5").querySelector(".legal-marker").hidden, true);
    h.square("e5").click();
    assert.equal(h.actions().length, 0, "an illegal destination never makes a network mutation");
    h.square("e2").click();
    h.setSnapshot(snapshot({ fen: AFTER_E4, turn: "b", history: OPENING.slice(0, 1), version: 6, legal: [{ from: "e7", to: "e5" }] }));
    h.square("e4").click();
    await settle();
    const action = h.actions()[0];
    assert.deepEqual(action.body, { room: "TEST123", action: "move", from: "e2", to: "e4", version: 5 });
    assert.equal(action.method, "POST");
    assert.equal(action.headers["Content-Type"], "application/json");
    assert.equal(action.headers["X-Nexo-Client"], "1");
    assert.match(action.headers.Authorization, /^Bearer TEST-TOKEN-/);
    assert.equal(h.square("e2").dataset.piece, "");
    assert.equal(h.square("e4").dataset.piece, "P");
    assert.equal(h.$("game-status").textContent, "Turno de las negras");
    assert.ok(h.square("e4").classList.contains("last-move"));
    assert.equal(h.square("e7").getAttribute("aria-disabled"), "true");
  });
});

test("black can move from its automatically flipped board on its turn", async () => {
  await withHarness(snapshot({ you: "b", fen: AFTER_E4, turn: "b", history: OPENING.slice(0, 1), legal: [{ from: "e7", to: "e5" }] }), async (h) => {
    h.setSnapshot(snapshot({ you: "b", fen: AFTER_E5, turn: "w", history: OPENING, version: 6 }));
    h.square("e7").click(); h.square("e5").click(); await settle();
    assert.equal(h.actions().length, 1);
    assert.equal(h.actions()[0].body.from, "e7");
    assert.equal(h.actions()[0].body.to, "e5");
    assert.equal(h.square("e5").dataset.piece, "p");
    assert.equal(h.$("board").firstChild.dataset.square, "h1");
  });
});

test("captures use a ring and replace the captured SVG with the resulting server piece", async () => {
  const fen = "7k/8/8/4p3/4R3/8/8/K7 w - - 0 1";
  const after = "7k/8/8/4R3/8/8/8/K7 b - - 0 1";
  await withHarness(snapshot({ fen, initialFen: fen, legal: [{ from: "e4", to: "e5" }] }), async (h) => {
    h.square("e4").click();
    const marker = h.square("e5").querySelector(".legal-marker");
    assert.equal(marker.hidden, false); assert.ok(marker.classList.contains("legal-ring"));
    h.setSnapshot(snapshot({ fen: after, initialFen: fen, turn: "b", version: 6, history: [{ san: "Rxe5", from: "e4", to: "e5", fen: after }] }));
    h.square("e5").click(); await settle();
    assert.equal(h.square("e4").dataset.piece, "");
    assert.equal(h.square("e5").dataset.piece, "R");
    assert.equal(h.square("e5").querySelectorAll("svg").length, 1);
    assert.equal(h.$("history-list").querySelector("button").textContent, "Rxe5");
  });
});

test("castling renders both resulting king and rook positions from the authoritative FEN", async () => {
  const fen = "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1";
  const after = "r3k2r/8/8/8/8/8/8/R4RK1 b kq - 1 1";
  await withHarness(snapshot({ fen, initialFen: fen, legal: [{ from: "e1", to: "g1" }, { from: "e1", to: "c1" }] }), async (h) => {
    h.square("e1").click();
    assert.equal(h.square("g1").querySelector(".legal-marker").hidden, false);
    assert.equal(h.square("c1").querySelector(".legal-marker").hidden, false);
    h.setSnapshot(snapshot({ fen: after, initialFen: fen, turn: "b", version: 6, history: [{ san: "O-O", from: "e1", to: "g1", fen: after }] }));
    h.square("g1").click(); await settle();
    assert.deepEqual(h.actions()[0].body, { room: "TEST123", action: "move", from: "e1", to: "g1", version: 5 });
    assert.equal(h.square("e1").dataset.piece, "");
    assert.equal(h.square("h1").dataset.piece, "");
    assert.equal(h.square("g1").dataset.piece, "K");
    assert.equal(h.square("f1").dataset.piece, "R");
    assert.equal(h.$("history-list").querySelector("button").textContent, "O-O");
  });
});

test("en passant uses a capture marker on its empty target and removes the bypassed pawn", async () => {
  const fen = "7k/8/8/3pP3/8/8/8/K7 w - d6 0 1";
  const after = "7k/8/3P4/8/8/8/8/K7 b - - 0 1";
  await withHarness(snapshot({ fen, initialFen: fen, legal: [{ from: "e5", to: "d6" }] }), async (h) => {
    h.square("e5").click();
    assert.equal(h.square("d6").dataset.piece, "");
    const marker = h.square("d6").querySelector(".legal-marker");
    assert.equal(marker.hidden, false);
    assert.ok(marker.classList.contains("legal-ring"));
    h.setSnapshot(snapshot({ fen: after, initialFen: fen, turn: "b", version: 6, history: [{ san: "exd6", from: "e5", to: "d6", fen: after }] }));
    h.square("d6").click(); await settle();
    assert.equal(h.square("e5").dataset.piece, "");
    assert.equal(h.square("d5").dataset.piece, "");
    assert.equal(h.square("d6").dataset.piece, "P");
    assert.equal(h.actions()[0].body.to, "d6");
  });
});

test("promotion presents four real choices and submits the selected underpromotion", async () => {
  for (const choice of ["q", "r", "b", "n"]) {
    const fen = "7k/P7/8/8/8/8/8/7K w - - 0 1";
    const legal = ["q", "r", "b", "n"].map((promotion) => ({ from: "a7", to: "a8", promotion }));
    await withHarness(snapshot({ fen, initialFen: fen, legal }), async (h) => {
      h.square("a7").click(); h.square("a8").click();
      assert.equal(h.$("promotion-dialog").open, true);
      const buttons = [...h.$("promotion-choices").querySelectorAll("button")];
      assert.equal(buttons.length, 4);
      assert.deepEqual(buttons.map((button) => button.textContent.trim()), ["Dama", "Torre", "Alfil", "Caballo"]);
      assert.ok(buttons.every((button) => !button.disabled));
      assert.equal(h.actions().length, 0, "opening the dialog must not auto-promote");
      const after = `${choice.toUpperCase()}6k/8/8/8/8/8/8/7K b - - 0 1`;
      h.setSnapshot(snapshot({ fen: after, initialFen: fen, turn: "b", version: 6, legal: [], history: [{ san: `a8=${choice.toUpperCase()}`, from: "a7", to: "a8", fen: after }] }));
      buttons[["q", "r", "b", "n"].indexOf(choice)].click(); await settle();
      assert.equal(h.actions().length, 1);
      assert.equal(h.actions()[0].body.promotion, choice);
      assert.equal(h.actions()[0].body.version, 5);
      assert.equal(h.$("promotion-dialog").open, false);
      assert.equal(h.square("a8").dataset.piece, choice.toUpperCase());
    });
  }
});

test("cancelled promotion sends no action and legal moves can be selected again", async () => {
  const fen = "7k/P7/8/8/8/8/8/7K w - - 0 1";
  await withHarness(snapshot({ fen, initialFen: fen, legal: ["q", "r", "b", "n"].map((promotion) => ({ from: "a7", to: "a8", promotion })) }), async (h) => {
    h.square("a7").click(); h.square("a8").click(); h.click("promotion-cancel");
    assert.equal(h.$("promotion-dialog").open, false);
    assert.equal(h.actions().length, 0);
    h.square("a8").click();
    assert.equal(h.$("promotion-dialog").open, true);
  });
});

test("check, mate, and stalemate render meaningful status without allowing finished moves", async () => {
  await withHarness(snapshot({ fen: "7k/8/8/8/8/8/7r/7K w - - 0 1", check: true, legal: [{ from: "h1", to: "g1" }] }), async (h) => {
    assert.equal(h.$("game-status").textContent, "Jaque a las blancas");
    assert.ok(h.square("h1").classList.contains("checked"));
    assert.match(h.square("h1").getAttribute("aria-label"), /en jaque/);
    await h.poll(snapshot({ fen: "7k/8/8/8/8/8/6r1/6qK w - - 0 1", check: true, outcome: "0-1", reason: "Jaque mate", version: 6, legal: [] }));
    assert.equal(h.$("game-status").textContent, "Ganan las negras");
    assert.match(h.$("game-substatus").textContent, /Jaque mate.*0-1/);
    assert.equal(h.$("game-phase").textContent, "PARTIDA TERMINADA");
    assert.equal(h.$("draw-action").disabled, true);
    h.square("h1").click(); h.square("g1").click(); assert.equal(h.actions().length, 0);
    await h.poll(snapshot({ fen: "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1", turn: "b", outcome: "1/2-1/2", reason: "Ahogado", version: 7, legal: [] }));
    assert.equal(h.$("game-status").textContent, "Tablas");
    assert.match(h.$("game-substatus").textContent, /Ahogado/);
  });
});

test("history shows the actual older FEN, disables moves, preserves review while polling, and resumes live", async () => {
  const live = snapshot({ fen: AFTER_E5, history: OPENING, version: 7, legal: [{ from: "g1", to: "f3" }] });
  await withHarness(live, async (h) => {
    assert.equal(h.$("history-list").querySelectorAll("button").length, 2);
    h.click("history-back");
    assert.equal(h.square("e5").dataset.piece, "");
    assert.equal(h.square("e7").dataset.piece, "p");
    assert.equal(h.square("e4").dataset.piece, "P");
    assert.equal(h.$("review-note").hidden, false);
    assert.equal(h.$("undo-action").disabled, true);
    assert.equal(h.square("g1").getAttribute("aria-disabled"), "true");
    h.square("g1").click(); h.square("f3").click(); assert.equal(h.actions().length, 0);
    await h.poll({ ...live, black: { name: "Rival", online: false } });
    assert.equal(h.square("e5").dataset.piece, "", "presence polling must not jump review to live");
    h.click("history-start");
    assert.equal(h.square("e2").dataset.piece, "P");
    assert.equal(h.square("e4").dataset.piece, "");
    assert.equal(h.$("history-back").disabled, true);
    h.click("history-forward"); assert.equal(h.square("e4").dataset.piece, "P");
    h.click("history-live");
    assert.equal(h.$("review-note").hidden, true);
    assert.equal(h.square("e5").dataset.piece, "p");
    assert.equal(h.square("g1").getAttribute("aria-disabled"), "false");
    h.square("g1").click(); assert.equal(h.square("f3").querySelector(".legal-marker").hidden, false);
  });
});

test("duplicate destination clicks and polls cannot overlap an outstanding move request", async () => {
  await withHarness(snapshot(), async (h) => {
    let release;
    h.on("/api/action", () => new Promise((resolve) => { release = resolve; }));
    h.square("e2").click(); h.square("e4").click(); h.square("e4").click();
    await settle();
    assert.equal(h.actions().length, 1);
    assert.equal(h.$("undo-action").disabled, true);
    assert.equal(h.$("flip-board").disabled, true);
    assert.equal(h.$("history-back").disabled, true);
    assert.equal(h.pendingPolls(), 0, "mutations stop the scheduled poll");
    release(snapshot({ fen: AFTER_E4, turn: "b", version: 6, history: OPENING.slice(0, 1) })); await settle();
    assert.equal(h.maximumRequests, 1);
    assert.equal(h.pendingPolls(), 1, "completion schedules exactly one poll");
    assert.equal(h.actions().length, 1);
  });
});

test("a click during an outstanding poll queues one mutation after that poll, never a parallel fetch", async () => {
  await withHarness(snapshot(), async (h) => {
    let release;
    h.on("/api/state", () => new Promise((resolve) => { release = resolve; }));
    const after = snapshot({ fen: AFTER_E4, turn: "b", version: 6, history: OPENING.slice(0, 1) });
    h.on("/api/action", async () => after);
    await h.poll();
    assert.equal(typeof release, "function");
    h.square("e2").click(); h.square("e4").click(); h.square("e4").click(); await settle();
    assert.equal(h.actions().length, 0, "mutation must await the request already in progress");
    assert.equal(h.maximumRequests, 1);
    release(snapshot()); await settle();
    assert.equal(h.actions().length, 1);
    assert.equal(h.actions()[0].body.version, 5);
    assert.equal(h.maximumRequests, 1);
    assert.equal(h.square("e4").dataset.piece, "P");
    assert.equal(h.pendingPolls(), 1);
  });
});

test("keyboard navigation follows visible board columns and Escape clears legal markers", async () => {
  await withHarness(snapshot(), async (h) => {
    const key = (value) => h.document.activeElement.dispatchEvent(new h.window.KeyboardEvent("keydown", { key: value, bubbles: true, cancelable: true }));
    h.square("e2").focus(); h.square("e2").click();
    assert.equal(h.square("e4").querySelector(".legal-marker").hidden, false);
    key("ArrowRight"); assert.equal(h.document.activeElement.dataset.square, "f2");
    key("Home"); assert.equal(h.document.activeElement.dataset.square, "a2");
    key("End"); assert.equal(h.document.activeElement.dataset.square, "h2");
    key("ArrowRight"); assert.equal(h.document.activeElement.dataset.square, "h2");
    key("Escape"); assert.equal(h.square("e4").querySelector(".legal-marker").hidden, true);
    h.click("flip-board"); h.square("e2").focus();
    key("ArrowRight"); assert.equal(h.document.activeElement.dataset.square, "d2");
    assert.equal(h.$("board").querySelectorAll('button[tabindex="0"]').length, 1);
    assert.equal(h.actions().length, 0);
  });
});

test("equal-version snapshots refresh presence and lower-version responses cannot roll the board back", async () => {
  const live = snapshot({ fen: AFTER_E5, history: OPENING, version: 10 });
  await withHarness(live, async (h) => {
    await h.poll({ ...live, black: { name: "Rival", online: false } });
    assert.match(h.$("player-top").textContent, /Rival.*Ausente/);
    await h.poll(snapshot({ version: 9, black: { name: "Outdated", online: true } }));
    assert.equal(h.square("e4").dataset.piece, "P");
    assert.equal(h.square("e5").dataset.piece, "p");
    assert.equal(h.$("history-list").querySelectorAll("button").length, 2);
    assert.match(h.$("player-top").textContent, /Rival.*Ausente/);
    assert.ok(!h.$("player-top").textContent.includes("Outdated"));
    assert.equal(h.pendingPolls(), 1);
  });
});

test("theme button toggles the DOM theme and accessible label", async () => {
  await withHarness(snapshot(), async (h) => {
    assert.equal(h.document.documentElement.dataset.theme, "dark");
    assert.match(h.$("theme-toggle").getAttribute("aria-label"), /claro/);
    h.click("theme-toggle");
    assert.equal(h.document.documentElement.dataset.theme, "light");
    assert.match(h.$("theme-toggle").getAttribute("aria-label"), /oscuro/);
    h.click("theme-toggle"); assert.equal(h.document.documentElement.dataset.theme, "dark");
    assert.equal(h.actions().length, 0);
  }, { dark: true });
});

test("spectators cannot select pieces, mutate the game, or answer proposals", async () => {
  await withHarness(snapshot({ you: "spectator", pending: { kind: "draw", by: "b" } }), async (h) => {
    assert.equal(h.$("role-badge").textContent, "Espectador");
    assert.equal(h.$("player-actions").hidden, true);
    assert.equal(h.$("pending-actions").hidden, true);
    assert.equal(h.$("accept-proposal").disabled, true);
    assert.equal(h.square("e2").getAttribute("aria-disabled"), "true");
    h.square("e2").click(); h.square("e4").click();
    h.click("draw-action"); h.click("accept-proposal");
    assert.equal(h.square("e2").getAttribute("aria-selected"), "false");
    assert.equal(h.actions().length, 0);
    h.click("flip-board"); assert.equal(h.$("board").firstChild.dataset.square, "h1");
  });
});

test("reload uses the tab session token and restores the room without creating another session", async () => {
  await withHarness(snapshot({ you: "b" }), async (h) => {
    assert.equal(h.logs.filter((entry) => entry.endpoint === "/api/session").length, 0);
    assert.ok(h.logs.every((entry) => entry.headers.Authorization === "Bearer EXISTING-TAB-TOKEN"));
    assert.equal(h.window.sessionStorage.getItem("nexo-session"), "EXISTING-TAB-TOKEN");
    assert.equal(h.$("game-view").hidden, false);
    assert.equal(h.$("game-room-code").textContent, "TEST123");
    assert.equal(h.$("role-badge").textContent, "Juegas con negras");
  }, { token: "EXISTING-TAB-TOKEN" });
});

test("a temporary network failure reconnects using the same tab token and restores controls", async () => {
  await withHarness(snapshot(), async (h) => {
    const token = h.window.sessionStorage.getItem("nexo-session");
    h.on("/api/state", async () => { throw new h.window.TypeError("simulated offline"); });
    await h.poll();
    assert.equal(h.$("connection-label").textContent, "Reconectando");
    assert.equal(h.square("e2").getAttribute("aria-disabled"), "true");
    assert.equal(h.$("draw-action").disabled, true);
    h.on("/api/state", async () => snapshot());
    await h.poll();
    assert.equal(h.$("connection-label").textContent, "En tu red");
    assert.equal(h.square("e2").getAttribute("aria-disabled"), "false");
    assert.equal(h.$("draw-action").disabled, false);
    assert.equal(h.window.sessionStorage.getItem("nexo-session"), token);
    assert.equal(h.logs.filter((entry) => entry.endpoint === "/api/session").length, 1);
  });
});

test("a committed create with a lost response recovers its seat and room from authenticated lobby membership without replaying create", async () => {
  const created = snapshot({ you: "b", white: null, black: { name: "Manu", online: true }, version: 1, legal: [] });
  await withHarness(created, async (h) => {
    assert.equal(h.$("lobby-view").hidden, false);
    h.on("/api/rooms", async () => {
      // The server has already committed this create before delivery fails.
      h.setSnapshot(created);
      throw new h.window.TypeError("response delivery lost after commit");
    });
    h.$("player-name").value = "Manu";
    h.$("room-name").value = "Partida de prueba";
    h.document.querySelector('input[name="create-color"][value="b"]').checked = true;
    h.$("create-form").dispatchEvent(new h.window.Event("submit", { bubbles: true, cancelable: true }));
    await settle();
    assert.equal(h.logs.filter((entry) => entry.endpoint === "/api/rooms").length, 1);
    assert.equal(h.$("game-view").hidden, true, "lost response has not supplied a room snapshot");
    assert.equal(h.window.location.hash, "");
    assert.equal(h.$("connection-label").textContent, "Reconectando");
    const token = h.window.sessionStorage.getItem("nexo-session");
    h.on("/api/lobby", async () => ({
      rooms: [{ id: created.id, name: created.name, white: null, black: "Manu", ply: 0, status: "*" }],
      activeRoom: created.id
    }));
    await h.poll();
    const states = h.logs.filter((entry) => entry.endpoint === "/api/state");
    assert.equal(states.length, 1);
    assert.equal(states[0].url, "/api/state?room=TEST123");
    assert.equal(states[0].headers.Authorization, `Bearer ${token}`);
    assert.equal(h.$("game-view").hidden, false);
    assert.equal(h.$("lobby-view").hidden, true);
    assert.equal(h.$("game-room-code").textContent, "TEST123");
    assert.equal(h.$("role-badge").textContent, "Juegas con negras");
    assert.match(h.$("player-bottom").textContent, /Manu.*Negras/);
    assert.equal(h.$("board").firstChild.dataset.square, "h1");
    assert.equal(h.window.location.hash, "#room=TEST123");
    assert.match(h.$("notice-text").textContent, /sala se ha recuperado/);
    assert.equal(h.$("connection-label").textContent, "En tu red");
    assert.equal(h.logs.filter((entry) => entry.endpoint === "/api/rooms").length, 1);
    assert.equal(h.logs.filter((entry) => entry.endpoint === "/api/join").length, 0);
    assert.equal(h.actions().length, 0);
    assert.equal(h.window.sessionStorage.getItem("nexo-session"), token);
  }, { lobby: true });
});

test("lobby rooms alone never restore another room when activeRoom is absent or null", async () => {
  for (const membership of [{}, { activeRoom: null }]) {
    await withHarness(snapshot(), async (h) => {
      h.on("/api/lobby", async () => ({
        rooms: [{ id: "OTHER123", name: "Partida ajena", white: "Otro", black: "Otra", ply: 2, status: "*" }],
        ...membership
      }));
      await h.poll();
      assert.equal(h.$("lobby-view").hidden, false);
      assert.equal(h.$("game-view").hidden, true);
      assert.equal(h.$("room-count").textContent, "1");
      assert.match(h.$("rooms-list").textContent, /Partida ajena/);
      assert.equal(h.window.location.hash, "");
      assert.equal(h.logs.filter((entry) => entry.endpoint === "/api/state").length, 0);
      assert.equal(h.logs.filter((entry) => entry.endpoint === "/api/rooms").length, 0);
      assert.equal(h.logs.filter((entry) => entry.endpoint === "/api/join").length, 0);
      assert.equal(h.actions().length, 0);
    }, { lobby: true });
  }
});

test("accepting and rejecting opponent proposals include the exact current version", async () => {
  for (const accept of [true, false]) {
    const pending = snapshot({ pending: { kind: "draw", by: "b" }, version: 17 });
    await withHarness(pending, async (h) => {
      assert.equal(h.$("pending-panel").hidden, false);
      assert.equal(h.$("pending-actions").hidden, false);
      assert.match(h.$("pending-text").textContent, /Rival propone acordar tablas/);
      h.setSnapshot({ ...pending, pending: null, version: 18, ...(accept ? { outcome: "1/2-1/2", reason: "Tablas acordadas" } : {}) });
      h.click(accept ? "accept-proposal" : "reject-proposal"); await settle();
      assert.deepEqual(h.actions()[0].body, { room: "TEST123", action: "respond", version: 17, accept });
      assert.equal(h.$("pending-panel").hidden, true);
      assert.equal(h.$("game-status").textContent, accept ? "Tablas" : "Turno de las blancas");
    });
  }
});

test("own proposal cannot be answered and offering tables uses version", async () => {
  await withHarness(snapshot(), async (h) => {
    h.setSnapshot(snapshot({ pending: { kind: "draw", by: "w" }, version: 6 }));
    h.click("draw-action"); await settle();
    assert.deepEqual(h.actions()[0].body, { room: "TEST123", action: "draw", version: 5 });
    assert.equal(h.$("pending-actions").hidden, true);
    assert.equal(h.$("draw-action").disabled, true);
    assert.match(h.$("pending-text").textContent, /Has propuesto/);
    h.click("accept-proposal"); assert.equal(h.actions().length, 1);
  });
});

test("HTTP 409 refreshes once, keeps the newer board, and never repeats the rejected move", async () => {
  await withHarness(snapshot(), async (h) => {
    h.on("/api/action", async () => ({ status: 409, payload: { error: "La partida ha cambiado." } }));
    h.setSnapshot(snapshot({ fen: AFTER_E5, history: OPENING, version: 7 }));
    const statesBefore = h.logs.filter((entry) => entry.endpoint === "/api/state").length;
    h.square("e2").click(); h.square("e4").click(); await settle();
    assert.equal(h.actions().length, 1);
    assert.equal(h.logs.filter((entry) => entry.endpoint === "/api/state").length, statesBefore + 1);
    assert.equal(h.square("e5").dataset.piece, "p");
    assert.match(h.$("notice-text").textContent, /partida ha cambiado/);
    assert.equal(h.pendingPolls(), 1);
  });
});

test("lost room membership returns to lobby with invitation ready for rejoining", async () => {
  await withHarness(snapshot(), async (h) => {
    h.on("/api/state", async () => ({ status: 403, payload: { error: "Vuelve a entrar en la sala." } }));
    await h.poll();
    assert.equal(h.$("game-view").hidden, true);
    assert.equal(h.$("lobby-view").hidden, false);
    assert.equal(h.$("join-form").hidden, false);
    assert.equal(h.$("room-code").value, "TEST123");
    assert.equal(h.$("join-submit").disabled, false);
  });
});

test("manually changing or removing the hash preserves active room membership even during a mutation", async () => {
  for (const hash of ["", "#room=OTHER123"]) {
    for (const busy of [false, true]) {
      await withHarness(snapshot(), async (h) => {
        let release;
        if (busy) {
          h.on("/api/action", () => new Promise((resolve) => { release = resolve; }));
          h.square("e2").click(); h.square("e4").click(); await settle();
        }
        const statesBefore = h.logs.filter((entry) => entry.endpoint === "/api/state").length;
        h.window.history.replaceState(null, "", `/${hash}`);
        h.window.dispatchEvent(new h.window.Event("hashchange")); await settle();
        assert.equal(h.window.location.hash, "#room=TEST123");
        assert.equal(h.$("game-view").hidden, false);
        assert.equal(h.$("game-room-code").textContent, "TEST123");
        assert.equal(h.logs.filter((entry) => entry.endpoint === "/api/state").length, statesBefore);
        assert.match(h.$("notice-text").textContent, /Salir de la sala/i);
        if (busy) { release(snapshot({ version: 6 })); await settle(); }
        assert.ok(!h.actions().some((entry) => entry.body.action === "leave"));
      });
    }
  }
});

test("HTTP invitation copying falls back to an exactly selected input", async () => {
  await withHarness(snapshot(), async (h) => {
    h.click("copy-invite"); await settle();
    const input = h.$("share-link");
    assert.equal(input.value, "http://nexo.invalid/#room=TEST123");
    assert.equal(h.document.activeElement, input);
    assert.equal(input.selectionStart, 0);
    assert.equal(input.selectionEnd, input.value.length);
    assert.equal(h.clipboardCalls, 0, "insecure HTTP must not attempt the secure clipboard API");
    assert.match(h.$("notice-text").textContent, /Enlace seleccionado.*Ctrl\+C/);
  });
});

test("join and create forms submit chosen identities and colors; player labels remain plain text", async () => {
  await withHarness(snapshot({ white: { name: "<img src=x onerror=alert(1)>", online: true } }), async (h) => {
    assert.equal(h.$("player-bottom").querySelector(".player-name").textContent, "<img src=x onerror=alert(1)>");
    assert.equal(h.$("player-bottom").querySelectorAll("img").length, 0);
  });
  await withHarness(snapshot({ you: "b" }), async (h) => {
    h.$("player-name").value = "Manu";
    h.$("room-name").value = "Nuestra partida";
    h.document.querySelector('input[name="create-color"][value="b"]').checked = true;
    h.$("create-form").dispatchEvent(new h.window.Event("submit", { bubbles: true, cancelable: true })); await settle();
    const create = h.logs.find((entry) => entry.endpoint === "/api/rooms");
    assert.deepEqual(create.body, { name: "Nuestra partida", playerName: "Manu", color: "b" });
    assert.equal(h.$("role-badge").textContent, "Juegas con negras");
  }, { lobby: true });
  await withHarness(snapshot({ you: "spectator" }), async (h) => {
    h.click("tab-join"); h.$("player-name").value = "Manu"; h.$("room-code").value = "test123";
    h.document.querySelector('input[name="join-color"][value="spectator"]').checked = true;
    h.$("join-form").dispatchEvent(new h.window.Event("submit", { bubbles: true, cancelable: true })); await settle();
    const join = h.logs.find((entry) => entry.endpoint === "/api/join");
    assert.deepEqual(join.body, { room: "TEST123", playerName: "Manu", color: "spectator" });
    assert.equal(h.$("role-badge").textContent, "Espectador");
  }, { lobby: true });
});

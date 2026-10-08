/* Original, self-contained chess silhouettes for Nexo Chess. */
(function () {
  "use strict";

  const silhouettes = {
    p: `
      <path d="M40 43h20l-3 10c-1 10 1 15 9 21H34c8-6 10-11 9-21Z"/>
      <circle cx="50" cy="29" r="13"/>
      <path d="M38 43h24" fill="none"/>
      <path d="M43 57h14" class="piece-detail"/>`,
    r: `
      <path d="M29 19h10v10h7V19h8v10h7V19h10v23H29Z"/>
      <path d="M35 42h30l-4 12v10l8 10H31l8-10V54Z"/>
      <path d="M35 42h30M39 56h22" class="piece-detail"/>`,
    n: `
      <path d="M32 74c6-8 10-14 11-24l-12 3-9-10 13-14 6-15 12 6 9 5c11 8 14 19 12 31l-4 18Z"/>
      <path d="m35 29 10-2M28 43l9 1M54 31c11 8 13 21 10 32" class="piece-detail"/>
      <circle cx="47" cy="34" r="2.7" class="piece-eye" stroke="none"/>
      <path d="m41 14 2 10" fill="none"/>`,
    b: `
      <path d="M50 13c-6 7-18 17-18 27 0 9 8 16 18 16s18-7 18-16c0-10-12-20-18-27Z"/>
      <path d="m56 24-12 16" class="piece-slit"/>
      <path d="M41 55h18l-2 8 10 11H33l10-11Z"/>
      <path d="M38 55h24M43 64h14" class="piece-detail"/>
      <circle cx="50" cy="12" r="3"/>`,
    q: `
      <path d="m29 29 11 9 10-15 10 15 11-9-7 24H36Z"/>
      <circle cx="28" cy="25" r="4.5"/>
      <circle cx="50" cy="19" r="4.5"/>
      <circle cx="72" cy="25" r="4.5"/>
      <path d="M37 53h26l-5 10 10 11H32l10-11Z"/>
      <path d="M36 47h28M38 54h24M42 64h16" class="piece-detail"/>`,
    k: `
      <path d="M46 10h8v8h8v8h-8v10h-8V26h-8v-8h8Z"/>
      <path d="M34 38c3-5 10-4 16 0 6-4 13-5 16 0 5 8-2 17-8 20H42c-6-3-13-12-8-20Z"/>
      <path d="M42 57h16l-2 7 12 10H32l12-10Z"/>
      <path d="M38 51h24M43 63h14" class="piece-detail"/>`
  };

  const pedestal = `
    <path d="M31 72h38l4 7H27Z"/>
    <path d="M27 79h46c3 0 5 2 5 5v3H22v-3c0-3 2-5 5-5Z"/>
    <path d="M29 83h42" class="piece-detail"/>`;

  function svg(pieceCode) {
    if (typeof pieceCode !== "string" || !/^[KQRBNPkqrbnp]$/.test(pieceCode)) {
      return "";
    }

    const white = pieceCode === pieceCode.toUpperCase();
    const fill = white ? "#f6f0df" : "#203834";
    const outline = white ? "#344d46" : "#c4cec0";
    const accent = white ? "#ad9d78" : "#90a397";
    const eye = white ? "#344d46" : "#e3e8d8";
    const shapes = (silhouettes[pieceCode.toLowerCase()] + pedestal)
      .replaceAll('class="piece-detail"', `fill="none" stroke="${accent}" stroke-width="2.5"`)
      .replaceAll('class="piece-slit"', `fill="none" stroke="${outline}" stroke-width="4"`)
      .replaceAll('class="piece-eye"', `fill="${eye}"`);

    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="100" height="100" aria-hidden="true" focusable="false" class="chess-piece piece-${white ? "white" : "black"}">
      <g fill="${fill}" stroke="${outline}" stroke-width="2.7" stroke-linecap="round" stroke-linejoin="round">
        ${shapes}
      </g>
    </svg>`;
  }

  window.NexoPieces = Object.freeze({ svg });
})();

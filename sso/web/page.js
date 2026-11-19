// Keeps the neat login session open while this page is, and tells neat when
// the page goes away so the waiting add_character call can return promptly.
(function () {
  "use strict";
  var toast = document.getElementById("toast");
  if (toast) {
    setTimeout(function () { toast.classList.add("hide"); }, 4000);
    toast.addEventListener("transitionend", function () { toast.remove(); });
  }

  var token = document.body.dataset.token;
  if (!token) return;

  function body() {
    return new URLSearchParams({ token: token });
  }

  function ended() {
    clearInterval(timer);
    document.getElementById("ended").hidden = false;
    document.querySelectorAll("button").forEach(function (b) { b.disabled = true; });
  }

  function ping() {
    fetch("/ping", { method: "POST", body: body() })
      .then(function (r) { if (!r.ok) ended(); })
      .catch(ended);
  }

  // Ping once right away: on a reload, the previous page's close beacon
  // reaches neat after this page was served (the old page unloads only
  // once the new one arrives) and cuts the session to a few seconds, so
  // waiting for the first interval would let it expire under this page.
  var timer = setInterval(ping, 10000);
  ping();

  window.addEventListener("pagehide", function (e) {
    // persisted means the page went into the back/forward cache rather
    // than away for good, so it may still come back.
    if (!e.persisted) navigator.sendBeacon("/closed", body());
  });
})();

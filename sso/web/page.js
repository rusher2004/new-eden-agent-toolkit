// Keeps the neat login session open while this page is, and tells neat when
// the page goes away so the waiting add_character call can return promptly.
(function () {
  "use strict";
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

  var timer = setInterval(function () {
    fetch("/ping", { method: "POST", body: body() })
      .then(function (r) { if (!r.ok) ended(); })
      .catch(ended);
  }, 10000);

  window.addEventListener("pagehide", function (e) {
    // persisted means the page went into the back/forward cache rather
    // than away for good, so it may still come back.
    if (!e.persisted) navigator.sendBeacon("/closed", body());
  });
})();

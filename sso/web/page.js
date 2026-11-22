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

  // Account controls: picking an account moves the character straight
  // away; "New account…" shows a name field and the Move button instead.
  document.querySelectorAll("form.move").forEach(function (form) {
    var select = form.querySelector("select");
    var name = form.querySelector("input[name=new_account]");
    var button = form.querySelector("button");
    name.hidden = true;
    button.hidden = true;
    select.addEventListener("change", function () {
      if (select.value === "new") {
        name.hidden = false;
        button.hidden = false;
        name.required = true;
        name.focus();
      } else {
        form.submit();
      }
    });
  });
  // Rename: the form is shown without JS; with it, the Rename button
  // toggles it.
  document.querySelectorAll(".account").forEach(function (account) {
    var toggle = account.querySelector("[data-rename]");
    var form = account.querySelector("form.rename");
    if (!toggle || !form) return;
    form.hidden = true;
    toggle.setAttribute("aria-expanded", "false");
    toggle.addEventListener("click", function () {
      form.hidden = !form.hidden;
      toggle.setAttribute("aria-expanded", String(!form.hidden));
      if (!form.hidden) form.querySelector("input[name=to]").select();
    });
  });
  document.querySelectorAll("form[data-confirm]").forEach(function (form) {
    form.addEventListener("submit", function (e) {
      if (!window.confirm(form.dataset.confirm)) e.preventDefault();
    });
  });

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

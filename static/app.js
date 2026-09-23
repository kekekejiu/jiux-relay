// 内置浏览器二次检测(前端兜底)
(function () {
  var ua = navigator.userAgent.toLowerCase();
  var inApp = /micromessenger|qq\/|qqbrowser|weibo|dingtalk|alipayclient|ucbrowser/.test(ua);
  if (inApp && location.pathname !== "/blocked") {
    location.href = "/blocked";
  }
})();

// 延迟检测:图片探测法,避免 CORS
function ping(host) {
  return new Promise(function (resolve) {
    var start = Date.now();
    var img = new Image();
    var done = false;
    var finish = function (ok) {
      if (done) return;
      done = true;
      resolve(ok ? Date.now() - start : -1);
    };
    img.onload = function () { finish(true); };
    img.onerror = function () { finish(true); }; // 即使 404,能连上也算通
    setTimeout(function () { finish(false); }, 6000);
    img.src = "https://" + host + "/favicon.ico?_=" + start;
  });
}

function pingClass(ms) {
  if (ms < 0) return "bad";
  if (ms <= 150) return "good";
  if (ms <= 600) return "mid";
  return "bad";
}
function pingText(ms) {
  return ms < 0 ? "超时" : ms + "ms";
}

// 加载入口列表并渲染
function loadLinks() {
  var box = document.getElementById("links-box");
  if (!box) return;
  fetch("/api/links")
    .then(function (r) { return r.json(); })
    .then(function (res) {
      if (!res.ok) { location.reload(); return; }
      box.innerHTML = "";
      if (!res.links || res.links.length === 0) {
        box.innerHTML = '<div class="info-row">暂无可用入口，请稍后再来</div>';
        return;
      }
      res.links.forEach(function (l) {
        var a = document.createElement("a");
        a.className = "link-item";
        a.href = "/go/" + l.id;
        a.innerHTML =
          '<div class="link-icon">&#128279;</div>' +
          '<div class="link-body">' +
            '<div class="link-title">' + escapeHtml(l.title) + "</div>" +
            (l.remark ? '<div class="link-remark">' + escapeHtml(l.remark) + "</div>" : "") +
          "</div>" +
          '<div class="ping" id="ping-' + l.id + '">检测中</div>';
        box.appendChild(a);

        ping(l.host).then(function (ms) {
          var el = document.getElementById("ping-" + l.id);
          if (el) {
            el.textContent = pingText(ms);
            el.className = "ping " + pingClass(ms);
          }
        });
      });
    })
    .catch(function () {
      box.innerHTML = '<div class="info-row">加载失败，请刷新重试</div>';
    });
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, function (c) {
    return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
  });
}

document.addEventListener("DOMContentLoaded", function () {
  loadLinks();
});

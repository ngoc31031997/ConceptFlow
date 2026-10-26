/**
 * Settings page: the Creator overrides any DeepSeek selector with one copied
 * from DevTools, checks what it finds on an open DeepSeek tab (the adapter
 * outlines the matches in red) and saves it to chrome.storage.sync, where the
 * adapter reads it at the start of every run.
 */
(function () {
  const DS = globalThis.DeepSeekSelectors;
  const form = document.getElementById("form");
  const status = document.getElementById("status");
  const result = document.getElementById("result");

  function say(text, tone) {
    status.textContent = text;
    status.style.color = tone === "bad" ? "var(--bad)" : tone === "ok" ? "var(--ok)" : "";
  }

  function render(values) {
    form.textContent = "";
    DS.FIELDS.forEach(function (f) {
      const label = document.createElement("label");
      label.htmlFor = "f-" + f.key;
      label.textContent = f.label;
      const hint = document.createElement("small");
      hint.textContent = f.hint;
      const input = document.createElement("input");
      input.id = "f-" + f.key;
      input.name = f.key;
      input.placeholder = f.value || "(để trống: tự tìm)";
      input.value = values[f.key] || "";
      const found = document.createElement("div");
      found.className = "found";
      found.id = "found-" + f.key;
      label.append(hint, input, found);
      form.append(label);
    });
  }

  function current() {
    const out = {};
    DS.FIELDS.forEach(function (f) {
      const v = form.elements[f.key].value.trim();
      if (v) out[f.key] = v;
    });
    return out;
  }

  function showProbe(probe) {
    DS.FIELDS.forEach(function (f) {
      const el = document.getElementById("found-" + f.key);
      el.textContent = "";
      el.className = "found";
    });
    Object.keys(probe).forEach(function (key) {
      const el = document.getElementById("found-" + key);
      const r = probe[key];
      if (!el || !r || key === "invalid") return;
      if (r.error) {
        el.textContent = "Lỗi: " + r.error;
        el.className = "found bad";
        return;
      }
      let text = r.count ? "Tìm thấy " + r.count : "Không tìm thấy";
      if (r.total !== undefined) text += " (trên trang có " + r.total + " khối markdown, kể cả phần suy nghĩ)";
      if (r.on !== undefined) text += " · đang " + (r.on ? "bật" : "tắt") + " · class=\"" + r.className + "\"";
      el.textContent = text;
      el.className = r.count ? "found ok" : "found bad";
    });
    result.hidden = false;
    result.textContent = JSON.stringify(probe, null, 2);
  }

  async function check() {
    const tabs = await chrome.tabs.query({ url: "https://chat.deepseek.com/*" });
    if (!tabs.length) {
      say("Chưa có tab DeepSeek nào. Mở chat.deepseek.com rồi bấm Kiểm tra lại.", "bad");
      return;
    }
    try {
      const probe = await chrome.tabs.sendMessage(tabs[0].id, { kind: "probe", selectors: current() });
      showProbe(probe);
      say("Đã kiểm tra trên tab DeepSeek: phần tìm thấy được viền đỏ, nút Copy dưới câu trả lời cuối viền xanh, trong vài giây. Ô khối câu trả lời và nút dưới câu trả lời chỉ kiểm được khi tab đang có một câu trả lời đã xong.", "ok");
    } catch (err) {
      say("Tab DeepSeek chưa nạp extension. Tải lại tab DeepSeek (Cmd+R) rồi bấm Kiểm tra lại.", "bad");
    }
  }

  async function save() {
    await chrome.storage.sync.set({ [DS.STORAGE_KEY]: current() });
    const S = DS.resolve(current());
    say(S.invalid.length ? "Đã lưu, nhưng ô " + S.invalid.join(", ") + " không hợp lệ nên dùng mặc định." : "Đã lưu. Lượt chạy sau dùng các selector này.", S.invalid.length ? "bad" : "ok");
  }

  async function reset() {
    await chrome.storage.sync.remove(DS.STORAGE_KEY);
    render({});
    say("Đã về mặc định.", "ok");
  }

  document.getElementById("check").addEventListener("click", function () {
    check().catch(function (err) {
      say("Không kiểm tra được: " + err.message, "bad");
    });
  });
  document.getElementById("save").addEventListener("click", function () {
    save().catch(function (err) {
      say("Không lưu được: " + err.message, "bad");
    });
  });
  document.getElementById("reset").addEventListener("click", function () {
    reset().catch(function (err) {
      say("Không xoá được: " + err.message, "bad");
    });
  });

  chrome.storage.sync.get(DS.STORAGE_KEY).then(
    function (stored) {
      render(stored[DS.STORAGE_KEY] || {});
    },
    function (err) {
      render({});
      say("Không đọc được selector đã lưu: " + err.message, "bad");
    },
  );
})();

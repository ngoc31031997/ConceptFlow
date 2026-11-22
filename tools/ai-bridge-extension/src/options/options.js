/**
 * Settings page: the Creator picks the default provider (chrome.storage.local),
 * and, per provider, overrides any selector with one copied from DevTools,
 * checks what it finds on an open tab of that provider (the adapter outlines
 * the matches in red) and saves it to chrome.storage.sync, where the adapter
 * reads it at the start of every run.
 */
(function () {
  const PR = globalThis.AiBridgeProviders;
  const SELECTORS = globalThis.AiBridgeSelectors;
  const form = document.getElementById("form");
  const status = document.getElementById("status");
  const result = document.getElementById("result");
  const providerSelect = document.getElementById("provider");
  const defaultSelect = document.getElementById("default-provider");

  function provider() {
    return PR.find(providerSelect.value);
  }

  function selectorsOf() {
    return SELECTORS[provider().id];
  }

  function say(text, tone) {
    status.textContent = text;
    status.style.color = tone === "bad" ? "var(--bad)" : tone === "ok" ? "var(--ok)" : "";
  }

  function fillProviders(select) {
    PR.LIST.forEach(function (p) {
      const option = document.createElement("option");
      option.value = p.id;
      option.textContent = p.name;
      select.append(option);
    });
  }

  function render(values) {
    form.textContent = "";
    selectorsOf().FIELDS.forEach(function (f) {
      const label = document.createElement("label");
      label.htmlFor = "f-" + f.key;
      label.textContent = f.label;
      const hint = document.createElement("small");
      hint.textContent = f.hint;
      const input = document.createElement("input");
      input.id = "f-" + f.key;
      input.name = f.key;
      input.placeholder = f.value || "(để trống)";
      input.value = values[f.key] || "";
      const found = document.createElement("div");
      found.className = "found";
      found.id = "found-" + f.key;
      label.append(hint, input, found);
      form.append(label);
    });
    result.hidden = true;
  }

  function current() {
    const out = {};
    selectorsOf().FIELDS.forEach(function (f) {
      const v = form.elements[f.key].value.trim();
      if (v) out[f.key] = v;
    });
    return out;
  }

  function showProbe(probe) {
    selectorsOf().FIELDS.forEach(function (f) {
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
      el.textContent = text;
      el.className = r.count ? "found ok" : "found bad";
    });
    result.hidden = false;
    result.textContent = JSON.stringify(probe, null, 2);
  }

  async function check() {
    const p = provider();
    const tabs = await chrome.tabs.query({ url: p.match });
    if (!tabs.length) {
      say("Chưa có tab " + p.name + " nào. Mở " + p.host + " rồi bấm Kiểm tra lại.", "bad");
      return;
    }
    try {
      const probe = await chrome.tabs.sendMessage(tabs[0].id, { kind: "probe", selectors: current() });
      showProbe(probe);
      say("Đã kiểm tra trên tab " + p.name + ": phần tìm thấy được viền đỏ, nút Copy dưới câu trả lời cuối viền xanh, trong vài giây. Các ô về câu trả lời chỉ kiểm được khi tab đang có một câu trả lời đã xong.", "ok");
    } catch (err) {
      say("Tab " + p.name + " chưa nạp extension. Tải lại tab " + p.name + " (Cmd+R) rồi bấm Kiểm tra lại.", "bad");
    }
  }

  async function save() {
    const K = selectorsOf();
    await chrome.storage.sync.set({ [K.STORAGE_KEY]: current() });
    const S = K.resolve(current(), document);
    say(S.invalid.length ? "Đã lưu, nhưng ô " + S.invalid.join(", ") + " không hợp lệ nên dùng mặc định." : "Đã lưu. Lượt chạy sau dùng các selector này.", S.invalid.length ? "bad" : "ok");
  }

  async function reset() {
    await chrome.storage.sync.remove(selectorsOf().STORAGE_KEY);
    render({});
    say("Đã về mặc định.", "ok");
  }

  async function showProvider() {
    const K = selectorsOf();
    try {
      const stored = await chrome.storage.sync.get(K.STORAGE_KEY);
      render(stored[K.STORAGE_KEY] || {});
    } catch (err) {
      render({});
      say("Không đọc được selector đã lưu: " + err.message, "bad");
    }
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
  providerSelect.addEventListener("change", function () {
    say("", "");
    void showProvider();
  });
  defaultSelect.addEventListener("change", function () {
    chrome.storage.local.set({ [PR.DEFAULT_KEY]: defaultSelect.value }).then(
      function () {
        say("Đã lưu nhà cung cấp mặc định: " + PR.nameOf(defaultSelect.value) + ".", "ok");
      },
      function (err) {
        say("Không lưu được nhà cung cấp mặc định: " + err.message, "bad");
      },
    );
  });

  fillProviders(providerSelect);
  fillProviders(defaultSelect);
  chrome.storage.local.get(PR.DEFAULT_KEY).then(
    function (stored) {
      defaultSelect.value = PR.readDefault(stored[PR.DEFAULT_KEY]);
    },
    function (err) {
      defaultSelect.value = PR.FALLBACK_ID;
      say("Không đọc được nhà cung cấp mặc định: " + err.message, "bad");
    },
  );
  void showProvider();
})();

/**
 * The toolbar popup. "Hàng đợi AI" lists the items kept in chrome.storage.local,
 * lets the Creator pick the provider of each item that has not run yet, and
 * sends the Creator's commands (pause, retry, drop, cancel, provider) to the
 * service worker, which owns the queue; "Cài đặt" opens the settings page.
 */
(function () {
  const Q = globalThis.AiBridgeQueue;
  const PR = globalThis.AiBridgeProviders;
  const QUEUE_KEY = "queue";

  /** The line for a running item's stage, naming the provider it runs on. */
  function stageText(stage, name) {
    const text = {
      preparing: "Đang lấy prompt",
      opening: "Đang mở " + name,
      sending: "Đang gửi prompt",
      answering: name + " đang trả lời",
      reading: "Đang đọc câu trả lời",
      saving: "Đang lưu vào ConceptFlow",
    };
    return text[stage] || stage;
  }
  const BADGES = { waiting: "Chờ", running: "Đang chạy", done: "Xong", failed: "Lỗi" };

  const list = document.getElementById("items");
  const empty = document.getElementById("empty");
  const summary = document.getElementById("summary");
  const toggle = document.getElementById("toggle");
  const clear = document.getElementById("clear");
  let state = Q.emptyState();
  let defaultId = PR.FALLBACK_ID;

  function send(message) {
    chrome.runtime.sendMessage(message).catch(function (err) {
      console.warn("ai-bridge popup: the extension did not take " + message.kind + ":", err);
    });
  }

  function button(text, onClick) {
    const b = document.createElement("button");
    b.type = "button";
    b.textContent = text;
    b.addEventListener("click", onClick);
    return b;
  }

  function line(className, text) {
    const el = document.createElement("div");
    el.className = className;
    el.textContent = text;
    return el;
  }

  /**
   * The provider line of an item: a select while it can still change (waiting or failed),
   * else the provider it ran on (running or done).
   */
  function providerRow(item) {
    const row = document.createElement("div");
    row.className = "provider";
    if (item.status === "running" || item.status === "done") {
      row.textContent = (item.status === "running" ? "Đang chạy trên " : "Đã chạy trên ") + PR.nameOf(item.ranOn);
      return row;
    }
    const label = document.createElement("label");
    label.textContent = "Chạy trên ";
    const select = document.createElement("select");
    select.setAttribute("aria-label", "Nhà cung cấp AI của mục " + item.label);
    const standard = document.createElement("option");
    standard.value = "";
    standard.textContent = "Mặc định (" + PR.nameOf(defaultId) + ")";
    select.append(standard);
    PR.LIST.forEach(function (p) {
      const option = document.createElement("option");
      option.value = p.id;
      option.textContent = p.name;
      select.append(option);
    });
    select.value = item.provider || "";
    select.addEventListener("change", function () { send({ kind: "queue_set_provider", id: item.id, provider: select.value }); });
    label.append(select);
    row.append(label);
    if (item.status === "failed" && item.ranOn) row.append(document.createTextNode(" · lần chạy trước: " + PR.nameOf(item.ranOn)));
    return row;
  }

  function renderItem(item) {
    const li = document.createElement("li");
    li.dataset.status = item.status;
    const top = document.createElement("div");
    top.className = "top";
    const badge = document.createElement("span");
    badge.className = "badge " + item.status;
    badge.textContent = BADGES[item.status];
    const label = document.createElement("span");
    label.className = "label";
    label.textContent = item.label;
    top.append(badge, label);
    li.append(top);
    if (item.project) li.append(line("project", item.project));
    li.append(providerRow(item));
    if (item.status === "running" && item.stage) li.append(line("stage", stageText(item.stage, PR.nameOf(item.ranOn))));
    if (item.status === "failed") {
      li.append(line("error", item.error));
      if (item.reply) {
        const details = document.createElement("details");
        const sum = document.createElement("summary");
        sum.textContent = "Câu trả lời của " + PR.nameOf(item.ranOn || Q.providerOf(item, defaultId));
        const pre = document.createElement("pre");
        pre.textContent = item.reply;
        details.append(sum, pre);
        li.append(details);
      }
    }
    const row = document.createElement("div");
    row.className = "row";
    if (item.status === "failed") row.append(button("Thử lại", function () { send({ kind: "queue_retry", id: item.id }); }));
    if (item.status === "running") row.append(button("Huỷ", function () { send({ kind: "queue_cancel", id: item.id }); }));
    else row.append(button("Bỏ", function () { send({ kind: "queue_drop", id: item.id }); }));
    li.append(row);
    return li;
  }

  function render() {
    list.textContent = "";
    state.items.forEach(function (item) { list.append(renderItem(item)); });
    empty.hidden = state.items.length > 0;
    const n = Q.counts(state);
    const parts = [];
    if (n.running) parts.push(n.running + " đang chạy");
    if (n.waiting) parts.push(n.waiting + " chờ");
    if (n.failed) parts.push(n.failed + " lỗi");
    if (n.done) parts.push(n.done + " xong");
    summary.textContent = parts.join(" · ");
    toggle.textContent = state.paused ? "Chạy tiếp" : "Tạm dừng";
    clear.disabled = state.items.every(function (it) { return it.status === "running"; });
  }

  toggle.addEventListener("click", function () { send({ kind: state.paused ? "queue_resume" : "queue_pause" }); });
  clear.addEventListener("click", function () { send({ kind: "queue_clear" }); });
  document.getElementById("open-options").addEventListener("click", function () { chrome.runtime.openOptionsPage(); });

  const tabs = ["queue", "settings"];
  tabs.forEach(function (name) {
    document.getElementById("tab-" + name).addEventListener("click", function () {
      tabs.forEach(function (other) {
        document.getElementById("tab-" + other).setAttribute("aria-selected", String(other === name));
        document.getElementById("panel-" + other).hidden = other !== name;
      });
    });
  });

  chrome.storage.onChanged.addListener(function (changes, area) {
    if (area !== "local") return;
    if (changes[QUEUE_KEY]) state = Q.readState(changes[QUEUE_KEY].newValue);
    if (changes[PR.DEFAULT_KEY]) defaultId = PR.readDefault(changes[PR.DEFAULT_KEY].newValue);
    if (changes[QUEUE_KEY] || changes[PR.DEFAULT_KEY]) render();
  });
  chrome.storage.local.get([QUEUE_KEY, PR.DEFAULT_KEY]).then(function (stored) {
    state = Q.readState(stored[QUEUE_KEY]);
    defaultId = PR.readDefault(stored[PR.DEFAULT_KEY]);
    render();
  });
})();

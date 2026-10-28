/**
 * The toolbar popup. "Hàng đợi" lists the items kept in chrome.storage.local and
 * sends the Creator's commands (pause, retry, drop, cancel) to the service
 * worker, which owns the queue; "Cài đặt" opens the selector settings page.
 */
(function () {
  const Q = globalThis.AiBridgeQueue;
  const QUEUE_KEY = "queue";

  const STAGES = {
    preparing: "Đang lấy prompt",
    opening: "Đang mở DeepSeek",
    sending: "Đang gửi prompt",
    answering: "DeepSeek đang trả lời",
    reading: "Đang đọc câu trả lời",
    saving: "Đang lưu vào ConceptFlow",
  };
  const BADGES = { waiting: "Chờ", running: "Đang chạy", done: "Xong", failed: "Lỗi" };

  const list = document.getElementById("items");
  const empty = document.getElementById("empty");
  const summary = document.getElementById("summary");
  const toggle = document.getElementById("toggle");
  const clear = document.getElementById("clear");
  let state = Q.emptyState();

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
    if (item.status === "running" && item.stage) li.append(line("stage", STAGES[item.stage] || item.stage));
    if (item.status === "failed") {
      li.append(line("error", item.error));
      if (item.reply) {
        const details = document.createElement("details");
        const sum = document.createElement("summary");
        sum.textContent = "Câu trả lời của DeepSeek";
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
    if (area === "local" && changes[QUEUE_KEY]) {
      state = Q.readState(changes[QUEUE_KEY].newValue);
      render();
    }
  });
  chrome.storage.local.get(QUEUE_KEY).then(function (stored) {
    state = Q.readState(stored[QUEUE_KEY]);
    render();
  });
})();

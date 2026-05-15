const defaultFiles = [
  "/Users/hgkim/Downloads/2026년 1차 희망하우징(공공기숙사) 잔여세대 입주자 모집공고문.pdf",
  "/Users/hgkim/Downloads/제50차 장기전세 입주자 모집 공고문.pdf",
  "/Users/hgkim/Downloads/제7차 장기전세주택2(미리내집) 공고문.pdf",
];

const state = {
  report: null,
  query: "",
};

const fileInput = document.getElementById("fileInput");
const refreshButton = document.getElementById("refreshButton");
const apiStatus = document.getElementById("apiStatus");
const stats = document.getElementById("stats");
const searchInput = document.getElementById("searchInput");
const offeringsBody = document.getElementById("offeringsBody");

fileInput.value = defaultFiles.join("\n");

refreshButton.addEventListener("click", loadReport);
searchInput.addEventListener("input", () => {
  state.query = searchInput.value.trim().toLowerCase();
  renderOfferings();
});

loadReport();

async function loadReport() {
  const files = fileInput.value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);

  if (files.length === 0) {
    setStatus("PDF 경로를 입력하세요.", true);
    return;
  }

  const params = new URLSearchParams();
  for (const file of files) {
    params.append("file", file);
  }

  setStatus("API 조회 중...");
  refreshButton.disabled = true;
  try {
    const response = await fetch(`/reports/pdf-offerings?${params.toString()}`, {
      headers: { Accept: "application/json" },
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    state.report = await response.json();
    setStatus(`조회 완료: ${state.report.generated_at}`);
    renderStats();
    renderOfferings();
  } catch (error) {
    state.report = null;
    stats.innerHTML = "";
    offeringsBody.innerHTML = `<tr><td class="error" colspan="12">조회 실패: ${escapeHTML(error.message)}</td></tr>`;
    setStatus("조회 실패", true);
  } finally {
    refreshButton.disabled = false;
  }
}

function renderStats() {
  const totals = state.report?.totals ?? { files: 0, artifacts: 0, offerings: 0 };
  stats.innerHTML = [
    ["PDF", totals.files],
    ["Artifacts", totals.artifacts],
    ["Offerings", totals.offerings],
    ["API", "/reports/pdf-offerings"],
  ]
    .map(([label, value]) => `<div class="stat"><span>${escapeHTML(label)}</span><strong>${escapeHTML(String(value))}</strong></div>`)
    .join("");
}

function renderOfferings() {
  const rows = (state.report?.offerings ?? []).filter((row) => {
    if (!state.query) {
      return true;
    }
    return [
      row.file,
      row.application_unit_label,
      row.housing_name,
      row.gender_requirement,
      row.source_span,
    ]
      .join(" ")
      .toLowerCase()
      .includes(state.query);
  });

  if (rows.length === 0) {
    offeringsBody.innerHTML = '<tr><td class="empty" colspan="12">조회된 공급항목이 없습니다.</td></tr>';
    return;
  }

  offeringsBody.innerHTML = rows
    .map((row, index) => `<tr>
      <td class="num">${index + 1}</td>
      <td>${escapeHTML(shortFileName(row.file))}</td>
      <td>${escapeHTML(row.application_unit_label)}</td>
      <td>${escapeHTML(row.housing_name || row.complex_name)}</td>
      <td class="num">${formatValue(row.exclusive_area_m2)}</td>
      <td class="num">${formatValue(row.supply_count)}</td>
      <td class="num">${formatMoney(row.jeonse_deposit_krw)}</td>
      <td class="num">${formatMoney(row.deposit_krw)}</td>
      <td class="num">${formatMoney(row.monthly_rent_krw)}</td>
      <td class="num">${formatMoney(row.dormitory_fee_krw)}</td>
      <td>${escapeHTML(row.gender_requirement)}</td>
      <td class="num">${formatValue(row.confidence)}</td>
    </tr>`)
    .join("");
}

function setStatus(message, failed = false) {
  apiStatus.textContent = message;
  apiStatus.style.color = failed ? "var(--danger)" : "var(--muted)";
}

function shortFileName(path) {
  return String(path || "").split("/").pop();
}

function formatMoney(value) {
  if (value === null || value === undefined || value === "") {
    return "";
  }
  return new Intl.NumberFormat("ko-KR").format(value);
}

function formatValue(value) {
  if (value === null || value === undefined || value === "") {
    return "";
  }
  return String(value);
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

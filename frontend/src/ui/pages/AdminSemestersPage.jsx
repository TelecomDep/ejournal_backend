import React, { useCallback, useEffect, useState } from "react";
import api from "../../services/api";

const STATUS_LABELS = {
  open: { label: "Открыт (текущий)", className: "status-open", color: "#10b981" },
  planned: { label: "Запланирован", className: "status-planned", color: "#3b82f6" },
  closed: { label: "Закрыт", className: "status-closed", color: "#6b7280" },
  archived: { label: "В архиве", className: "status-archived", color: "#9ca3af" }
};

const formatDate = (val) => {
  if (!val) return "—";
  const d = new Date(val);
  if (Number.isNaN(d.getTime())) return "—";
  return new Intl.DateTimeFormat("ru-RU", { day: "2-digit", month: "2-digit", year: "numeric" }).format(d);
};

const toInputDate = (val) => {
  if (!val) return "";
  const d = new Date(val);
  if (Number.isNaN(d.getTime())) return "";
  return d.toISOString().slice(0, 10);
};

const AdminSemestersPage = ({ token }) => {
  const [semesters, setSemesters] = useState([]);
  const [currentSemester, setCurrentSemester] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [actionLoading, setActionLoading] = useState(null);

  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [form, setForm] = useState({
    academic_year: "2026/2027",
    term_num: 1,
    name: "2026/2027, осенний семестр",
    starts_at: "2026-09-01",
    ends_at: "2027-01-31",
    status: "planned"
  });

  const loadData = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const listData = await api.getSemesters(token);
      const list = Array.isArray(listData?.items) ? listData.items : (Array.isArray(listData?.semesters) ? listData.semesters : (Array.isArray(listData) ? listData : []));
      setSemesters(list);

      try {
        const curr = await api.getCurrentSemester(token);
        setCurrentSemester(curr?.semester || curr || null);
      } catch {
        setCurrentSemester(list.find((s) => s.status === "open" || s.is_current) || null);
      }
    } catch (err) {
      setError(api.getErrorMessage(err, "Ошибка загрузки семестров"));
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleActivate = async (semester) => {
    if (!window.confirm(`Открыть семестр "${semester.name}"? Это сделает его текущим активным семестром.`)) return;
    setActionLoading(semester.semester_id);
    setError("");
    setSuccess("");
    try {
      await api.activateAdminSemester(token, semester.semester_id);
      setSuccess(`Семестр "${semester.name}" успешно открыт и назначен текущим!`);
      await loadData();
    } catch (err) {
      setError(api.getErrorMessage(err, "Не удалось открыть семестр"));
    } finally {
      setActionLoading(null);
    }
  };

  const handleClose = async (semester) => {
    if (!window.confirm(`Закрыть семестр "${semester.name}"? Редактирование оценок и посещаемости будет ограничено.`)) return;
    setActionLoading(semester.semester_id);
    setError("");
    setSuccess("");
    try {
      await api.closeAdminSemester(token, semester.semester_id);
      setSuccess(`Семестр "${semester.name}" успешно закрыт!`);
      await loadData();
    } catch (err) {
      setError(api.getErrorMessage(err, "Не удалось закрыть семестр"));
    } finally {
      setActionLoading(null);
    }
  };

  const handleArchive = async (semester) => {
    if (!window.confirm(`Перенести семестр "${semester.name}" в архив?`)) return;
    setActionLoading(semester.semester_id);
    setError("");
    setSuccess("");
    try {
      await api.archiveAdminSemester(token, semester.semester_id);
      setSuccess(`Семестр "${semester.name}" перенесен в архив.`);
      await loadData();
    } catch (err) {
      setError(api.getErrorMessage(err, "Не удалось архивировать семестр"));
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (semester) => {
    if (!window.confirm(`Удалить семестр "${semester.name}"?`)) return;
    setActionLoading(semester.semester_id);
    setError("");
    setSuccess("");
    try {
      await api.deleteAdminSemester(token, semester.semester_id);
      setSuccess(`Семестр "${semester.name}" удален.`);
      await loadData();
    } catch (err) {
      setError(api.getErrorMessage(err, "Не удалось удалить семестр"));
    } finally {
      setActionLoading(null);
    }
  };

  const handleCreateSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setSuccess("");
    try {
      const payload = {
        academic_year: form.academic_year.trim(),
        term_num: Number(form.term_num),
        name: form.name.trim(),
        starts_at: new Date(form.starts_at + "T00:00:00Z").toISOString(),
        ends_at: new Date(form.ends_at + "T23:59:59Z").toISOString(),
        status: form.status
      };
      await api.createAdminSemester(token, payload);
      setSuccess(`Семестр "${payload.name}" успешно создан!`);
      setIsCreateOpen(false);
      await loadData();
    } catch (err) {
      setError(api.getErrorMessage(err, "Ошибка создания семестра"));
    }
  };

  const openSemester = semesters.find((s) => s.status === "open" || s.is_current);

  return (
    <section className="ui-page">
      <div className="ui-page-header">
        <div>
          <p className="ui-kicker">Панель администратора</p>
          <h1>Управление семестрами</h1>
          <p>Открытие, закрытие, архивация и создание учебных периодов (семестров).</p>
        </div>
        <div className="ui-page-actions" style={{ display: "flex", gap: "10px" }}>
          <button
            type="button"
            className="ui-btn ui-btn-primary"
            onClick={() => setIsCreateOpen(true)}
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: "6px",
              padding: "10px 16px",
              backgroundColor: "#2563eb",
              color: "#fff",
              borderRadius: "8px",
              fontWeight: "600",
              border: "none",
              cursor: "pointer"
            }}
          >
            + Создать семестр
          </button>
          <button
            type="button"
            className="ui-btn"
            onClick={loadData}
            title="Обновить"
            style={{
              padding: "10px 14px",
              backgroundColor: "#f3f4f6",
              borderRadius: "8px",
              border: "1px solid #d1d5db",
              cursor: "pointer"
            }}
          >
            ↻ Обновить
          </button>
        </div>
      </div>

      {success && (
        <div style={{
          backgroundColor: "#ecfdf5",
          color: "#065f46",
          border: "1px solid #a7f3d0",
          borderRadius: "8px",
          padding: "12px 16px",
          marginBottom: "16px",
          fontWeight: "500"
        }}>
          ✓ {success}
        </div>
      )}

      {error && (
        <div style={{
          backgroundColor: "#fef2f2",
          color: "#991b1b",
          border: "1px solid #fecaca",
          borderRadius: "8px",
          padding: "12px 16px",
          marginBottom: "16px",
          fontWeight: "500"
        }}>
          ✕ {error}
        </div>
      )}

      {/* Banner Current Active Semester */}
      <div style={{
        backgroundColor: openSemester ? "#f0fdf4" : "#fefce8",
        border: openSemester ? "1px solid #86efac" : "1px solid #fde047",
        borderRadius: "12px",
        padding: "20px",
        marginBottom: "24px",
        display: "flex",
        justifyContent: "space-between",
        alignItems: "center",
        flexWrap: "wrap",
        gap: "16px"
      }}>
        <div>
          <div style={{ display: "flex", alignItems: "center", gap: "10px", marginBottom: "6px" }}>
            <span style={{ fontSize: "1.2rem" }}>{openSemester ? "🟢" : "⏸️"}</span>
            <strong style={{ fontSize: "1.15rem", color: openSemester ? "#166534" : "#854d0e" }}>
              {openSemester ? `Текущий активный семестр: ${openSemester.name}` : "Нет открытого семестра (межсезонье)"}
            </strong>
          </div>
          <p style={{ margin: 0, color: "#4b5563", fontSize: "0.95rem" }}>
            {openSemester
              ? `Учебный год: ${openSemester.academic_year} | Семестр: ${openSemester.term_num} | Период: ${formatDate(openSemester.starts_at)} — ${formatDate(openSemester.ends_at)}`
              : "Когда нет открытого семестра, мобильное приложение и система работают в межсезонном режиме. Вы можете открыть нужный семестр ниже."}
          </p>
        </div>

        {openSemester && (
          <button
            type="button"
            onClick={() => handleClose(openSemester)}
            disabled={actionLoading === openSemester.semester_id}
            style={{
              padding: "10px 18px",
              backgroundColor: "#ea580c",
              color: "#fff",
              borderRadius: "8px",
              border: "none",
              fontWeight: "600",
              cursor: "pointer"
            }}
          >
            {actionLoading === openSemester.semester_id ? "Закрытие..." : "Закрыть семестр"}
          </button>
        )}
      </div>

      {/* Semesters Table */}
      <div style={{ backgroundColor: "#fff", borderRadius: "12px", border: "1px solid #e5e7eb", overflow: "hidden" }}>
        <table style={{ width: "100%", borderCollapse: "collapse", textAlign: "left" }}>
          <thead>
            <tr style={{ backgroundColor: "#f9fafb", borderBottom: "1px solid #e5e7eb", color: "#6b7280", fontSize: "0.85rem", textTransform: "uppercase" }}>
              <th style={{ padding: "14px 16px" }}>ID</th>
              <th style={{ padding: "14px 16px" }}>Название семестра</th>
              <th style={{ padding: "14px 16px" }}>Учебный год</th>
              <th style={{ padding: "14px 16px" }}>Сроки проведения</th>
              <th style={{ padding: "14px 16px" }}>Статус</th>
              <th style={{ padding: "14px 16px", textAlign: "right" }}>Действия</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={6} style={{ padding: "32px", textAlign: "center", color: "#6b7280" }}>
                  Загрузка семестров...
                </td>
              </tr>
            ) : semesters.length === 0 ? (
              <tr>
                <td colSpan={6} style={{ padding: "32px", textAlign: "center", color: "#6b7280" }}>
                  Семестры не найдены. Нажмите «+ Создать семестр».
                </td>
              </tr>
            ) : (
              semesters.map((sem) => {
                const isOp = sem.status === "open" || sem.is_current;
                const statusMeta = STATUS_LABELS[sem.status] || { label: sem.status, color: "#6b7280" };
                const isLoadingRow = actionLoading === sem.semester_id;

                return (
                  <tr key={sem.semester_id} style={{ borderBottom: "1px solid #f3f4f6", backgroundColor: isOp ? "#f0fdf4" : "transparent" }}>
                    <td style={{ padding: "14px 16px", fontWeight: "bold", color: "#6b7280" }}>#{sem.semester_id}</td>
                    <td style={{ padding: "14px 16px" }}>
                      <strong style={{ color: "#111827", fontSize: "1rem" }}>{sem.name}</strong>
                    </td>
                    <td style={{ padding: "14px 16px", color: "#374151" }}>
                      {sem.academic_year} (№{sem.term_num})
                    </td>
                    <td style={{ padding: "14px 16px", color: "#4b5563", fontSize: "0.9rem" }}>
                      {formatDate(sem.starts_at)} — {formatDate(sem.ends_at)}
                    </td>
                    <td style={{ padding: "14px 16px" }}>
                      <span style={{
                        display: "inline-block",
                        padding: "4px 10px",
                        borderRadius: "20px",
                        fontSize: "0.82rem",
                        fontWeight: "600",
                        backgroundColor: isOp ? "#dcfce7" : sem.status === "planned" ? "#dbeafe" : "#f3f4f6",
                        color: statusMeta.color
                      }}>
                        {statusMeta.label}
                      </span>
                    </td>
                    <td style={{ padding: "14px 16px", textAlign: "right" }}>
                      <div style={{ display: "inline-flex", gap: "8px", justifyContent: "flex-end" }}>
                        {!isOp && sem.status !== "archived" && (
                          <button
                            type="button"
                            onClick={() => handleActivate(sem)}
                            disabled={isLoadingRow}
                            style={{
                              padding: "6px 12px",
                              backgroundColor: "#10b981",
                              color: "#fff",
                              border: "none",
                              borderRadius: "6px",
                              fontWeight: "600",
                              fontSize: "0.85rem",
                              cursor: "pointer"
                            }}
                          >
                            {isLoadingRow ? "..." : "Открыть"}
                          </button>
                        )}

                        {isOp && (
                          <button
                            type="button"
                            onClick={() => handleClose(sem)}
                            disabled={isLoadingRow}
                            style={{
                              padding: "6px 12px",
                              backgroundColor: "#f97316",
                              color: "#fff",
                              border: "none",
                              borderRadius: "6px",
                              fontWeight: "600",
                              fontSize: "0.85rem",
                              cursor: "pointer"
                            }}
                          >
                            {isLoadingRow ? "..." : "Закрыть"}
                          </button>
                        )}

                        {sem.status === "closed" && (
                          <button
                            type="button"
                            onClick={() => handleArchive(sem)}
                            disabled={isLoadingRow}
                            style={{
                              padding: "6px 12px",
                              backgroundColor: "#6b7280",
                              color: "#fff",
                              border: "none",
                              borderRadius: "6px",
                              fontSize: "0.85rem",
                              cursor: "pointer"
                            }}
                          >
                            В архив
                          </button>
                        )}

                        {!isOp && (
                          <button
                            type="button"
                            onClick={() => handleDelete(sem)}
                            disabled={isLoadingRow}
                            style={{
                              padding: "6px 10px",
                              backgroundColor: "transparent",
                              color: "#ef4444",
                              border: "1px solid #fca5a5",
                              borderRadius: "6px",
                              fontSize: "0.85rem",
                              cursor: "pointer"
                            }}
                          >
                            Удалить
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Modal Create Semester */}
      {isCreateOpen && (
        <div style={{
          position: "fixed",
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: "rgba(0,0,0,0.5)",
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          zIndex: 1000
        }}>
          <div style={{
            backgroundColor: "#fff",
            borderRadius: "16px",
            padding: "28px",
            width: "90%",
            maxWidth: "520px",
            boxShadow: "0 20px 25px -5px rgba(0,0,0,0.1)"
          }}>
            <h2 style={{ margin: "0 0 16px 0", fontSize: "1.3rem" }}>Создать новый семестр</h2>
            <form onSubmit={handleCreateSubmit}>
              <div style={{ marginBottom: "14px" }}>
                <label style={{ display: "block", fontSize: "0.88rem", fontWeight: "600", marginBottom: "4px" }}>Учебный год (формат YYYY/YYYY):</label>
                <input
                  type="text"
                  required
                  placeholder="2026/2027"
                  value={form.academic_year}
                  onChange={(e) => setForm({ ...form, academic_year: e.target.value })}
                  style={{ width: "100%", padding: "8px 12px", borderRadius: "6px", border: "1px solid #d1d5db" }}
                />
              </div>

              <div style={{ display: "flex", gap: "12px", marginBottom: "14px" }}>
                <div style={{ flex: 1 }}>
                  <label style={{ display: "block", fontSize: "0.88rem", fontWeight: "600", marginBottom: "4px" }}>Номер семестра:</label>
                  <select
                    value={form.term_num}
                    onChange={(e) => {
                      const num = Number(e.target.value);
                      const termName = num === 1 ? "осенний" : "весенний";
                      setForm({
                        ...form,
                        term_num: num,
                        name: `${form.academic_year}, ${termName} семестр`
                      });
                    }}
                    style={{ width: "100%", padding: "8px 12px", borderRadius: "6px", border: "1px solid #d1d5db" }}
                  >
                    <option value={1}>1 (Осенний)</option>
                    <option value={2}>2 (Весенний)</option>
                  </select>
                </div>

                <div style={{ flex: 1 }}>
                  <label style={{ display: "block", fontSize: "0.88rem", fontWeight: "600", marginBottom: "4px" }}>Статус:</label>
                  <select
                    value={form.status}
                    onChange={(e) => setForm({ ...form, status: e.target.value })}
                    style={{ width: "100%", padding: "8px 12px", borderRadius: "6px", border: "1px solid #d1d5db" }}
                  >
                    <option value="planned">Запланирован</option>
                    <option value="open">Открыть сразу</option>
                  </select>
                </div>
              </div>

              <div style={{ marginBottom: "14px" }}>
                <label style={{ display: "block", fontSize: "0.88rem", fontWeight: "600", marginBottom: "4px" }}>Название семестра:</label>
                <input
                  type="text"
                  required
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  style={{ width: "100%", padding: "8px 12px", borderRadius: "6px", border: "1px solid #d1d5db" }}
                />
              </div>

              <div style={{ display: "flex", gap: "12px", marginBottom: "20px" }}>
                <div style={{ flex: 1 }}>
                  <label style={{ display: "block", fontSize: "0.88rem", fontWeight: "600", marginBottom: "4px" }}>Дата начала:</label>
                  <input
                    type="date"
                    required
                    value={form.starts_at}
                    onChange={(e) => setForm({ ...form, starts_at: e.target.value })}
                    style={{ width: "100%", padding: "8px 12px", borderRadius: "6px", border: "1px solid #d1d5db" }}
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <label style={{ display: "block", fontSize: "0.88rem", fontWeight: "600", marginBottom: "4px" }}>Дата окончания:</label>
                  <input
                    type="date"
                    required
                    value={form.ends_at}
                    onChange={(e) => setForm({ ...form, ends_at: e.target.value })}
                    style={{ width: "100%", padding: "8px 12px", borderRadius: "6px", border: "1px solid #d1d5db" }}
                  />
                </div>
              </div>

              <div style={{ display: "flex", justifyContent: "flex-end", gap: "10px" }}>
                <button
                  type="button"
                  onClick={() => setIsCreateOpen(false)}
                  style={{ padding: "8px 16px", borderRadius: "6px", border: "1px solid #d1d5db", backgroundColor: "#f3f4f6", cursor: "pointer" }}
                >
                  Отмена
                </button>
                <button
                  type="submit"
                  style={{ padding: "8px 18px", borderRadius: "6px", border: "none", backgroundColor: "#2563eb", color: "#fff", fontWeight: "600", cursor: "pointer" }}
                >
                  Создать семестр
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </section>
  );
};

export default AdminSemestersPage;

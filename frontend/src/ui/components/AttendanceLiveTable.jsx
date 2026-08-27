import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";
import api from "../../services/api";

const POLL_INTERVAL_MS = 2000;

const STATUS_OPTIONS = [
  { value: "present", label: "Присутствует" },
  { value: "late", label: "Опоздал" },
  { value: "excused", label: "Уважительная причина" },
  { value: "absent", label: "Не отмечен" }
];

const formatTime = (value) => {
  if (!value) return "Нет отметки";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Нет отметки";
  return date.toLocaleTimeString("ru-RU", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit"
  });
};

const sourceLabel = (value, isFraud) => {
  if (isFraud) return "Антифрод: заблокировано";
  if (value === "self") return "Студент";
  if (value === "teacher") return "Преподаватель";
  return "Нет отметки";
};

const fraudReasonLabel = (reason) => {
  if (reason === "student is too far from lesson location") return "Вне зоны занятия";
  if (
    reason === "device_id already used in this lesson" ||
    reason === "Повторная отметка с устройства другого студента" ||
    reason === "duplicate_device"
  ) {
    return "Повторное устройство";
  }
  return reason || "Нарушение антифрода";
};

const updateSnapshotStudent = (snapshot, studentId, status) => {
  if (!snapshot) return snapshot;

  const students = (snapshot.students || []).map((student) => (
    Number(student.student_id) === Number(studentId)
      ? {
        ...student,
        status,
        marked_at: new Date().toISOString(),
        marked_by: "teacher"
      }
      : student
  ));
  const markedCount = students.filter((student) => student.status !== "absent" && !student.is_fraud).length;
  const rosterSize = students.length;

  return {
    ...snapshot,
    students,
    marked_count: markedCount,
    roster_size: rosterSize,
    attendance_percent: rosterSize ? (markedCount * 100) / rosterSize : 0
  };
};

const AttendanceLiveTable = ({ token, session }) => {
  const lessonId = Number(session?.lesson_id || session?.id || 0);
  const [snapshot, setSnapshot] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [updatingStudentId, setUpdatingStudentId] = useState(0);

  const mountedRef = useRef(true);
  const requestInFlightRef = useRef(false);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const loadRoster = useCallback(async (silent = false) => {
    if (!lessonId || requestInFlightRef.current) return;

    requestInFlightRef.current = true;
    if (!silent && mountedRef.current) setLoading(true);
    try {
      const payload = await api.getAttendanceSessionRoster(token, lessonId);
      if (!mountedRef.current) return;
      setSnapshot(payload);
      setError("");
    } catch (err) {
      if (mountedRef.current) {
        setError(api.getErrorMessage(err, "Не удалось обновить список посещаемости"));
      }
    } finally {
      requestInFlightRef.current = false;
      if (mountedRef.current) setLoading(false);
    }
  }, [lessonId, token]);

  useEffect(() => {
    setSnapshot(null);
    setLoading(true);
    setError("");
    loadRoster(false);

    const intervalId = window.setInterval(() => loadRoster(true), POLL_INTERVAL_MS);
    const refreshWhenVisible = () => {
      if (document.visibilityState === "visible") loadRoster(true);
    };
    document.addEventListener("visibilitychange", refreshWhenVisible);

    return () => {
      window.clearInterval(intervalId);
      document.removeEventListener("visibilitychange", refreshWhenVisible);
    };
  }, [loadRoster]);

  const handleStatusChange = useCallback(async (studentId, status) => {
    const student = (snapshot?.students || []).find((s) => Number(s.student_id) === Number(studentId));
    if (student?.is_fraud) {
      setError("Нельзя изменить статус: у студента зафиксирована попытка мошенничества (антифрод)");
      return;
    }
    setUpdatingStudentId(Number(studentId));
    setError("");
    setSnapshot((current) => updateSnapshotStudent(current, studentId, status));

    try {
      await api.updateAttendanceStatus(token, lessonId, studentId, status);
      await loadRoster(true);
    } catch (err) {
      if (mountedRef.current) {
        setError(api.getErrorMessage(err, "Не удалось сохранить новый статус"));
        await loadRoster(true);
      }
    } finally {
      if (mountedRef.current) setUpdatingStudentId(0);
    }
  }, [lessonId, loadRoster, snapshot?.students, token]);

  const visibleStudents = useMemo(() => {
    const query = search.trim().toLocaleLowerCase("ru-RU");
    return (snapshot?.students || []).filter((student) => {
      if (statusFilter === "marked" && (student.status === "absent" || student.is_fraud)) return false;
      if (statusFilter === "unmarked" && student.status !== "absent" && !student.is_fraud) return false;
      if (statusFilter === "fraud" && !student.is_fraud) return false;
      if (!query) return true;
      return `${student.student_name || ""} ${student.group_name || ""}`
        .toLocaleLowerCase("ru-RU")
        .includes(query);
    });
  }, [search, snapshot?.students, statusFilter]);

  const columns = useMemo(() => [
    {
      accessorKey: "student_name",
      header: "Студент",
      cell: ({ row, getValue }) => (
        <div className="attendance-live-student-col">
          <strong className="attendance-live-name">{getValue()}</strong>
          {row.original.is_fraud && (
            <span className="attendance-fraud-tag" title={fraudReasonLabel(row.original.fraud_reason)}>
              ⚠️ ЧИТЕР
            </span>
          )}
        </div>
      )
    },
    {
      accessorKey: "group_name",
      header: "Группа"
    },
    {
      accessorKey: "status",
      header: "Статус",
      cell: ({ row, getValue }) => {
        const isFraud = Boolean(row.original.is_fraud);
        if (isFraud) {
          return (
            <div className="attendance-status-fraud" title={fraudReasonLabel(row.original.fraud_reason)}>
              <span className="attendance-fraud-badge">
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M12 3 5 6v5c0 4.6 2.7 8.1 7 10 4.3-1.9 7-5.4 7-10V6l-7-3Z" />
                  <path d="m12 8v4" />
                  <path d="M12 16h.01" />
                </svg>
                Нарушение (антифрод)
              </span>
              <small className="attendance-fraud-reason">{fraudReasonLabel(row.original.fraud_reason)}</small>
            </div>
          );
        }
        return (
          <select
            className={`attendance-status-select is-${getValue()}`}
            value={getValue()}
            aria-label={`Статус посещаемости: ${row.original.student_name}`}
            disabled={Number(updatingStudentId) === Number(row.original.student_id)}
            onChange={(event) => handleStatusChange(row.original.student_id, event.target.value)}
          >
            {STATUS_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
        );
      }
    },
    {
      accessorKey: "marked_at",
      header: "Последнее изменение",
      cell: ({ getValue }) => formatTime(getValue())
    },
    {
      accessorKey: "marked_by",
      header: "Источник",
      cell: ({ row, getValue }) => {
        const isFraud = Boolean(row.original.is_fraud);
        return (
          <span className={`attendance-source ${isFraud ? "is-fraud" : `is-${getValue() || "empty"}`}`}>
            {sourceLabel(row.original.marked_at ? getValue() : "", isFraud)}
          </span>
        );
      }
    }
  ], [handleStatusChange, updatingStudentId]);

  const table = useReactTable({
    data: visibleStudents,
    columns,
    getCoreRowModel: getCoreRowModel()
  });

  const rosterSize = Number(snapshot?.roster_size ?? session?.roster_size ?? 0);
  const markedCount = Number(snapshot?.marked_count ?? session?.marked_count ?? 0);
  const fraudCount = Number((snapshot?.students || []).filter((s) => s.is_fraud).length);
  const lastUpdated = snapshot?.server_time ? formatTime(snapshot.server_time) : "";

  return (
    <section className="attendance-live-panel" aria-labelledby="attendance-live-title">
      <div className="attendance-live-head">
        <div>
          <div className="attendance-live-title-row">
            <span className="attendance-live-indicator" aria-hidden="true" />
            <h2 id="attendance-live-title">Отметки в реальном времени</h2>
          </div>
          <p>
            Отметились <strong>{markedCount}</strong> из <strong>{rosterSize}</strong>
            {fraudCount > 0 && (
              <span className="attendance-live-fraud-count">
                {" "}• <strong style={{ color: "#dc2626" }}>{fraudCount}</strong> заблокировано антифродом
              </span>
            )}
            {lastUpdated ? `, обновлено в ${lastUpdated}` : ""}
          </p>
        </div>

        <div className="attendance-live-filters">
          <label>
            <span>Поиск</span>
            <input
              type="search"
              value={search}
              placeholder="ФИО или группа"
              onChange={(event) => setSearch(event.target.value)}
            />
          </label>
          <label>
            <span>Показать</span>
            <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
              <option value="all">Всех</option>
              <option value="marked">Отмеченных</option>
              <option value="unmarked">Не отмеченных</option>
              {fraudCount > 0 && <option value="fraud">Нарушения ({fraudCount})</option>}
            </select>
          </label>
        </div>
      </div>

      {error && (
        <div className="attendance-live-error" role="alert">
          <span>{error}</span>
          <button type="button" onClick={() => loadRoster(false)}>Повторить</button>
        </div>
      )}

      <div className="teacher-table-wrap attendance-live-table-wrap">
        <table className="teacher-table attendance-live-table">
          <thead>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th key={header.id}>
                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {loading && !snapshot ? (
              Array.from({ length: 4 }, (_, index) => (
                <tr className="attendance-live-skeleton" key={index} aria-hidden="true">
                  {columns.map((column, cellIndex) => (
                    <td key={column.accessorKey}><span style={{ width: `${54 + cellIndex * 7}%` }} /></td>
                  ))}
                </tr>
              ))
            ) : table.getRowModel().rows.length ? (
              table.getRowModel().rows.map((row) => (
                <tr key={row.id} className={row.original.is_fraud ? "is-fraud-row" : ""}>
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</td>
                  ))}
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={columns.length} className="attendance-live-empty">
                  {snapshot?.students?.length
                    ? "По выбранному фильтру студентов нет."
                    : "В составе этого занятия пока нет студентов."}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <p className="attendance-live-note">
        Статус можно исправить прямо в таблице. Изменение сохраняется сразу (для записей с нарушением антифрода ручное изменение заблокировано).
      </p>
    </section>
  );
};

export default AttendanceLiveTable;

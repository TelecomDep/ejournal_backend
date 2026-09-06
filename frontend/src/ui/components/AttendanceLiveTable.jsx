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
  if (value === "teacher") return "Преподаватель";
  if (isFraud) return "Автопроверка";
  if (value === "self") return "Студент";
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

const needsManualReview = (student) => Boolean(student?.is_fraud) && student?.marked_by !== "teacher";

const formatCompactStudentName = (value) => {
  const parts = String(value || "").trim().split(/\s+/).filter(Boolean);
  if (parts.length < 2) return parts[0] || "Студент";
  const initials = parts.slice(1).map((part) => `${part.charAt(0).toLocaleUpperCase("ru-RU")}.`).join("");
  return `${parts[0]} ${initials}`;
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
  const markedCount = students.filter((student) => student.status !== "absent").length;
  const rosterSize = students.length;

  return {
    ...snapshot,
    students,
    marked_count: markedCount,
    roster_size: rosterSize,
    attendance_percent: rosterSize ? (markedCount * 100) / rosterSize : 0
  };
};

const mergePendingStatuses = (payload, pendingStatuses) => {
  if (!payload || pendingStatuses.size === 0) return payload;

  const students = (payload.students || []).map((student) => {
    const pending = pendingStatuses.get(Number(student.student_id));
    if (!pending) return student;
    if (pending.confirmed && student.status === pending.status) {
      pendingStatuses.delete(Number(student.student_id));
      return student;
    }
    return {
      ...student,
      status: pending.status,
      marked_at: pending.markedAt,
      marked_by: "teacher"
    };
  });
  const markedCount = students.filter((student) => student.status !== "absent").length;

  return {
    ...payload,
    students,
    marked_count: markedCount,
    roster_size: students.length,
    attendance_percent: students.length ? (markedCount * 100) / students.length : 0
  };
};

const isSameRosterSnapshot = (current, next) => {
  if (!current || !next) return false;
  if (Number(current.roster_size) !== Number(next.roster_size)) return false;
  if (Number(current.marked_count) !== Number(next.marked_count)) return false;

  const currentStudents = current.students || [];
  const nextStudents = next.students || [];
  if (currentStudents.length !== nextStudents.length) return false;

  return currentStudents.every((student, index) => {
    const candidate = nextStudents[index];
    return Number(student.student_id) === Number(candidate?.student_id)
      && student.status === candidate?.status
      && student.marked_at === candidate?.marked_at
      && student.marked_by === candidate?.marked_by
      && Boolean(student.is_fraud) === Boolean(candidate?.is_fraud)
      && student.fraud_reason === candidate?.fraud_reason;
  });
};

const AttendanceLiveTable = ({ token, session, compactStudentIdentity = false }) => {
  const lessonId = Number(session?.lesson_id || session?.session_id || session?.id || 0);
  const [snapshot, setSnapshot] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [studentSearch, setStudentSearch] = useState("");
  const [groupFilter, setGroupFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [updatingStudentId, setUpdatingStudentId] = useState(0);

  const mountedRef = useRef(true);
  const requestInFlightRef = useRef(false);
  const statusInteractionRef = useRef(false);
  const interactionSafetyTimerRef = useRef(0);
  const pendingStatusesRef = useRef(new Map());
  const statusSequenceRef = useRef(0);
  const lessonIdRef = useRef(lessonId);
  lessonIdRef.current = lessonId;

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      window.clearTimeout(interactionSafetyTimerRef.current);
    };
  }, []);

  const loadRoster = useCallback(async (silent = false) => {
    if (!lessonId || lessonIdRef.current !== lessonId) return;
    if (requestInFlightRef.current) return;
    if (silent && statusInteractionRef.current) return;
    requestInFlightRef.current = true;
    if (!silent && mountedRef.current) setLoading(true);
    try {
      const payload = await api.getAttendanceSessionRoster(token, lessonId);
      if (!mountedRef.current || lessonIdRef.current !== lessonId) return;
      if (silent && statusInteractionRef.current) return;
      const nextSnapshot = mergePendingStatuses(payload, pendingStatusesRef.current);
      setSnapshot((current) => (isSameRosterSnapshot(current, nextSnapshot) ? current : nextSnapshot));
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

  const handleStatusInteractionEnd = useCallback(() => {
    window.clearTimeout(interactionSafetyTimerRef.current);
    statusInteractionRef.current = false;
  }, []);

  const handleStatusInteractionStart = useCallback(() => {
    statusInteractionRef.current = true;
    window.clearTimeout(interactionSafetyTimerRef.current);
    interactionSafetyTimerRef.current = window.setTimeout(handleStatusInteractionEnd, 15000);
  }, [handleStatusInteractionEnd]);

  useEffect(() => {
    pendingStatusesRef.current.clear();
    statusInteractionRef.current = false;
    window.clearTimeout(interactionSafetyTimerRef.current);
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
    const sequence = statusSequenceRef.current + 1;
    statusSequenceRef.current = sequence;
    pendingStatusesRef.current.set(Number(studentId), {
      status,
      sequence,
      confirmed: false,
      markedAt: new Date().toISOString()
    });
    setUpdatingStudentId(Number(studentId));
    setError("");
    setSnapshot((current) => updateSnapshotStudent(current, studentId, status));

    try {
      await api.updateAttendanceStatus(token, lessonId, studentId, status);
      const pending = pendingStatusesRef.current.get(Number(studentId));
      if (pending?.sequence === sequence) {
        pendingStatusesRef.current.set(Number(studentId), { ...pending, confirmed: true });
      }
      await loadRoster(true);
    } catch (err) {
      if (mountedRef.current) {
        const pending = pendingStatusesRef.current.get(Number(studentId));
        if (pending?.sequence === sequence) {
          pendingStatusesRef.current.delete(Number(studentId));
        }
        setError(api.getErrorMessage(err, "Не удалось сохранить новый статус"));
        await loadRoster(true);
      }
    } finally {
      if (mountedRef.current) setUpdatingStudentId(0);
    }
  }, [lessonId, loadRoster, token]);

  const handleStatusSelectChange = useCallback((event, studentId) => {
    const select = event.currentTarget;
    const nextStatus = select.value;

    // Let the browser close its native select popup before React changes and
    // disables the control. Replacing the focused element synchronously can
    // freeze Safari's native picker, especially in fullscreen mode.
    select.blur();
    window.requestAnimationFrame(() => {
      handleStatusInteractionEnd();
      handleStatusChange(studentId, nextStatus);
    });
  }, [handleStatusChange, handleStatusInteractionEnd]);

  const availableGroups = useMemo(() => (
    Array.from(new Set(
      (snapshot?.students || [])
        .map((student) => String(student.group_name || "").trim())
        .filter(Boolean)
    )).sort((left, right) => left.localeCompare(right, "ru-RU"))
  ), [snapshot?.students]);

  const visibleStudents = useMemo(() => {
    const studentQuery = studentSearch.trim().toLocaleLowerCase("ru-RU");
    return (snapshot?.students || []).filter((student) => {
      if (statusFilter === "marked" && student.status === "absent") return false;
      if (statusFilter === "unmarked" && student.status !== "absent") return false;
      if (statusFilter === "fraud" && !needsManualReview(student)) return false;
      if (studentQuery && !(student.student_name || "").toLocaleLowerCase("ru-RU").includes(studentQuery)) return false;
      if (groupFilter !== "all" && String(student.group_name || "") !== groupFilter) return false;
      return true;
    });
  }, [groupFilter, snapshot?.students, statusFilter, studentSearch]);

  const columns = useMemo(() => [
    {
      accessorKey: "student_name",
      header: "Студент",
      cell: ({ row, getValue }) => (
        <div className="attendance-live-student-col">
          <strong className="attendance-live-name">
            {compactStudentIdentity ? formatCompactStudentName(getValue()) : getValue()}
          </strong>
          {compactStudentIdentity && (
            <small className="attendance-live-student-group">
              {row.original.group_name ? `Группа ${row.original.group_name}` : "Группа не указана"}
            </small>
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
        return (
          <div className="attendance-status-control">
            <select
              className={`attendance-status-select is-${getValue()}`}
              value={getValue()}
              aria-label={`Статус посещаемости: ${row.original.student_name}`}
              disabled={Number(updatingStudentId) === Number(row.original.student_id)}
              onFocus={handleStatusInteractionStart}
              onPointerDown={handleStatusInteractionStart}
              onBlur={handleStatusInteractionEnd}
              onChange={(event) => handleStatusSelectChange(event, row.original.student_id)}
            >
              {STATUS_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
            {isFraud && (
              <small className="attendance-fraud-reason" title={fraudReasonLabel(row.original.fraud_reason)}>
                {row.original.marked_by === "teacher"
                  ? "Статус подтверждён преподавателем"
                  : `Нужна ручная проверка: ${fraudReasonLabel(row.original.fraud_reason)}`}
              </small>
            )}
          </div>
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
          <span className={`attendance-source ${getValue() === "teacher" ? "is-teacher" : isFraud ? "is-fraud" : `is-${getValue() || "empty"}`}`}>
            {sourceLabel(row.original.marked_at ? getValue() : "", isFraud)}
          </span>
        );
      }
    }
  ], [compactStudentIdentity, handleStatusInteractionEnd, handleStatusInteractionStart, handleStatusSelectChange, updatingStudentId]);

  const table = useReactTable({
    data: visibleStudents,
    columns,
    getCoreRowModel: getCoreRowModel()
  });

  const rosterSize = Number(snapshot?.roster_size ?? session?.roster_size ?? 0);
  const markedCount = Number(snapshot?.marked_count ?? session?.marked_count ?? 0);
  const reviewCount = Number((snapshot?.students || []).filter(needsManualReview).length);
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
            {reviewCount > 0 && (
              <span className="attendance-live-fraud-count">
                {" "}• <strong>{reviewCount}</strong> требуют ручной проверки
              </span>
            )}
            {lastUpdated ? `, обновлено в ${lastUpdated}` : ""}
          </p>
        </div>

        <div className="attendance-live-filters">
          <label>
            <span>ФИО</span>
            <input
              type="search"
              value={studentSearch}
              placeholder="Найти студента"
              onChange={(event) => setStudentSearch(event.target.value)}
            />
          </label>
          <label>
            <span>Группа</span>
            <select value={groupFilter} onChange={(event) => setGroupFilter(event.target.value)}>
              <option value="all">Все группы</option>
              {availableGroups.map((groupName) => (
                <option key={groupName} value={groupName}>{groupName}</option>
              ))}
            </select>
          </label>
          <label>
            <span>Показать</span>
            <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
              <option value="all">Всех</option>
              <option value="marked">Отмеченных</option>
              <option value="unmarked">Не отмеченных</option>
              <option value="fraud">Требуют проверки ({reviewCount})</option>
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
                <tr key={row.id}>
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
        Статус можно исправить прямо в таблице. Ручное решение преподавателя имеет приоритет над автоматической проверкой.
      </p>
    </section>
  );
};

export default React.memo(AttendanceLiveTable);

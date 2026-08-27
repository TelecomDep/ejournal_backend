import axios from 'axios';

const DEFAULT_BACKEND_URL = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8888';
const BACKEND_URL = (import.meta.env.VITE_BACKEND_URL || DEFAULT_BACKEND_URL).replace(/\/$/, '');

function unwrapApiResponse(data) {
  if (data && typeof data === 'object' && Object.prototype.hasOwnProperty.call(data, 'ok')) {
    if (data.ok) {
      return data.result || {};
    }
    throw new Error(data.error || 'Backend request failed');
  }
  return data;
}

function extractError(error) {
  return error.response?.data?.error || error.message || 'Backend request failed';
}

function authHeaders(token) {
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  };
}

function resolveAssetUrl(url) {
  if (!url) {
    return '';
  }
  const avatarIdx = url.indexOf('/api/user/avatar/');
  if (avatarIdx >= 0) {
    return `${BACKEND_URL}${url.slice(avatarIdx)}`;
  }
  const uploadsIdx = url.indexOf('/uploads/');
  if (uploadsIdx >= 0) {
    return `${BACKEND_URL}${url.slice(uploadsIdx)}`;
  }
  if (/^https?:\/\//i.test(url)) {
    return url;
  }
  return `${BACKEND_URL}${url.startsWith('/') ? '' : '/'}${url}`;
}

const api = {
  // Login endpoint
  async login(login, password, twoFaCode = '') {
    try {
      const response = await axios.post(`${BACKEND_URL}/login`, {
        login,
        password,
        two_fa_code: twoFaCode
      }, {
        headers: {
          'Content-Type': 'application/json'
        }
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка входа:', error);
      throw error;
    }
  },

  // Register endpoint
  async register(login, password, registrationCode) {
    try {
      const response = await axios.post(`${BACKEND_URL}/register/by-invite`, {
        login,
        password,
        invite_code: registrationCode
      }, {
        headers: {
          'Content-Type': 'application/json'
        }
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка регистрации:', error);
      throw error;
    }
  },

  // Get profile using token
  async getProfile(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/profile`, {
        headers: authHeaders(token)
      });
      const profile = unwrapApiResponse(response.data);
      if (profile && profile.avatar) {
        profile.avatar = resolveAssetUrl(profile.avatar);
      }
      return profile;
    } catch (error) {
      console.error('Ошибка загрузки профиля:', error);
      throw error;
    }
  },

  async uploadAvatar(token, file) {
    try {
      const form = new FormData();
      form.append('avatar', file);
      const response = await axios.post(`${BACKEND_URL}/api/user/upload-avatar`, form, {
        headers: { 'Authorization': `Bearer ${token}` },
        timeout: 20000
      });
      return resolveAssetUrl(unwrapApiResponse(response.data));
    } catch (error) {
      console.error('Ошибка загрузки фото:', error);
      throw error;
    }
  },

  async getStudentScheduleDay(token, date) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/student/schedule/day`, {
        headers: authHeaders(token),
        params: date ? { date } : {}
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки расписания:', error);
      throw error;
    }
  },

  async getTeacherScheduleDay(token, date) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/teacher/schedule/day`, {
        headers: authHeaders(token),
        params: date ? { date } : {}
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки расписания преподавателя:', error);
      throw error;
    }
  },

  // Student: Get students list
  async getStudentsList(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/students`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки списка студентов:', error);
      throw error;
    }
  },

  // Student: Get attendance table
  async getAttendanceTable(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/attendance/table`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки таблицы посещаемости:', error);
      throw error;
    }
  },

  // Teacher: Create attendance link
  async createAttendanceLink(token, subjectId, groupIds, lessonName, expiresMinutes) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/attendance-link`,
        {
          subject_id: subjectId,
          group_ids: groupIds,
          lesson_name: lessonName,
          expires_minutes: expiresMinutes
        },
        {
          headers: authHeaders(token)
        }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка создания ссылки посещаемости:', error);
      throw error;
    }
  },

  async getTeacherActiveAttendanceSession(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/teacher/attendance/session/active`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки активного занятия преподавателя:', error);
      throw error;
    }
  },

  async getAttendanceSessionRoster(token, lessonId) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/teacher/attendance/session/roster`, {
        headers: authHeaders(token),
        params: { lesson_id: Number(lessonId) }
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки списка посещаемости:', error);
      throw error;
    }
  },

  async updateAttendanceStatus(token, lessonId, studentId, status) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/attendance/mark`,
        {
          lesson_id: Number(lessonId),
          student_id: Number(studentId),
          status
        },
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка исправления посещаемости:', error);
      throw error;
    }
  },

  // Teacher: Get assigned subjects and groups
  async getTeacherSubjects(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/teacher/subjects`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки предметов преподавателя:', error);
      throw error;
    }
  },

  // Student: Confirm attendance
  async confirmAttendance(token, inviteToken) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/student/attendance/confirm`,
        {
          invite_token: inviteToken
        },
        {
          headers: authHeaders(token)
        }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка подтверждения посещаемости:', error);
      throw error;
    }
  },

  // Teacher: Get group statistics
  async getGroupStats(token, groupId, subjectId = null) {
    try {
      const payload = {
        group_id: groupId
      };

      if (subjectId) {
        payload.subject_id = subjectId;
      }

      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/attendance/group`,
        payload,
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки статистики группы:', error);
      throw error;
    }
  },

  // Teacher: Get combined group performance (attendance + grades) for a subject
  async getGroupPerformance(token, groupId, subjectId) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/group/performance`,
        {
          group_id: groupId,
          subject_id: subjectId
        },
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки сводки по группе:', error);
      throw error;
    }
  },

  // Student: Get attendance history for the heatmap
  async getStudentAttendanceHeatmap(token, year) {
    try {
      const response = await axios.get(
        `${BACKEND_URL}/api/student/attendance/history`,
        {
          headers: authHeaders(token),
          params: { year }
        }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки истории посещаемости:', error);
      throw error;
    }
  },

  async createGradeItem(token, payload) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/grades/items`,
        payload,
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка создания контрольной точки:', error);
      throw error;
    }
  },

  async deleteGradeItem(token, payload) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/grades/items/delete`,
        payload,
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка удаления контрольной точки:', error);
      throw error;
    }
  },

  async getTeacherGradeItems(token, subjectId) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/grades/items/list`,
        { subject_id: subjectId },
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки контрольных точек:', error);
      throw error;
    }
  },

  async saveStudentGrade(token, payload) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/grades`,
        payload,
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка сохранения оценки:', error);
      throw error;
    }
  },

  async getTeacherStudentGrades(token, studentId, subjectId) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/grades/student`,
        {
          student_id: studentId,
          subject_id: subjectId
        },
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки ведомости студента:', error);
      throw error;
    }
  },

  async getStudentGrades(token, subjectId) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/student/grades`,
        { subject_id: subjectId },
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки оценок студента:', error);
      throw error;
    }
  },

  async getStudentPerformanceRadar(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/student/performance/radar`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки диаграммы успеваемости:', error);
      throw error;
    }
  },

  async getTeacherStudentPerformanceRadar(token, studentId) {
    try {
      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/student/performance/radar`,
        { student_id: studentId },
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки диаграммы успеваемости студента:', error);
      throw error;
    }
  },

  // Password Recovery / Reset
  async forgotPassword(identity) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/auth/forgot-password`, {
        identity
      }, {
        headers: { 'Content-Type': 'application/json' }
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка запроса восстановления пароля:', error);
      throw error;
    }
  },

  async resetPassword(token, newPassword) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/auth/reset-password`, {
        token,
        new_password: newPassword
      }, {
        headers: { 'Content-Type': 'application/json' }
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка сброса пароля:', error);
      throw error;
    }
  },

  async updateEmail(token, email) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/user/email`, {
        email
      }, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка обновления email:', error);
      throw error;
    }
  },

  async requestEmailBind(token, email) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/user/email/bind/request`, { email }, { headers: authHeaders(token) });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка запроса привязки email:', error);
      throw error;
    }
  },

  async confirmEmailBind(token, code) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/user/email/bind/confirm`, { code }, { headers: authHeaders(token) });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка подтверждения email:', error);
      throw error;
    }
  },

  async request2FAEnable(token) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/user/2fa/request-enable`, {}, { headers: authHeaders(token) });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка запроса подключения 2FA:', error);
      throw error;
    }
  },

  async generate2FA(token, emailCode) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/user/2fa/generate`, { email_code: emailCode }, { headers: authHeaders(token) });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка генерации 2FA:', error);
      throw error;
    }
  },

  async verify2FA(token, code) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/user/2fa/verify`, { code }, { headers: authHeaders(token) });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка подтверждения 2FA:', error);
      throw error;
    }
  },

  async disable2FA(token, code) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/user/2fa/disable`, { code }, { headers: authHeaders(token) });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка отключения 2FA:', error);
      throw error;
    }
  },

  async getCurrentAgreement(token) {
    const response = await axios.get(`${BACKEND_URL}/api/user/agreements/current`, { headers: authHeaders(token) });
    return unwrapApiResponse(response.data);
  },

  async recordAgreementDecision(token, decision, version) {
    const response = await axios.post(
      `${BACKEND_URL}/api/user/agreements/decision`,
      { decision, version },
      { headers: authHeaders(token) }
    );
    return unwrapApiResponse(response.data);
  },

  // Student: all subjects with their grade items and totals in one request
  async getStudentAllGrades(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/student/grades/all`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки оценок:', error);
      throw error;
    }
  },

  async getAdminUsers(token, filters = {}) {
    try {
      const params = Object.fromEntries(
        Object.entries(filters).filter(([, value]) => value !== '' && value !== undefined && value !== null)
      );
      const response = await axios.get(`${BACKEND_URL}/api/admin/users`, {
        headers: authHeaders(token),
        params
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки пользователей:', error);
      throw error;
    }
  },

  async getAdminUser(token, userId) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/admin/users/${userId}`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки пользователя:', error);
      throw error;
    }
  },

  async createAdminUser(token, payload) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/admin/users`, payload, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка создания пользователя:', error);
      throw error;
    }
  },

  async updateAdminUser(token, userId, payload) {
    try {
      const response = await axios.patch(`${BACKEND_URL}/api/admin/users/${userId}`, payload, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка обновления пользователя:', error);
      throw error;
    }
  },

  async archiveAdminUser(token, userId) {
    try {
      const response = await axios.delete(`${BACKEND_URL}/api/admin/users/${userId}`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка архивирования пользователя:', error);
      throw error;
    }
  },

  async getAdminInvites(token, filters = {}) {
    try {
      const params = new URLSearchParams();
      if (filters.role) params.append('role', filters.role);
      if (filters.status) params.append('status', filters.status);
      const query = params.toString() ? `?${params.toString()}` : '';

      const response = await axios.get(`${BACKEND_URL}/api/admin/invites${query}`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки списка инвайтов:', error);
      throw error;
    }
  },

  async createTeacherInvite(token, payload) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/admin/invites/teacher`, payload, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка генерации инвайта преподавателя:', error);
      throw error;
    }
  },

  async createStudentInvite(token, payload) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/admin/invites/student`, payload, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка генерации инвайта студента:', error);
      throw error;
    }
  },

  async revokeAdminInvite(token, inviteId) {
    try {
      const response = await axios.delete(`${BACKEND_URL}/api/admin/invites/${inviteId}`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка отзыва инвайта:', error);
      throw error;
    }
  },

  async getAdminCatalogs(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/admin/catalogs`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки справочников:', error);
      return { groups: [], lecterns: [], faculties: [], teachers: [], students: [] };
    }
  },

  async getSemesters(token) {
    try {
      const config = token ? { headers: authHeaders(token) } : {};
      const response = await axios.get(`${BACKEND_URL}/api/semesters`, config);
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки семестров:', error);
      throw error;
    }
  },

  async getStaffGeneralRating(token, semesterId) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/staff/ratings/general`, {
        headers: authHeaders(token),
        params: {
          ...(semesterId ? { semester_id: semesterId } : {}),
          page: 1,
          page_size: 50
        }
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки параметров отчёта:', error);
      throw error;
    }
  },

  async getStaffAnalytics(token, filters = {}) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/staff/analytics`, {
        headers: authHeaders(token),
        params: {
          ...(filters.semesterId ? { semester_id: filters.semesterId } : {}),
          ...(filters.scopeType ? { scope_type: filters.scopeType } : {}),
          ...(filters.scopeId ? { scope_id: filters.scopeId } : {}),
          ...(filters.subjectId ? { subject_id: filters.subjectId } : {})
        }
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки аналитики для сотрудников:', error);
      throw error;
    }
  },

  async getStudentGroupRating(token, semesterId) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/student/ratings/group`, {
        headers: authHeaders(token),
        params: {
          ...(semesterId ? { semester_id: semesterId } : {}),
          page: 1,
          page_size: 1
        }
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки рейтинга группы:', error);
      throw error;
    }
  },

  async downloadStaffPerformanceReport(token, format, semesterId = '', filters = {}) {
    const normalizedFormat = format === 'pdf' ? 'pdf' : 'xlsx';
    const fallbackName = `performance-report.${normalizedFormat}`;

    try {
      const response = await axios.get(
        `${BACKEND_URL}/api/staff/reports/performance.${normalizedFormat}`,
        {
          headers: { 'Authorization': `Bearer ${token}` },
          params: {
            ...(semesterId ? { semester_id: semesterId } : {}),
            ...(filters.departmentId ? { department_id: filters.departmentId } : {}),
            ...(filters.subjectId ? { subject_id: filters.subjectId } : {}),
            ...(filters.groupIds?.length ? { group_ids: filters.groupIds.join(',') } : {})
          },
          responseType: 'blob'
        }
      );
      const disposition = response.headers['content-disposition'] || '';
      const encodedMatch = disposition.match(/filename\*=UTF-8''([^;]+)/i);
      const plainMatch = disposition.match(/filename="?([^";]+)"?/i);
      let filename = fallbackName;

      if (encodedMatch?.[1]) {
        try {
          filename = decodeURIComponent(encodedMatch[1]);
        } catch {
          filename = encodedMatch[1];
        }
      } else if (plainMatch?.[1]) {
        filename = plainMatch[1];
      }

      return { blob: response.data, filename };
    } catch (error) {
      if (error.response?.data instanceof Blob) {
        try {
          error.response.data = JSON.parse(await error.response.data.text());
        } catch {
          // Keep the original response when the backend did not return JSON.
        }
      }
      console.error('Ошибка формирования отчёта:', error);
      throw error;
    }
  },

  // Supervisory overview (teacher/head/dean/admin), scoped by role on the backend
  async getStaffOverview(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/staff/overview`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки сводки:', error);
      throw error;
    }
  },

  async getAntifraudLogs(token, filters = {}) {
    try {
      const params = Object.fromEntries(
        Object.entries(filters).filter(([, value]) => value !== '' && value !== undefined && value !== null)
      );
      const response = await axios.get(`${BACKEND_URL}/api/staff/antifraud/logs`, {
        headers: authHeaders(token),
        params
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки журнала антифрода:', error);
      throw error;
    }
  },

  async getAntifraudTopCheaters(token, filters = {}) {
    try {
      const params = Object.fromEntries(
        Object.entries(filters).filter(([, value]) => value !== '' && value !== undefined && value !== null)
      );
      const response = await axios.get(`${BACKEND_URL}/api/staff/antifraud/top-cheaters`, {
        headers: authHeaders(token),
        params
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки рейтинга нарушителей:', error);
      throw error;
    }
  },

  async getNotifications(token, filters = {}) {
    try {
      const params = Object.fromEntries(
        Object.entries(filters).filter(([, value]) => value !== '' && value !== undefined && value !== null)
      );
      const response = await axios.get(`${BACKEND_URL}/api/user/notifications`, {
        headers: authHeaders(token),
        params
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки уведомлений:', error);
      throw error;
    }
  },

  async getUnreadNotificationsCount(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/user/notifications/unread-count`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки количества уведомлений:', error);
      throw error;
    }
  },

  async markNotificationRead(token, notificationId) {
    try {
      const response = await axios.patch(
        `${BACKEND_URL}/api/user/notifications/${notificationId}/read`,
        {},
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка отметки уведомления прочитанным:', error);
      throw error;
    }
  },

  async markAllNotificationsRead(token) {
    try {
      const response = await axios.patch(
        `${BACKEND_URL}/api/user/notifications/read-all`,
        {},
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка отметки всех уведомлений прочитанными:', error);
      throw error;
    }
  },

  async getNotificationSettings(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/user/notification-settings`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки настроек уведомлений:', error);
      throw error;
    }
  },

  async updateNotificationSettings(token, settings) {
    try {
      const response = await axios.put(
        `${BACKEND_URL}/api/user/notification-settings`,
        settings,
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка сохранения настроек уведомлений:', error);
      throw error;
    }
  },

  async getAdminNotifications(token, filters = {}) {
    try {
      const params = Object.fromEntries(
        Object.entries(filters).filter(([, value]) => value !== '' && value !== undefined && value !== null)
      );
      const response = await axios.get(`${BACKEND_URL}/api/admin/notifications`, {
        headers: authHeaders(token),
        params
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки административных уведомлений:', error);
      throw error;
    }
  },

  async createAdminNotification(token, payload) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/admin/notifications`, payload, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка создания уведомления:', error);
      throw error;
    }
  },

  async updateAdminNotification(token, notificationId, payload) {
    try {
      const response = await axios.patch(
        `${BACKEND_URL}/api/admin/notifications/${notificationId}`,
        payload,
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка обновления уведомления:', error);
      throw error;
    }
  },

  async deleteAdminNotification(token, notificationId) {
    try {
      const response = await axios.delete(`${BACKEND_URL}/api/admin/notifications/${notificationId}`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка удаления уведомления:', error);
      throw error;
    }
  },

  async finishAttendanceSession(token, sessionId) {
    try {
      const numId = sessionId ? Number(sessionId) : undefined;
      const response = await axios.post(
        `${BACKEND_URL}/api/teacher/attendance/session/finish`,
        { session_id: numId, lesson_id: numId },
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error("Ошибка завершения занятия:", error);
      throw error;
    }
  },

  async getStudentActiveSession(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/student/attendance/active-session`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error("Ошибка загрузки активного занятия студента:", error);
      throw error;
    }
  },

  // Semesters management endpoints
  async getCurrentSemester(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/semesters/current`, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error("Ошибка загрузки текущего семестра:", error);
      throw error;
    }
  },

  async createAdminSemester(token, payload) {
    try {
      const response = await axios.post(`${BACKEND_URL}/api/admin/semesters`, payload, {
        headers: authHeaders(token)
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error("Ошибка создания семестра:", error);
      throw error;
    }
  },

  async activateAdminSemester(token, semesterId) {
    try {
      const response = await axios.patch(
        `${BACKEND_URL}/api/admin/semesters/${semesterId}/activate`,
        {},
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error("Ошибка активации семестра:", error);
      throw error;
    }
  },

  async closeAdminSemester(token, semesterId) {
    try {
      const response = await axios.patch(
        `${BACKEND_URL}/api/admin/semesters/${semesterId}/close`,
        {},
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error("Ошибка закрытия семестра:", error);
      throw error;
    }
  },

  async archiveAdminSemester(token, semesterId) {
    try {
      const response = await axios.patch(
        `${BACKEND_URL}/api/admin/semesters/${semesterId}/archive`,
        {},
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error("Ошибка архивации семестра:", error);
      throw error;
    }
  },

  async deleteAdminSemester(token, semesterId) {
    try {
      const response = await axios.delete(
        `${BACKEND_URL}/api/admin/semesters/${semesterId}`,
        { headers: authHeaders(token) }
      );
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error("Ошибка удаления семестра:", error);
      throw error;
    }
  },

  getErrorMessage(error, fallback) {
    return extractError(error) || fallback;
  }
};

export default api;

import axios from 'axios';

const DEFAULT_BACKEND_URL = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8888';
const BACKEND_URL = (process.env.REACT_APP_BACKEND_URL || DEFAULT_BACKEND_URL).replace(/\/$/, '');

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
  async login(login, password) {
    try {
      const response = await axios.post(`${BACKEND_URL}/login`, {
        login,
        password
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
        headers: { 'Authorization': `Bearer ${token}` }
      });
      return resolveAssetUrl(unwrapApiResponse(response.data));
    } catch (error) {
      console.error('Ошибка загрузки фото:', error);
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

  getErrorMessage(error, fallback) {
    return extractError(error) || fallback;
  }
};

export default api;

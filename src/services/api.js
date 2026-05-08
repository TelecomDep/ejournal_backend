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
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Ошибка загрузки профиля:', error);
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

  getErrorMessage(error, fallback) {
    return extractError(error) || fallback;
  }
};

export default api;

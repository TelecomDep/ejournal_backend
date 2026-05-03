import axios from 'axios';

const DEFAULT_BACKEND_URL = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8888';
const BACKEND_URL = (process.env.REACT_APP_BACKEND_URL || DEFAULT_BACKEND_URL).replace(/\/$/, '');

function unwrapApiResponse(data) {
  if (data && typeof data === 'object' && Object.prototype.hasOwnProperty.call(data, 'ok')) {
    if (data.ok) {
      return data.result || {};
    }

    const error = new Error(data.error || 'Backend request failed');
    error.backend = data;
    throw error;
  }

  return data;
}

function extractError(error) {
  return error.response?.data?.error || error.backend?.error || error.message || 'Backend request failed';
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
      console.error('Login Error:', error);
      throw error;
    }
  },

  // Register endpoint
  async register(login, password, roleHash, inviteCode) {
    try {
      const endpoint = inviteCode ? '/register/by-invite' : '/register';
      const body = inviteCode
        ? { login, password, invite_code: inviteCode }
        : { login, password, role_hash: roleHash };

      const response = await axios.post(`${BACKEND_URL}${endpoint}`, body, {
        headers: {
          'Content-Type': 'application/json'
        }
      });
      return unwrapApiResponse(response.data);
    } catch (error) {
      console.error('Register Error:', error);
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
      console.error('Get Profile Error:', error);
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
      console.error('Create Attendance Link Error:', error);
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
      console.error('Confirm Attendance Error:', error);
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
      console.error('Get Group Stats Error:', error);
      throw error;
    }
  },

  getErrorMessage(error, fallback) {
    return extractError(error) || fallback;
  }
};

export default api;

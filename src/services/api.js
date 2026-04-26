import axios from 'axios';

const BACKEND_URL = process.env.REACT_APP_BACKEND_URL || 'http://localhost:8888';

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
      return response.data;
    } catch (error) {
      console.error('Login Error:', error);
      throw error;
    }
  },

  // Register endpoint
  async register(login, password, role, inviteCode) {
    try {
      const endpoint = (role === 'student' && inviteCode) ? '/register/by-invite' : '/register';
      const body = role === 'student' && inviteCode
        ? { login, password, invite_code: inviteCode }
        : { login, password, role };

      const response = await axios.post(`${BACKEND_URL}${endpoint}`, body, {
        headers: {
          'Content-Type': 'application/json'
        }
      });
      return response.data;
    } catch (error) {
      console.error('Register Error:', error);
      throw error;
    }
  },

  // Get profile using token
  async getProfile(token) {
    try {
      const response = await axios.get(`${BACKEND_URL}/profile`, {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        }
      });
      return response.data;
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
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
          }
        }
      );
      return response.data;
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
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
          }
        }
      );
      return response.data;
    } catch (error) {
      console.error('Confirm Attendance Error:', error);
      throw error;
    }
  },

  // Teacher: Get group statistics
  async getGroupStats(token, groupId, subjectId = null) {
    try {
      const params = new URLSearchParams();
      params.append('group_id', groupId);
      if (subjectId) {
        params.append('subject_id', subjectId);
      }

      const response = await axios.get(
        `${BACKEND_URL}/api/teacher/attendance-stats?${params.toString()}`,
        {
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
          }
        }
      );
      return response.data;
    } catch (error) {
      console.error('Get Group Stats Error:', error);
      throw error;
    }
  }
};

export default api;

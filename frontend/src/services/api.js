import axios from 'axios';

const BACKEND_URL = process.env.REACT_APP_BACKEND_URL || 'http://localhost:9999';

function unwrapApiResponse(data) {
  if (data && typeof data === 'object' && Object.prototype.hasOwnProperty.call(data, 'ok')) {
    if (data.ok) {
      return data.result || {};
    }
    const err = new Error(data.error || 'Backend request failed');
    err.backend = data;
    throw err;
  }
  return data;
}

function authHeaders(token) {
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`
  };
}

const api = {
  async login(login, passwordHash) {
    const response = await axios.post(
      `${BACKEND_URL}/login`,
      { login, password: passwordHash },
      { headers: { 'Content-Type': 'application/json' } }
    );
    return unwrapApiResponse(response.data);
  },

  async register(login, passwordHash, roleHash) {
    const response = await axios.post(`${BACKEND_URL}/register`, {
      login,
      password: passwordHash,
      role_hash: roleHash
    }, {
      headers: { 'Content-Type': 'application/json' }
    });

    return unwrapApiResponse(response.data);
  },

  async getProfile(token) {
    const response = await axios.get(`${BACKEND_URL}/profile`, {
      headers: authHeaders(token)
    });
    return unwrapApiResponse(response.data);
  },

  async createAttendanceLink(token, payload) {
    const response = await axios.post(`${BACKEND_URL}/api/teacher/attendance-link`, payload, {
      headers: authHeaders(token)
    });
    return unwrapApiResponse(response.data);
  },

  async getAttendanceByGroup(token, payload) {
    const response = await axios.post(`${BACKEND_URL}/api/teacher/attendance/group`, payload, {
      headers: authHeaders(token)
    });
    return unwrapApiResponse(response.data);
  },

  async confirmAttendance(token, inviteToken) {
    const response = await axios.post(
      `${BACKEND_URL}/api/student/attendance/confirm`,
      { invite_token: inviteToken },
      { headers: authHeaders(token) }
    );
    return unwrapApiResponse(response.data);
  }
};

export default api;

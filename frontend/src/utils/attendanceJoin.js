/**
 * @param {{ hash?: string, pathname?: string, search?: string }} [location]
 */
export const getJoinInviteToken = (location = window.location) => {
  const hash = location.hash || '';
  if (hash.startsWith('#/attendance/join')) {
    const queryStart = hash.indexOf('?');
    if (queryStart !== -1) {
      const params = new URLSearchParams(hash.slice(queryStart + 1));
      return params.get('token') || '';
    }
  }

  if (location.pathname === '/attendance/join') {
    const params = new URLSearchParams(location.search);
    return params.get('token') || '';
  }

  return '';
};

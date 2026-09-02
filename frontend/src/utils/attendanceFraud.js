const DEVICE_STORAGE_KEY = 'ejournal_attendance_device_id';

const randomDeviceId = () => {
  try {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
      return `web-${crypto.randomUUID()}`;
    }
  } catch {
    // Fall through to the compatibility-safe random value below.
  }

  return `web-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`;
};

export const getBrowserDeviceId = () => {
  if (typeof window === 'undefined') return randomDeviceId();

  try {
    const stored = window.localStorage.getItem(DEVICE_STORAGE_KEY);
    if (stored && stored.trim()) return stored.trim();

    const generated = randomDeviceId();
    window.localStorage.setItem(DEVICE_STORAGE_KEY, generated);
    return generated;
  } catch {
    // Private browsing or blocked storage should not prevent attendance.
    return randomDeviceId();
  }
};

export const getBrowserLocation = (options = {}) => new Promise((resolve) => {
  if (typeof navigator === 'undefined' || !navigator.geolocation) {
    resolve(null);
    return;
  }

  navigator.geolocation.getCurrentPosition(
    ({ coords }) => {
      const lat = Number(coords?.latitude);
      const lon = Number(coords?.longitude);
      if (Number.isFinite(lat) && Number.isFinite(lon) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180) {
        resolve({ lat, lon });
      } else {
        resolve(null);
      }
    },
    () => resolve(null),
    {
      enableHighAccuracy: false,
      timeout: 3000,
      maximumAge: 30000,
      ...options
    }
  );
});

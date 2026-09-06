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

const getGeolocationError = (error) => {
  switch (error?.code) {
    case 1:
      return new Error(
        'Доступ к геолокации запрещён. Разрешите этому сайту доступ к местоположению в настройках браузера. На iPhone также проверьте службы геолокации для Safari в настройках устройства.'
      );
    case 2:
      return new Error('Не удалось определить местоположение. Проверьте, что геолокация включена, и попробуйте ещё раз.');
    case 3:
      return new Error('Определение местоположения заняло слишком много времени. Подойдите к окну или выйдите на открытое место и повторите попытку.');
    default:
      return new Error('Не удалось получить геолокацию. Проверьте настройки браузера и попробуйте ещё раз.');
  }
};

export const getBrowserLocation = (options = {}) => new Promise((resolve, reject) => {
  if (typeof window !== 'undefined' && window.isSecureContext === false) {
    reject(new Error('Геолокация доступна только при защищённом подключении HTTPS.'));
    return;
  }

  if (typeof navigator === 'undefined' || !navigator.geolocation) {
    reject(new Error('Этот браузер не поддерживает определение геолокации.'));
    return;
  }

  navigator.geolocation.getCurrentPosition(
    ({ coords }) => {
      const lat = coords?.latitude;
      const lon = coords?.longitude;
      if (Number.isFinite(lat) && Number.isFinite(lon) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180) {
        resolve({ lat, lon });
      } else {
        reject(new Error('Браузер вернул некорректные координаты. Попробуйте ещё раз.'));
      }
    },
    (error) => reject(getGeolocationError(error)),
    {
      enableHighAccuracy: true,
      timeout: 15000,
      maximumAge: 0,
      ...options
    }
  );
});

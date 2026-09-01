/**
 * Runtime application configuration helper.
 * Reads configuration injected at container startup into window.__APP_CONFIG__,
 * with fallback to Vite environment variables and sensible defaults.
 */

export const getAppConfig = () => {
  const runtime = (typeof window !== 'undefined' && window.__APP_CONFIG__) || {};

  const currentHost = typeof window !== 'undefined' ? window.location.hostname : 'ezachetka.ru';
  const domain = runtime.DOMAIN || import.meta.env.VITE_DOMAIN || currentHost || 'ezachetka.ru';
  const supportEmail = runtime.SUPPORT_EMAIL || import.meta.env.VITE_SUPPORT_EMAIL || `support@${domain}`;
  const abuseEmail = runtime.ABUSE_EMAIL || import.meta.env.VITE_ABUSE_EMAIL || supportEmail;
  const feedbackTelegram = runtime.FEEDBACK_TELEGRAM || import.meta.env.VITE_FEEDBACK_TELEGRAM || '';
  const organizationName = runtime.ORGANIZATION_NAME || import.meta.env.VITE_ORGANIZATION_NAME || 'Электронный журнал ezachetka.ru';
  const disputeDays = Number(runtime.DISPUTE_TIMEFRAME_DAYS || import.meta.env.VITE_DISPUTE_TIMEFRAME_DAYS || 7);

  return {
    domain,
    supportEmail,
    abuseEmail,
    feedbackTelegram,
    organizationName,
    disputeTimeframeDays: disputeDays > 0 ? disputeDays : 7,
  };
};

export default getAppConfig;

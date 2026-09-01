import React, { useState, useEffect } from 'react';
import './CookieBanner.css';

const COOKIE_STORAGE_KEY = 'ezachetka_cookie_consent_v1';

const CookieBanner = ({ onOpenLegal }) => {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    try {
      const consent = localStorage.getItem(COOKIE_STORAGE_KEY);
      if (!consent) {
        // Small timeout for smooth entry animation after page load
        const timer = setTimeout(() => setIsVisible(true), 600);
        return () => clearTimeout(timer);
      }
    } catch {
      // In case localStorage is blocked in private browsing
      setIsVisible(true);
    }
  }, []);

  const handleAccept = () => {
    try {
      localStorage.setItem(COOKIE_STORAGE_KEY, 'accepted');
    } catch {
      // ignore
    }
    setIsVisible(false);
  };

  if (!isVisible) return null;

  return (
    <div className="cookie-banner-wrap" role="region" aria-label="Уведомление об использовании cookies">
      <div className="cookie-banner">
        <div className="cookie-banner-content">
          <span className="cookie-banner-icon" aria-hidden="true">🍪</span>
          <p className="cookie-banner-text">
            Мы используем файлы cookie и локальное хранилище для аутентификации и корректной работы сервиса.
            Продолжая использовать сайт, вы соглашаетесь с{' '}
            <button
              type="button"
              className="cookie-banner-link"
              onClick={() => onOpenLegal && onOpenLegal('privacy')}
            >
              Политикой обработки персональных данных (152-ФЗ)
            </button>.
          </p>
        </div>
        <div className="cookie-banner-actions">
          <button
            type="button"
            className="cookie-banner-btn"
            onClick={handleAccept}
          >
            Принять и продолжить
          </button>
        </div>
      </div>
    </div>
  );
};

export default CookieBanner;
